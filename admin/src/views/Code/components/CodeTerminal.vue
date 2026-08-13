<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick, watch } from "vue"
import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import "@xterm/xterm/css/xterm.css"
import { useAuthStore } from "@/store/auth"
import { useI18n } from "vue-i18n"
import { getCodeSession } from "@/api/modules/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { useCodexRuntimeState } from "../useCodexRuntimeState"
import CodeTerminalStatusBar from "./CodeTerminalStatusBar.vue"
import { isDeliveredCodeSession } from "./codeTerminalSession"

const authStore = useAuthStore()
const { t } = useI18n({ messages: codeProjectMessages })

const props = defineProps<{
	taskId: number | null
	sessionId?: number | null
	autoTakeControl?: boolean
}>()

const emit = defineEmits<{
	(e: "task-created", taskId: number): void
	(e: "new-session"): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
let term: Terminal
let fitAddon: FitAddon
let ws: WebSocket
let pingInterval: any
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let resizeObserver: ResizeObserver | null = null
let intentionalClose = false
let serverErrorShown = false
let lastSequence = 0
let resyncRequest = 0
let pendingResyncId = ""
let receivedServerMessage = false
let initialReconnectAttempts = 0
let autoTakeControlPending = Boolean(props.autoTakeControl)
let activatedOnce = false
// 实例会被 KeepAlive 缓存复用：切走时 DOM 进隐藏容器但 WebSocket 不断，
// isActive 用来把「只有在屏幕上才该做」的事（fit、codex 状态轮询）关掉。
const isActive = ref(true)
const nativeProtocol = ref(false)
const hasTerminalControl = ref(true)
const reconnecting = ref(false)
const connectionFailed = ref(false)
const sessionDelivered = ref(false)
const {
	runtimeState,
	runtimeError,
	runtimeSupported,
	executorId,
	loadRuntimeState,
	startRuntimePolling,
	stopRuntimePolling,
	disableRuntimeState,
} = useCodexRuntimeState(() => props.sessionId, isActive)

const markSessionDelivered = () => {
	sessionDelivered.value = true
	hasTerminalControl.value = false
	reconnecting.value = false
	if (pingInterval) clearInterval(pingInterval)
	pingInterval = null
	disableRuntimeState()
	resizeObserver?.disconnect()
	if (term) term.dispose()
}

const refreshDeliveredSession = async () => {
	if (!props.sessionId) return false
	try {
		const response = await getCodeSession(props.sessionId)
		const session = response.data.session
		executorId.value = session.agentName || ""
		if (!isDeliveredCodeSession(session.status)) return false
		markSessionDelivered()
		return true
	} catch {
		return false
	}
}

const sendTerminalAck = (sequence: number) => {
	if (ws?.readyState === WebSocket.OPEN && sequence > 0) {
		ws.send(JSON.stringify({ type: "ack", data: String(sequence) }))
	}
}

const writeTerminalData = (data: string, forceBottom = false) => {
	const buffer = term.buffer.active
	const shouldFollow = forceBottom || buffer.baseY - buffer.viewportY <= 1
	term.write(data, () => {
		if (shouldFollow) term.scrollToBottom()
	})
}

const requestTerminalResync = () => {
	if (ws?.readyState !== WebSocket.OPEN || pendingResyncId) return
	pendingResyncId = `${Date.now()}-${++resyncRequest}`
	ws.send(JSON.stringify({
		type: "resync",
		data: JSON.stringify({ sequence: lastSequence, requestId: pendingResyncId }),
	}))
}

const initTerminal = () => {
	if (!terminalRef.value) return

	term = new Terminal({
		cursorBlink: true,
		fontSize: 14,
		fontFamily: 'Menlo, Monaco, "Courier New", monospace',
		theme: {
			background: "#1e1e1e",
			foreground: "#d4d4d4",
		},
	})

	fitAddon = new FitAddon()
	term.loadAddon(fitAddon)
	term.open(terminalRef.value)

	nextTick(() => {
		try {
			fitAddon.fit()
		} catch (e) {
			console.warn("Fit addon error on init", e)
		}
	})

	connectWebSocket()

	term.onData(data => {
		if (ws && ws.readyState === WebSocket.OPEN && hasTerminalControl.value) {
			ws.send(JSON.stringify({ type: "cmd", data }))
		}
	})

	pingInterval = setInterval(() => {
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: "ping" }))
		}
	}, 15000)
}

const connectWebSocket = () => {
	const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
	const token = authStore.auth || ""

	let wsUrl = `${protocol}//${window.location.host}/api/code/terminal?token=${token}&cols=${term.cols}&rows=${term.rows}`

	if (props.sessionId) {
		wsUrl += `&session_id=${props.sessionId}`
		if (lastSequence > 0) wsUrl += `&after_sequence=${lastSequence}`
		if (autoTakeControlPending) {
			wsUrl += "&take_control=1"
		}
	} else if (props.taskId) {
		wsUrl += `&task_id=${props.taskId}`
	}

	ws = new WebSocket(wsUrl)

	ws.onopen = () => {
		reconnecting.value = false
		if (props.sessionId) {
			term.writeln(`\x1b[32m[GoPanel] ${t("code.openingSession", { id: props.sessionId })}\x1b[0m\r\n`)
		} else if (props.taskId) {
			term.writeln(`\x1b[32m[GoPanel] 正在恢复历史任务 #${props.taskId} 的上下文...\x1b[0m\r\n`)
		}
	}

	ws.onmessage = event => {
		receivedServerMessage = true
		initialReconnectAttempts = 0
		if (typeof event.data === "string" && (event.data.includes("失败") || event.data.includes("错误"))) {
			serverErrorShown = true
			connectionFailed.value = true
		}
		try {
			const msg = JSON.parse(event.data)
			if (msg.type === "baseline") {
				nativeProtocol.value = true
				connectionFailed.value = false
				if (pendingResyncId && msg.requestId !== pendingResyncId) return
				autoTakeControlPending = false
				const sequence = Number(msg.sequence) || 0
				const chunkIndex = Number(msg.chunkIndex) || 0
				const chunkCount = Number(msg.chunkCount) || 1
				if (msg.truncated && chunkIndex === 0) term.reset()
				if (msg.data) writeTerminalData(msg.data, chunkIndex === chunkCount - 1)
				else if (chunkIndex === chunkCount - 1) term.scrollToBottom()
				if (chunkIndex === chunkCount - 1) {
					lastSequence = sequence
					pendingResyncId = ""
					hasTerminalControl.value = Boolean(msg.hasControl)
					sendTerminalAck(sequence)
				}
			} else if (msg.type === "output") {
				nativeProtocol.value = true
				connectionFailed.value = false
				const sequence = Number(msg.sequence) || 0
				if (pendingResyncId) return
				if (lastSequence > 0 && sequence !== lastSequence + 1) {
					requestTerminalResync()
					return
				}
				if (msg.data) writeTerminalData(msg.data)
				lastSequence = sequence
				sendTerminalAck(sequence)
			} else if (msg.type === "resync_required") {
				requestTerminalResync()
			} else if (msg.type === "control") {
				hasTerminalControl.value = Boolean(msg.hasControl)
				if (msg.controlReason) term.writeln(`\r\n\x1b[33m[GoPanel] ${t("code.terminalControlBusy")}\x1b[0m`)
			} else if (msg.type === "closed") {
				intentionalClose = true
				hasTerminalControl.value = false
				lastSequence = 0
			} else if (msg.type === "cmd") {
				writeTerminalData(msg.data)
			} else if (msg.type === "meta" && msg.task_id) {
				// 后端通知前端：新任务已创建，请更新 URL 或左侧列表
				emit("task-created", msg.task_id)
			} else if (msg.type === "pong") {
				// do nothing
			} else {
				writeTerminalData(event.data)
			}
		} catch (e) {
			writeTerminalData(event.data)
		}
	}

	ws.onerror = () => {
		term.writeln("\r\n\x1b[31m[错误] WebSocket 连接失败。\x1b[0m")
	}

	ws.onclose = () => {
		pendingResyncId = ""
		if (intentionalClose) {
			return
		}
		if (!serverErrorShown) {
			term.writeln("\r\n\x1b[33m[系统] 终端连接已断开。\x1b[0m")
		}
		const canRetryInitialConnection = !receivedServerMessage && initialReconnectAttempts < 3
		if (!connectionFailed.value && (nativeProtocol.value || canRetryInitialConnection) && !reconnectTimer) {
			initialReconnectAttempts++
			reconnecting.value = true
			reconnectTimer = setTimeout(async () => {
				reconnectTimer = null
				if (await refreshDeliveredSession()) return
				if (intentionalClose) return
				connectWebSocket()
			}, 1500)
		}
	}
}

const reconnectTerminal = () => {
	if (reconnectTimer) {
		clearTimeout(reconnectTimer)
		reconnectTimer = null
	}
	if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
		ws.onclose = null
		ws.close()
	}
	serverErrorShown = false
	receivedServerMessage = false
	initialReconnectAttempts = 0
	connectionFailed.value = false
	reconnecting.value = true
	connectWebSocket()
}

const takeTerminalControl = () => {
	if (ws && ws.readyState === WebSocket.OPEN) {
		ws.send(JSON.stringify({ type: "take_control", data: "" }))
		term.focus()
	}
}

const handleResize = () => {
	if (!fitAddon) return
	// 实例被缓存起来时 DOM 会被挪进隐藏容器，ResizeObserver 会以 0×0 触发一次。
	// 这时 fit() 算出来的 cols/rows 是垃圾值，发给后端会把 PTY 尺寸改坏，所以直接跳过。
	if (!isActive.value) return
	const element = terminalRef.value
	if (!element || element.clientWidth === 0 || element.clientHeight === 0) return
	try {
		fitAddon.fit()
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: "resize", data: JSON.stringify({ cols: term.cols, rows: term.rows }) }))
		}
	} catch (e) {
		console.warn("Fit addon resize error", e)
	}
}

const initializeSessionTerminal = async () => {
	if (await refreshDeliveredSession()) return
	if (intentionalClose) return
	initTerminal()
	startRuntimePolling()
}

onMounted(() => {
	void initializeSessionTerminal()
	window.addEventListener("resize", handleResize)
	if (terminalRef.value && typeof ResizeObserver !== "undefined") {
		resizeObserver = new ResizeObserver(() => {
			handleResize()
		})
		resizeObserver.observe(terminalRef.value)
	}
})

// 接管终端不再靠重建实例：连接是活的，直接在这条连接上发 take_control。
// 走不到 ws 的情况（还没连上）落回连接参数，跟首次挂载的行为一致。
watch(
	() => props.autoTakeControl,
	requested => {
		if (!requested || sessionDelivered.value) return
		if (ws?.readyState === WebSocket.OPEN) takeTerminalControl()
		else autoTakeControlPending = true
	},
)

// 缓存复用的两个 Vue 生命周期。挂载那一次不会走 onActivated 之外的路径，
// 所以第一次激活直接跳过，避免和 onMounted 里的初始化重复。
onActivated(() => {
	isActive.value = true
	if (!activatedOnce) {
		activatedOnce = true
		return
	}
	if (intentionalClose && !sessionDelivered.value) {
		intentionalClose = false
		serverErrorShown = false
		receivedServerMessage = false
		connectionFailed.value = false
		reconnecting.value = true
		connectWebSocket()
	}
	void nextTick(() => {
		handleResize()
		void loadRuntimeState()
	})
})

onDeactivated(() => {
	isActive.value = false
})

onBeforeUnmount(() => {
	window.removeEventListener("resize", handleResize)
	resizeObserver?.disconnect()
	if (pingInterval) clearInterval(pingInterval)
	stopRuntimePolling()
	if (reconnectTimer) clearTimeout(reconnectTimer)
	intentionalClose = true
	if (ws) ws.close()
	if (term) term.dispose()
})
</script>

<template>
  <div class="flex h-full min-h-0 w-full flex-col bg-[#1e1e1e]">
    <CodeTerminalStatusBar
      v-if="sessionId && runtimeSupported"
      :runtime-state="runtimeState"
      :runtime-error="runtimeError"
      :native-protocol="nativeProtocol"
      :has-control="hasTerminalControl"
      :reconnecting="reconnecting"
      :connection-failed="connectionFailed"
      @reconnect="reconnectTerminal"
      @take-control="takeTerminalControl"
    />
    <div
      v-if="sessionDelivered"
      class="flex min-h-0 flex-1 items-center justify-center p-6 text-center text-slate-200"
    >
      <div class="max-w-md">
        <div class="text-base font-medium">
          {{ t("code.deliveredSessionTerminalClosed") }}
        </div>
        <div class="mt-2 text-sm text-slate-400">
          {{ t("code.deliveredSessionTerminalHint") }}
        </div>
        <n-button
          class="mt-5"
          type="primary"
          @click="emit('new-session')"
        >
          {{ t("code.createNextSession") }}
        </n-button>
      </div>
    </div>
    <div
      v-else
      ref="terminalRef"
      class="min-h-0 w-full flex-1"
    />
  </div>
</template>

<style scoped>
:deep(.xterm),
:deep(.xterm-screen),
:deep(.xterm-helpers),
:deep(.xterm-viewport) {
	height: 100%;
}

:deep(.xterm) {
	padding: 16px 18px;
}

:deep(.xterm-viewport) {
	overflow-y: auto !important;
}
:deep(.xterm-viewport::-webkit-scrollbar) {
	width: 8px;
}
:deep(.xterm-viewport::-webkit-scrollbar-track) {
	background: #1e1e1e;
}
:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
	background: #424242;
	border-radius: 4px;
}
</style>
