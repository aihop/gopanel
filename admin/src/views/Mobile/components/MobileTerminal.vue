<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import "xterm/css/xterm.css"

const props = defineProps<{ show: boolean; sessionId: number }>()
const emit = defineEmits<{ (event: "update:show", value: boolean): void }>()
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
	socket = new WebSocket(url)
	socket.onopen = () => {
		connected.value = true
		reconnecting.value = false
	}
	socket.onmessage = event => {
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
	socket.onclose = () => {
		connected.value = false
		hasControl.value = false
		if (!closing && props.show && !reconnectTimer) {
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
	socket?.close()
	socket = null
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
	() => [props.show, props.sessionId] as const,
	([show]) => {
		if (show) nextTick(openTerminal)
		else closeTerminal()
	}
)

onBeforeUnmount(closeTerminal)
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="min(820px, 94dvh)" @update:show="emit('update:show', $event)">
		<n-drawer-content :closable="false" body-content-style="padding: 0; background: #0b1020;">
			<template #header>
				<div class="flex w-full min-w-0 items-center justify-between gap-2">
					<div class="min-w-0">
						<div class="font-semibold">{{ t("mobile.sharedTerminal") }}</div>
						<div class="mt-0.5 text-xs text-slate-400">
							{{ hasControl ? t("mobile.terminalControlling") : t("mobile.terminalReadOnly") }}
						</div>
					</div>
					<div class="flex shrink-0 items-center gap-2">
						<n-tag size="small" :type="connected ? 'success' : reconnecting ? 'warning' : 'default'" :bordered="false" round>
							{{ connected ? t("mobile.connected") : reconnecting ? t("mobile.reconnecting") : t("mobile.disconnected") }}
						</n-tag>
						<n-button v-if="connected && !hasControl" size="small" type="primary" @click="takeControl">
							{{ t("mobile.takeTerminalControl") }}
						</n-button>
						<n-button v-else-if="connected" size="small" secondary @click="releaseControl">
							{{ t("mobile.releaseTerminalControl") }}
						</n-button>
						<n-button quaternary circle :title="t('mobile.close')" @click="emit('update:show', false)">
							<template #icon><Icon name="mdi:close" /></template>
						</n-button>
					</div>
				</div>
			</template>
			<div ref="terminalElement" class="h-full min-h-[65dvh] w-full bg-[#0b1020]" />
		</n-drawer-content>
	</n-drawer>
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
