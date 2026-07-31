<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import { updateMobileSessionTitle } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import "@xterm/xterm/css/xterm.css"

const props = withDefaults(
	defineProps<{
		sessionId: number
		taskName: string
		projectName: string
		mode?: "ai" | "native"
	}>(),
	{ mode: "ai" }
)
const emit = defineEmits<{ back: []; openFiles: []; openStatus: []; renamed: [] }>()
const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const terminalElement = ref<HTMLElement | null>(null)
const commandInput = ref<HTMLInputElement | null>(null)
const commandDraft = ref("")
const commandComposing = ref(false)
const connected = ref(false)
const connecting = ref(true)
const reconnecting = ref(false)
const hasControl = ref(false)
const ctrlActive = ref(false)
const showRenameModal = ref(false)
const renameTitle = ref("")
const renameLoading = ref(false)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let socket: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let pingTimer: ReturnType<typeof setInterval> | null = null
let fitFrame: number | null = null
let reportedCols = 0
let reportedRows = 0
let lastSequence = 0
let resyncRequest = 0
let pendingResyncId = ""
let closing = false

function sendAck(sequence: number) {
	if (socket?.readyState === WebSocket.OPEN && sequence > 0) {
		socket.send(JSON.stringify({ type: "ack", data: String(sequence) }))
	}
}

function requestResync() {
	if (socket?.readyState !== WebSocket.OPEN || pendingResyncId) return
	if (props.mode === "native") {
		pendingResyncId = "native"
		socket.send(JSON.stringify({ type: "resync", data: "" }))
		return
	}
	pendingResyncId = `${Date.now()}-${++resyncRequest}`
	socket.send(JSON.stringify({
		type: "resync",
		data: JSON.stringify({ sequence: lastSequence, requestId: pendingResyncId })
	}))
}

function fit() {
	try {
		fitAddon?.fit()
		if (socket?.readyState === WebSocket.OPEN && hasControl.value && terminal && (terminal.cols !== reportedCols || terminal.rows !== reportedRows)) {
			socket.send(JSON.stringify({
				type: "resize",
				data: JSON.stringify({ cols: terminal.cols, rows: terminal.rows }),
			}))
			reportedCols = terminal.cols
			reportedRows = terminal.rows
		}
	} catch (error) {
		void error
	}
}

function updateControl(value: boolean) {
	hasControl.value = value
	if (!value) ctrlActive.value = false
	if (terminal) terminal.options.disableStdin = !value
	if (value) nextTick(() => {
		terminal?.focus()
		scheduleFit()
	})
}

function sendTerminalInput(data: string) {
	if (!hasControl.value || socket?.readyState !== WebSocket.OPEN) return
	socket.send(JSON.stringify({ type: "cmd", data }))
}

function writeTerminalData(data: string, forceBottom = false) {
	if (!terminal) return
	const buffer = terminal.buffer.active
	const shouldFollow = forceBottom || buffer.baseY - buffer.viewportY <= 1
	terminal.write(data, () => {
		if (shouldFollow) terminal?.scrollToBottom()
	})
}

function applyCtrlModifier(data: string) {
	if (!ctrlActive.value) return data
	ctrlActive.value = false
	if (data.length !== 1) return data
	if (data === "?") return "\x7f"
	const code = data.toUpperCase().charCodeAt(0)
	return code >= 64 && code <= 95 ? String.fromCharCode(code & 31) : data
}

function sendShortcut(data: string) {
	ctrlActive.value = false
	sendTerminalInput(data)
	nextTick(() => terminal?.focus())
}

function insertCommandSymbol(symbol: string) {
	if (!hasControl.value) return
	const input = commandInput.value
	const start = input?.selectionStart ?? commandDraft.value.length
	const end = input?.selectionEnd ?? start
	commandDraft.value = `${commandDraft.value.slice(0, start)}${symbol}${commandDraft.value.slice(end)}`
	nextTick(() => {
		commandInput.value?.focus()
		commandInput.value?.setSelectionRange(start + symbol.length, start + symbol.length)
	})
}

function submitCommand() {
	if (commandComposing.value || !commandDraft.value || !hasControl.value) return
	sendTerminalInput(`${commandDraft.value}\r`)
	commandDraft.value = ""
	nextTick(() => commandInput.value?.focus())
}

function toggleCtrl() {
	if (!hasControl.value) return
	ctrlActive.value = !ctrlActive.value
	nextTick(() => terminal?.focus())
}

function scheduleFit() {
	if (fitFrame !== null) return
	fitFrame = window.requestAnimationFrame(() => {
		fitFrame = null
		fit()
	})
}

function connect() {
	if (!terminal || !props.sessionId) return
	connecting.value = true
	const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
	let url =
		props.mode === "native"
			? `${protocol}//${window.location.host}/api/mobile/app/project-terminal/${props.sessionId}/ws`
			: `${protocol}//${window.location.host}/api/mobile/app/terminal?session_id=${props.sessionId}`
	url += `${url.includes("?") ? "&" : "?"}cols=${terminal.cols}&rows=${terminal.rows}`
	if (props.mode === "ai") url += "&read_only=1"
	if (lastSequence) url += `&after_sequence=${lastSequence}`
	const currentSocket = new WebSocket(url)
	socket = currentSocket
	currentSocket.onopen = () => {
		if (socket !== currentSocket) return
		connected.value = true
		connecting.value = false
		reconnecting.value = false
	}
	currentSocket.onmessage = event => {
		if (socket !== currentSocket) return
		try {
			const message = JSON.parse(event.data)
			if (message.type === "baseline") {
				if (props.mode === "ai" && pendingResyncId && message.requestId !== pendingResyncId) return
				const sequence = Number(message.sequence) || 0
				const chunkIndex = Number(message.chunkIndex) || 0
				const chunkCount = Number(message.chunkCount) || 1
				if ((props.mode === "native" || message.truncated) && chunkIndex === 0) terminal?.reset()
				if (message.data) writeTerminalData(message.data, chunkIndex === chunkCount - 1)
				else if (chunkIndex === chunkCount - 1) terminal?.scrollToBottom()
				if (chunkIndex === chunkCount - 1) {
					lastSequence = sequence
					pendingResyncId = ""
					updateControl(Boolean(message.hasControl))
					sendAck(sequence)
				}
			} else if (message.type === "output") {
				const sequence = Number(message.sequence) || 0
				if (pendingResyncId) return
				if (lastSequence > 0 && sequence !== lastSequence + 1) {
					requestResync()
					return
				}
				if (message.data) writeTerminalData(message.data)
				lastSequence = sequence
				sendAck(sequence)
			} else if (message.type === "resync_required") {
				requestResync()
			} else if (message.type === "control") {
				updateControl(Boolean(message.hasControl))
				if (message.controlReason || message.data) terminal?.writeln(`\r\n\x1b[33m[GoPanel] ${t("mobile.terminalControlBusy")}\x1b[0m`)
			} else if (message.type === "closed") {
				closing = true
				connected.value = false
				connecting.value = false
				updateControl(false)
			} else if (message.type === "error") {
				closing = true
				connected.value = false
				connecting.value = false
				updateControl(false)
				if (message.data) terminal?.writeln(`\r\n\x1b[31m[GoPanel] ${message.data}\x1b[0m`)
			} else if (message.type === "cmd" && message.data) {
				writeTerminalData(message.data)
			}
		} catch {
			writeTerminalData(event.data)
		}
	}
	currentSocket.onclose = () => {
		if (socket !== currentSocket) return
		socket = null
		connected.value = false
		updateControl(false)
		pendingResyncId = ""
		if (!closing && !reconnectTimer) {
			connecting.value = true
			reconnecting.value = true
			reconnectTimer = setTimeout(() => {
				reconnectTimer = null
				connect()
			}, 1500)
		}
	}
}

function openTerminal() {
	if (!terminalElement.value || terminal) return
	closing = false
	connecting.value = true
	lastSequence = 0
	pendingResyncId = ""
	reportedCols = 0
	reportedRows = 0
	terminal = new Terminal({
		disableStdin: true,
		cursorBlink: true,
		cursorStyle: "bar",
		fontSize: window.innerWidth < 640 ? 12 : 14,
		fontFamily: 'Menlo, Monaco, "Courier New", monospace',
		scrollback: 5000,
		theme: { background: "#0b1020", foreground: "#d4d4d8", cursor: "#60a5fa" },
	})
	fitAddon = new FitAddon()
	terminal.loadAddon(fitAddon)
	terminal.open(terminalElement.value)
	if (terminal.textarea) {
		terminal.textarea.inputMode = "text"
		terminal.textarea.autocapitalize = "none"
		terminal.textarea.autocomplete = "off"
		terminal.textarea.spellcheck = false
		terminal.textarea.setAttribute("autocorrect", "off")
	}
	terminal.onData(data => {
		sendTerminalInput(applyCtrlModifier(data))
	})
	resizeObserver = new ResizeObserver(scheduleFit)
	resizeObserver.observe(terminalElement.value)
	nextTick(() => {
		fit()
		connect()
	})
	pingTimer = setInterval(() => {
		if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: "ping" }))
	}, 15000)
}

function closeTerminal() {
	closing = true
	connected.value = false
	connecting.value = false
	updateControl(false)
	pendingResyncId = ""
	if (reconnectTimer) clearTimeout(reconnectTimer)
	if (pingTimer) clearInterval(pingTimer)
	if (fitFrame !== null) cancelAnimationFrame(fitFrame)
	reconnectTimer = null
	pingTimer = null
	fitFrame = null
	resizeObserver?.disconnect()
	resizeObserver = null
	const activeSocket = socket
	socket = null
	activeSocket?.close()
	terminal?.dispose()
	terminal = null
	fitAddon = null
}

function takeControl() {
	if (socket?.readyState !== WebSocket.OPEN) return
	socket.send(JSON.stringify({ type: "take_control", data: "" }))
}

function releaseControl() {
	if (socket?.readyState !== WebSocket.OPEN) return
	socket.send(JSON.stringify({ type: "release_control", data: "" }))
}

function openRenameModal() {
	renameTitle.value = props.taskName
	showRenameModal.value = true
}

async function renameSession() {
	const title = renameTitle.value.trim()
	if (!title || renameLoading.value) return
	renameLoading.value = true
	try {
		await updateMobileSessionTitle(props.sessionId, title)
		showRenameModal.value = false
		message.success(t("mobile.sessionRenameSuccess"))
		emit("renamed")
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.sessionRenameFailed"))
	} finally {
		renameLoading.value = false
	}
}

watch(
	() => props.sessionId,
	() => {
		closeTerminal()
		nextTick(openTerminal)
	},
	{ immediate: true },
)

onBeforeUnmount(closeTerminal)
</script>

<template>
	<section class="flex h-dvh w-full flex-col overflow-hidden bg-[#0b1020] text-white">
		<header class="flex shrink-0 items-center gap-2 border-b border-white/10 bg-slate-950/80 px-2 pb-2 pt-[max(8px,env(safe-area-inset-top))] backdrop-blur">
			<button class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-200 transition-colors active:bg-white/10" type="button" :title="t('commons.button.back')" :aria-label="t('commons.button.back')" @click="emit('back')">
				<svg viewBox="0 0 24 24" aria-hidden="true" class="h-6 w-6 fill-none stroke-current" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M19 12H5" />
					<path d="m12 19-7-7 7-7" />
				</svg>
			</button>
			<div class="min-w-0 flex-1">
				<div class="truncate text-sm font-semibold">{{ taskName }}</div>
			</div>
			<div class="flex shrink-0 items-center gap-1.5">
				<span class="max-w-[30vw] truncate text-right text-xs text-slate-400" :title="projectName">{{ projectName }}</span>
				<span class="h-2 w-2 rounded-full" :class="connected ? 'bg-emerald-400' : reconnecting ? 'bg-amber-400' : 'bg-slate-500'" :title="connected ? t('mobile.connected') : reconnecting ? t('mobile.reconnecting') : t('mobile.disconnected')" />
				<n-button v-if="mode === 'ai'" size="small" quaternary circle :title="t('mobile.renameSession')" :aria-label="t('mobile.renameSession')" @click="openRenameModal">
					<template #icon><Icon name="mdi:pencil-outline" :size="18" color="#cbd5e1" /></template>
				</n-button>
				<n-button v-if="mode === 'ai'" size="small" quaternary circle :title="t('mobile.taskStatus')" :aria-label="t('mobile.taskStatus')" @click="emit('openStatus')">
					<template #icon><Icon name="mdi:timeline-clock-outline" :size="19" color="#cbd5e1" /></template>
				</n-button>
				<n-button v-if="mode === 'ai'" size="small" quaternary circle :title="t('mobile.files')" :aria-label="t('mobile.files')" @click="emit('openFiles')">
					<template #icon><Icon name="mdi:folder-outline" :size="19" color="#cbd5e1" /></template>
				</n-button>
			</div>
		</header>
		<div class="relative min-h-0 w-full flex-1 bg-[#0b1020]">
			<div ref="terminalElement" class="h-full w-full" />
			<div v-if="connecting" class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-[#0b1020] text-sm text-slate-300" role="status" aria-live="polite">
				<Icon name="mdi:loading" :size="28" class="animate-spin text-blue-400" />
				<span>{{ reconnecting ? t("mobile.reconnecting") : t("mobile.terminalConnecting") }}</span>
			</div>
		</div>
		<div v-if="connected" class="flex min-h-12 items-center justify-between gap-3 border-t border-white/10 bg-slate-950 px-3 text-xs text-slate-400">
			<span>{{ hasControl ? t("mobile.terminalControlling") : t("mobile.terminalReadOnly") }}</span>
			<n-button size="small" round secondary type="primary" @click="hasControl ? releaseControl() : takeControl()">
				{{ hasControl ? t("mobile.releaseTerminalControl") : t("mobile.takeTerminalControl") }}
			</n-button>
		</div>
		<form class="flex shrink-0 items-center gap-2 border-t border-white/10 bg-slate-950 px-2 py-2" @submit.prevent="submitCommand">
			<input ref="commandInput" v-model="commandDraft" type="text" inputmode="text" enterkeyhint="send" autocapitalize="none" autocomplete="off" autocorrect="off" :spellcheck="false" :disabled="!hasControl" :placeholder="hasControl ? t('mobile.terminalInputPlaceholder') : t('mobile.terminalReadOnly')" class="h-10 min-w-0 flex-1 rounded-xl border border-white/10 bg-white/5 px-3 text-base text-white outline-none transition placeholder:text-slate-500 focus:border-blue-400/70 focus:bg-white/[0.08] disabled:cursor-not-allowed" @compositionstart="commandComposing = true" @compositionend="commandComposing = false" />
			<button type="submit" :disabled="!hasControl || !commandDraft" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-400/30 bg-blue-500/20 text-blue-100 transition active:scale-95 active:bg-blue-500/35 disabled:border-white/5 disabled:bg-white/5 disabled:text-slate-600" :title="t('mobile.sendTerminalInput')" :aria-label="t('mobile.sendTerminalInput')">
				<Icon name="mdi:send" :size="19" />
			</button>
		</form>
		<div class="flex shrink-0 items-center gap-1.5 overflow-x-auto border-t border-white/10 bg-slate-950 px-2 py-2 transition-opacity [scrollbar-width:none] [&::-webkit-scrollbar]:hidden" :class="hasControl ? '' : 'opacity-50'" role="toolbar" :aria-label="t('mobile.terminal')">
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="Esc" @pointerdown.prevent @click="sendShortcut('\x1b')">Esc</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="Tab" @pointerdown.prevent @click="sendShortcut('\t')">Tab</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" :class="ctrlActive ? 'border-blue-400 bg-blue-500/25 text-blue-200' : ''" aria-label="Ctrl" :aria-pressed="ctrlActive" @pointerdown.prevent @click="toggleCtrl">Ctrl</button>
			<button v-for="symbol in ['/', '-', '_', '.', '~', '|']" :key="symbol" type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-blue-400/20 bg-blue-500/10 px-2 font-mono text-base font-medium text-blue-200 transition active:scale-95 active:bg-blue-500/25 disabled:cursor-not-allowed" :aria-label="`${t('mobile.insertTerminalSymbol')} ${symbol}`" @pointerdown.prevent @click="insertCommandSymbol(symbol)">{{ symbol }}</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="←" @pointerdown.prevent @click="sendShortcut('\x1b[D')">←</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="↑" @pointerdown.prevent @click="sendShortcut('\x1b[A')">↑</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="↓" @pointerdown.prevent @click="sendShortcut('\x1b[B')">↓</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="→" @pointerdown.prevent @click="sendShortcut('\x1b[C')">→</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed" aria-label="⌫" @pointerdown.prevent @click="sendShortcut('\x7f')">⌫</button>
			<button type="button" :disabled="!hasControl" class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-blue-500/40 bg-blue-500/15 px-2 font-mono text-sm font-medium text-blue-200 transition active:scale-95 active:bg-blue-500/25 disabled:cursor-not-allowed" aria-label="↵" @pointerdown.prevent @click="sendShortcut('\r')">↵</button>
		</div>
		<div class="h-[max(8px,env(safe-area-inset-bottom))] shrink-0 bg-slate-950" aria-hidden="true" />
		<n-modal v-model:show="showRenameModal" preset="card" style="width: min(92vw, 420px)" :title="t('mobile.renameSession')">
			<n-input v-model:value="renameTitle" maxlength="255" show-count autofocus :placeholder="t('mobile.sessionNamePlaceholder')" @keydown.enter.prevent="renameSession" />
			<template #footer>
				<div class="flex justify-end gap-2">
					<n-button @click="showRenameModal = false">{{ t("mobile.cancel") }}</n-button>
					<n-button type="primary" :loading="renameLoading" :disabled="!renameTitle.trim()" @click="renameSession">{{ t("mobile.renameSession") }}</n-button>
				</div>
			</template>
		</n-modal>
	</section>
</template>

<style scoped>
:deep(.xterm) {
	height: 100%;
	padding: 12px 2px 12px 10px;
}

:deep(.xterm-viewport) {
	overscroll-behavior-y: contain;
	-webkit-overflow-scrolling: touch;
}
</style>
