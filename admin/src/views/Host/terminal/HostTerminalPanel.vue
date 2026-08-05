<script setup lang="ts">
import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import { nextTick, onActivated, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useAuthStore } from "@/store/auth"
import "@xterm/xterm/css/xterm.css"

const props = defineProps<{ sessionId: number }>()
const emit = defineEmits<{ closed: []; status: [status: string] }>()
const { t } = useI18n()
const authStore = useAuthStore()
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
let reconnectAttempts = 0
let activatedOnce = false

function send(type: string, data = "") {
	if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type, data }))
}

function fit() {
	try {
		fitAddon?.fit()
		if (terminal && hasControl.value) {
			send("resize", JSON.stringify({ cols: terminal.cols, rows: terminal.rows }))
		}
	} catch {}
}

function connect() {
	if (!terminal || !props.sessionId) return
	const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
	const token = encodeURIComponent(authStore.auth || "")
	const currentSocket = new WebSocket(`${protocol}//${window.location.host}/api/host/terminal/sessions/${props.sessionId}/ws?token=${token}`)
	socket = currentSocket
	currentSocket.onopen = () => {
		if (socket !== currentSocket) return
		connected.value = true
		reconnecting.value = false
		reconnectAttempts = 0
	}
	currentSocket.onmessage = event => {
		if (socket !== currentSocket) return
		try {
			const message = JSON.parse(event.data)
			if (message.type === "baseline") {
				terminal?.reset()
				if (message.data) terminal?.write(message.data)
				lastSequence = Number(message.sequence) || 0
				hasControl.value = Boolean(message.hasControl)
				if (message.truncated) terminal?.writeln(`\r\n\x1b[33m[GoPanel] ${t("terminal.historyTruncated")}\x1b[0m`)
				nextTick(fit)
			} else if (message.type === "output") {
				const sequence = Number(message.sequence) || 0
				if (lastSequence && sequence !== lastSequence + 1) {
					send("resync")
					return
				}
				if (message.data) terminal?.write(message.data)
				lastSequence = sequence
			} else if (message.type === "resync_required") {
				send("resync")
			} else if (message.type === "control") {
				hasControl.value = Boolean(message.hasControl)
				if (message.data) terminal?.writeln(`\r\n\x1b[33m[GoPanel] ${message.data}\x1b[0m`)
			} else if (message.type === "error") {
				closing = true
				terminal?.writeln(`\r\n\x1b[31m[GoPanel] ${message.data}\x1b[0m`)
				emit("status", "interrupted")
			} else if (message.type === "closed") {
				closing = true
				hasControl.value = false
				emit("closed")
			}
		} catch {
			terminal?.write(String(event.data))
		}
	}
	currentSocket.onclose = () => {
		if (socket !== currentSocket) return
		socket = null
		connected.value = false
		hasControl.value = false
		if (!closing && reconnectAttempts < 5 && !reconnectTimer) {
			reconnecting.value = true
			reconnectAttempts++
			reconnectTimer = setTimeout(() => {
				reconnectTimer = null
				connect()
			}, 1500)
		}
	}
}

function openTerminal() {
	if (!terminalElement.value) return
	closing = false
	lastSequence = 0
	terminal = new Terminal({
		cursorBlink: true,
		fontSize: 14,
		fontFamily: 'Menlo, Monaco, "Courier New", monospace',
		scrollback: 5000,
		theme: { background: "#0b1020", foreground: "#d4d4d8", cursor: "#60a5fa" }
	})
	fitAddon = new FitAddon()
	terminal.loadAddon(fitAddon)
	terminal.open(terminalElement.value)
	terminal.onData(data => {
		if (hasControl.value) send("cmd", data)
	})
	resizeObserver = new ResizeObserver(fit)
	resizeObserver.observe(terminalElement.value)
	nextTick(() => {
		fit()
		connect()
	})
	pingTimer = setInterval(() => send("ping"), 15000)
}

function disposeTerminal() {
	closing = true
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
	send("take_control")
	terminal?.focus()
}

function releaseControl() {
	send("release_control")
}

watch(() => props.sessionId, () => {
	disposeTerminal()
	nextTick(openTerminal)
})
onMounted(openTerminal)
onActivated(() => {
	if (!activatedOnce) {
		activatedOnce = true
		return
	}
	nextTick(() => {
		fit()
		if (!socket && !closing && terminal && !reconnectTimer) {
			reconnectAttempts = 0
			connect()
		}
	})
})
onBeforeUnmount(disposeTerminal)
</script>

<template>
	<div class="flex h-full min-h-0 flex-col overflow-hidden rounded-xl bg-[#0b1020]">
		<div class="flex h-11 shrink-0 items-center justify-between border-b border-white/10 px-4 text-xs text-slate-300">
			<div class="flex items-center gap-2">
				<span class="h-2 w-2 rounded-full" :class="connected ? 'bg-emerald-400' : reconnecting ? 'bg-amber-400' : 'bg-slate-500'" />
				<span>{{ connected ? t("terminal.connected") : reconnecting ? t("terminal.reconnecting") : t("terminal.disconnected") }}</span>
			</div>
			<div class="flex items-center gap-2">
				<n-tag v-if="hasControl" size="small" type="success" :bordered="false">{{ t("terminal.controlling") }}</n-tag>
				<n-button v-else size="tiny" type="warning" @click="takeControl">{{ t("terminal.takeControl") }}</n-button>
				<n-button v-if="hasControl" size="tiny" quaternary text-color="#cbd5e1" @click="releaseControl">{{ t("terminal.releaseControl") }}</n-button>
			</div>
		</div>
		<div ref="terminalElement" class="min-h-0 flex-1" />
	</div>
</template>

<style scoped>
:deep(.xterm) {
	height: 100%;
	padding: 14px;
}
:deep(.xterm-viewport) {
	overflow-y: auto !important;
}
</style>
