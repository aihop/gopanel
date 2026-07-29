<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import "xterm/css/xterm.css"

const props = defineProps<{ sessionId: number; title: string; projectName: string }>()
const emit = defineEmits<{ back: []; openFiles: [] }>()
const { t } = useI18n({ messages: mobileMessages })
const terminalElement = ref<HTMLElement | null>(null)
const connected = ref(false)
const reconnecting = ref(false)
const hasControl = ref(false)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let socket: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let pingTimer: ReturnType<typeof setInterval> | null = null
let lastSequence = 0
let closing = false

function fit() {
	try {
		fitAddon?.fit()
		if (socket?.readyState === WebSocket.OPEN && hasControl.value && terminal) {
			socket.send(JSON.stringify({
				type: "resize",
				data: JSON.stringify({ cols: terminal.cols, rows: terminal.rows })
			}))
		}
	} catch {
	}
}

function connect() {
	if (!terminal || !props.sessionId) return
	const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
	let url = `${protocol}//${window.location.host}/api/mobile/app/terminal?session_id=${props.sessionId}`
	url += `&cols=${terminal.cols}&rows=${terminal.rows}&read_only=1`
	if (lastSequence) url += `&after_sequence=${lastSequence}`
	const currentSocket = new WebSocket(url)
	socket = currentSocket
	currentSocket.onopen = () => {
		if (socket !== currentSocket) return
		connected.value = true
		reconnecting.value = false
	}
	currentSocket.onmessage = event => {
		if (socket !== currentSocket) return
		try {
			const message = JSON.parse(event.data)
			if (message.type === "baseline" || message.type === "output") {
				const sequence = Number(message.sequence) || 0
				lastSequence = message.type === "baseline" ? sequence : Math.max(lastSequence, sequence)
				if (message.type === "baseline") hasControl.value = Boolean(message.hasControl)
				if (message.data) terminal?.write(message.data)
			} else if (message.type === "control") {
				hasControl.value = Boolean(message.hasControl)
				if (hasControl.value) nextTick(() => terminal?.focus())
			} else if (message.type === "closed") {
				closing = true
				connected.value = false
				hasControl.value = false
			} else if (message.type === "cmd" && message.data) {
				terminal?.write(message.data)
			}
		} catch {
			terminal?.write(event.data)
		}
	}
	currentSocket.onclose = () => {
		if (socket !== currentSocket) return
		socket = null
		connected.value = false
		hasControl.value = false
		if (!closing && !reconnectTimer) {
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
	lastSequence = 0
	terminal = new Terminal({
		cursorBlink: true,
		cursorStyle: "bar",
		fontSize: window.innerWidth < 640 ? 12 : 14,
		fontFamily: 'Menlo, Monaco, "Courier New", monospace',
		scrollback: 5000,
		theme: { background: "#0b1020", foreground: "#d4d4d8", cursor: "#60a5fa" }
	})
	fitAddon = new FitAddon()
	terminal.loadAddon(fitAddon)
	terminal.open(terminalElement.value)
	terminal.onData(data => {
		if (hasControl.value && socket?.readyState === WebSocket.OPEN) {
			socket.send(JSON.stringify({ type: "cmd", data }))
		}
	})
	resizeObserver = new ResizeObserver(fit)
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
	hasControl.value = false
	if (reconnectTimer) clearTimeout(reconnectTimer)
	if (pingTimer) clearInterval(pingTimer)
	reconnectTimer = null
	pingTimer = null
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

watch(
	() => props.sessionId,
	() => {
		closeTerminal()
		nextTick(openTerminal)
	},
	{ immediate: true }
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
				<div class="truncate text-sm font-semibold">{{ projectName }}</div>
				<div class="mt-0.5 truncate text-[11px] text-slate-400">{{ title }}</div>
			</div>
			<div class="flex shrink-0 items-center gap-1.5">
				<n-tag size="small" :type="connected ? 'success' : reconnecting ? 'warning' : 'default'" :bordered="false" round>
					{{ connected ? t("mobile.connected") : reconnecting ? t("mobile.reconnecting") : t("mobile.disconnected") }}
				</n-tag>
				<n-button v-if="connected && !hasControl" size="tiny" type="primary" @click="takeControl">
					{{ t("mobile.takeTerminalControl") }}
				</n-button>
				<n-button v-else-if="connected" size="tiny" quaternary text-color="#cbd5e1" @click="releaseControl">
					{{ t("mobile.releaseTerminalControl") }}
				</n-button>
				<n-button size="small" quaternary circle text-color="#cbd5e1" :title="t('mobile.files')" :aria-label="t('mobile.files')" @click="emit('openFiles')">
					<template #icon><Icon name="mdi:folder-outline" :size="19" /></template>
				</n-button>
			</div>
		</header>
		<div ref="terminalElement" class="min-h-0 w-full flex-1 bg-[#0b1020] pb-[env(safe-area-inset-bottom)]" />
	</section>
</template>

<style scoped>
:deep(.xterm) {
	height: 100%;
	padding: 12px;
}

:deep(.xterm-screen),
:deep(.xterm-helpers),
:deep(.xterm-viewport) {
	height: 100%;
}

:deep(.xterm-viewport) {
	overflow-y: auto !important;
}
</style>
