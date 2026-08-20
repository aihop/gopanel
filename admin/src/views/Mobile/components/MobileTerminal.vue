<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import useClipboard from "vue-clipboard3"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import { mobileTerminalMessages } from "../mobileTerminalMessages"
import MobileTerminalHeader from "./MobileTerminalHeader.vue"
import MobileTerminalInput from "./MobileTerminalInput.vue"
import MobileTerminalRenameModal from "./MobileTerminalRenameModal.vue"
import { terminalBufferText } from "./mobileTerminalClipboard"
import {
	applyMobileTerminalCtrlModifier,
	mobileTerminalShellStyle,
	mobileTerminalSocketUrl,
	mobileTerminalViewportHeight,
	MobileTerminalInputFallback,
} from "./mobileTerminalInput"
import { MobileTerminalOutputQueue } from "./mobileTerminalOutputQueue"
import {
	authoritativeTerminalSize,
	codeTerminalOptions,
	keepingTerminalBottom,
	shouldAutoAcquireTerminalControl,
	terminalSizeData,
	terminalTakeControlMessage,
} from "@/views/Code/components/codeTerminalSession"
import "./mobileTerminal.css"
import "@xterm/xterm/css/xterm.css"

const props = withDefaults(
	defineProps<{
		sessionId: number
		taskName: string
		projectName: string
		mode?: "ai" | "native"
	}>(),
	{ mode: "ai" },
)
const emit = defineEmits<{ back: []; openFiles: []; openStatus: []; renamed: [] }>()
const terminalMessages = {
	zh: { mobile: { ...mobileMessages.zh.mobile, ...mobileTerminalMessages.zh.mobile } },
	en: { mobile: { ...mobileMessages.en.mobile, ...mobileTerminalMessages.en.mobile } },
}
const { t } = useI18n({ messages: terminalMessages })
const message = useMessage()
const { toClipboard } = useClipboard()
const terminalElement = ref<HTMLElement | null>(null)
const connected = ref(false)
const connecting = ref(true)
const reconnecting = ref(false)
const hasControl = ref(false)
const ctrlActive = ref(false)
const hasTerminalSelection = ref(false)
const renameModal = ref<{ open: () => void } | null>(null)
const viewportHeight = ref(0)
const terminalShellStyle = computed(() => mobileTerminalShellStyle(viewportHeight.value))
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let inputFallback: MobileTerminalInputFallback | null = null
let outputQueue: MobileTerminalOutputQueue | null = null
let socket: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let pingTimer: ReturnType<typeof setInterval> | null = null
let fitFrame: number | null = null
let reportedCols = 0
let reportedRows = 0
let receivedSequence = 0
let renderedSequence = 0
let resyncRequest = 0
let pendingResyncId = ""
let closing = false

function syncViewportHeight() {
	viewportHeight.value = mobileTerminalViewportHeight()
	nextTick(scheduleFit)
}

function sendAck(sequence: number) {
	if (props.mode === "ai" && socket?.readyState === WebSocket.OPEN && sequence > 0) {
		socket.send(JSON.stringify({ type: "ack", data: String(sequence) }))
	}
}

function requestResync() {
	if (socket?.readyState !== WebSocket.OPEN || pendingResyncId) return
	outputQueue?.clear()
	receivedSequence = renderedSequence
	if (props.mode === "native") {
		pendingResyncId = "native"
		socket.send(JSON.stringify({ type: "resync", data: "" }))
		return
	}
	pendingResyncId = `${Date.now()}-${++resyncRequest}`
	socket.send(
		JSON.stringify({
			type: "resync",
			data: JSON.stringify({ sequence: renderedSequence, requestId: pendingResyncId }),
		}),
	)
}

function fit() {
	try {
		if ((!connected.value || hasControl.value) && terminal && fitAddon) {
			keepingTerminalBottom(terminal, () => fitAddon?.fit())
		}
		if (
			socket?.readyState === WebSocket.OPEN &&
			hasControl.value &&
			terminal &&
			(terminal.cols !== reportedCols || terminal.rows !== reportedRows)
		) {
			socket.send(
				JSON.stringify({
					type: "resize",
					data: terminalSizeData(terminal.cols, terminal.rows),
				}),
			)
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
	if (value)
		nextTick(() => {
			terminal?.focus()
			scheduleFit()
		})
}

function applyAuthoritativeSize(cols: unknown, rows: unknown) {
	const size = authoritativeTerminalSize(cols, rows)
	if (!terminal || !size || (terminal.cols === size.cols && terminal.rows === size.rows)) return
	keepingTerminalBottom(terminal, () => terminal?.resize(size.cols, size.rows))
}

function sendTerminalInput(data: string) {
	if (!hasControl.value || socket?.readyState !== WebSocket.OPEN) return
	socket.send(JSON.stringify({ type: "cmd", data }))
}

function queueTerminalData(
	data: string,
	options: { sequence?: number; forceBottom?: boolean; resetBefore?: boolean } = {},
) {
	return outputQueue?.enqueue({ data, ...options }) ?? false
}

function applyCtrlModifier(data: string) {
	if (!ctrlActive.value) return data
	ctrlActive.value = false
	return applyMobileTerminalCtrlModifier(true, data)
}

function sendShortcut(data: string) {
	ctrlActive.value = false
	sendTerminalInput(data)
	nextTick(() => terminal?.focus())
}

async function copyTerminalContent(selectionOnly: boolean) {
	if (!terminal) return
	const content = selectionOnly ? terminal.getSelection() : terminalBufferText(terminal.buffer.active)
	if (!content) {
		message.warning(t(selectionOnly ? "mobile.noTerminalSelection" : "mobile.noTerminalOutput"))
		return
	}
	try {
		await toClipboard(content)
		terminal.clearSelection()
		hasTerminalSelection.value = false
		message.success(t("mobile.terminalCopied"))
	} catch (error) {
		void error
		message.error(t("mobile.terminalCopyFailed"))
	}
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
	const currentSocket = new WebSocket(
		mobileTerminalSocketUrl({
			mode: props.mode,
			sessionId: props.sessionId,
			cols: terminal.cols,
			rows: terminal.rows,
			host: window.location.host,
			protocol: window.location.protocol,
			afterSequence: renderedSequence,
		}),
	)
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
				if (props.mode === "ai") applyAuthoritativeSize(message.cols, message.rows)
				const sequence = Number(message.sequence) || 0
				const chunkIndex = Number(message.chunkIndex) || 0
				const chunkCount = Number(message.chunkCount) || 1
				const isLastChunk = chunkIndex === chunkCount - 1
				const replaceHistory = (props.mode === "native" || message.truncated) && chunkIndex === 0
				if (replaceHistory) outputQueue?.clear()
				if (
					!queueTerminalData(message.data || "", {
						sequence: isLastChunk ? sequence : undefined,
						forceBottom: isLastChunk,
						resetBefore: replaceHistory,
					})
				)
					return
				if (isLastChunk) {
					receivedSequence = sequence
					pendingResyncId = ""
					const gainedControl = Boolean(message.hasControl)
					updateControl(gainedControl)
					if (
						props.mode === "ai" &&
						shouldAutoAcquireTerminalControl(gainedControl, message.controlReason, message.leaseExpiresAt)
					)
						void nextTick(takeControl)
				}
			} else if (message.type === "output") {
				const sequence = Number(message.sequence) || 0
				if (pendingResyncId) return
				if (receivedSequence > 0 && sequence !== receivedSequence + 1) {
					requestResync()
					return
				}
				if (!queueTerminalData(message.data || "", { sequence })) return
				receivedSequence = sequence
			} else if (message.type === "resync_required") {
				requestResync()
			} else if (message.type === "control") {
				if (props.mode === "ai") applyAuthoritativeSize(message.cols, message.rows)
				const gainedControl = Boolean(message.hasControl)
				updateControl(gainedControl)
				if (
					props.mode === "ai" &&
					shouldAutoAcquireTerminalControl(gainedControl, message.controlReason, message.leaseExpiresAt)
				)
					void nextTick(takeControl)
				if (message.controlReason || message.data)
					queueTerminalData(`\r\n\x1b[33m[GoPanel] ${t("mobile.terminalControlBusy")}\x1b[0m\r\n`)
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
				if (message.data) queueTerminalData(`\r\n\x1b[31m[GoPanel] ${message.data}\x1b[0m`)
			} else if (message.type === "cmd" && message.data) {
				queueTerminalData(message.data)
			}
		} catch {
			queueTerminalData(event.data)
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
	receivedSequence = 0
	renderedSequence = 0
	pendingResyncId = ""
	reportedCols = 0
	reportedRows = 0
	terminal = new Terminal({
		...codeTerminalOptions(),
		disableStdin: true,
		screenReaderMode: true,
		cursorBlink: false,
		cursorStyle: "bar",
		fontSize: window.innerWidth < 640 ? 12 : 14,
		fontFamily: 'Menlo, Monaco, "Courier New", monospace',
		scrollback: 2000,
		scrollSensitivity: 1,
		fastScrollSensitivity: 3,
		smoothScrollDuration: 0,
		theme: { background: "#0b1020", foreground: "#d4d4d8", cursor: "#60a5fa" },
	})
	syncViewportHeight()
	window.visualViewport?.addEventListener("resize", syncViewportHeight)
	window.visualViewport?.addEventListener("scroll", syncViewportHeight)
	fitAddon = new FitAddon()
	terminal.loadAddon(fitAddon)
	terminal.open(terminalElement.value)
	outputQueue = new MobileTerminalOutputQueue(terminal, {
		onRendered: sequence => {
			renderedSequence = Math.max(renderedSequence, sequence)
			sendAck(renderedSequence)
		},
		onOverflow: requestResync,
	})
	const viewport = terminalElement.value.querySelector<HTMLElement>(".xterm-viewport")
	if (viewport) outputQueue.bindTouchScrolling(viewport)
	if (terminal.textarea) {
		const textarea = terminal.textarea
		textarea.inputMode = "text"
		textarea.autocapitalize = "none"
		textarea.autocomplete = "off"
		textarea.spellcheck = false
		textarea.setAttribute("autocorrect", "off")
		inputFallback = new MobileTerminalInputFallback()
		textarea.addEventListener("compositionstart", () => inputFallback?.startComposition())
		textarea.addEventListener("compositionend", () => inputFallback?.endComposition())
		textarea.addEventListener("input", event => {
			inputFallback?.queueInput(event as InputEvent, data => sendTerminalInput(applyCtrlModifier(data)))
		})
	}
	terminal.onData(data => {
		inputFallback?.recordTerminalData(data)
		sendTerminalInput(applyCtrlModifier(data))
	})
	terminal.onSelectionChange(() => {
		hasTerminalSelection.value = Boolean(terminal?.hasSelection())
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
	window.visualViewport?.removeEventListener("resize", syncViewportHeight)
	window.visualViewport?.removeEventListener("scroll", syncViewportHeight)
	outputQueue?.dispose()
	outputQueue = null
	inputFallback?.dispose()
	inputFallback = null
	const activeSocket = socket
	socket = null
	activeSocket?.close()
	terminal?.dispose()
	terminal = null
	hasTerminalSelection.value = false
	fitAddon = null
}

function takeControl() {
	if (socket?.readyState !== WebSocket.OPEN || !terminal) return
	const dimensions = fitAddon?.proposeDimensions() || { cols: terminal.cols, rows: terminal.rows }
	socket.send(terminalTakeControlMessage(dimensions.cols, dimensions.rows))
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
	{ immediate: true },
)

onBeforeUnmount(closeTerminal)
</script>

<template>
	<section
		class="flex h-dvh w-full flex-col overflow-hidden bg-[#0b1020] text-white"
		:style="terminalShellStyle"
	>
		<MobileTerminalHeader
			:task-name="taskName"
			:project-name="projectName"
			:mode="mode"
			:connected="connected"
			:reconnecting="reconnecting"
			:has-selection="hasTerminalSelection"
			@back="emit('back')"
			@open-files="emit('openFiles')"
			@open-status="emit('openStatus')"
			@open-rename="renameModal?.open()"
			@copy-selection="copyTerminalContent(true)"
			@copy-output="copyTerminalContent(false)"
		/>
		<div class="mobile-terminal relative min-h-0 w-full flex-1 bg-[#0b1020]">
			<div ref="terminalElement" class="h-full w-full" />
			<div
				v-if="connecting"
				class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-[#0b1020] text-sm text-slate-300"
				role="status"
				aria-live="polite"
			>
				<Icon name="mdi:loading" :size="28" class="animate-spin text-blue-400" />
				<span>{{ reconnecting ? t("mobile.reconnecting") : t("mobile.terminalConnecting") }}</span>
			</div>
		</div>
		<MobileTerminalInput
			:connected="connected"
			:has-control="hasControl"
			:ctrl-active="ctrlActive"
			@take-control="takeControl"
			@release-control="releaseControl"
			@send="sendTerminalInput"
			@shortcut="sendShortcut"
			@toggle-ctrl="toggleCtrl"
		/>
		<div class="h-[max(8px,env(safe-area-inset-bottom))] shrink-0 bg-slate-950" aria-hidden="true" />
		<MobileTerminalRenameModal
			ref="renameModal"
			:session-id="sessionId"
			:task-name="taskName"
			@renamed="emit('renamed')"
		/>
	</section>
</template>
