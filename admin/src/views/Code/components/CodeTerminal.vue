<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount, nextTick } from "vue"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import "xterm/css/xterm.css"
import { useAuthStore } from "@/store/auth"
import { useI18n } from "vue-i18n"
import { getCodeSession, getCodexRuntimeState } from "@/api/modules/code"
import type { CodexRuntimeState } from "@/api/interface/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const authStore = useAuthStore()
const { t } = useI18n({ messages: codeProjectMessages })

const props = defineProps<{
	taskId: number | null
	sessionId?: number | null
	autoTakeControl?: boolean
}>()

const emit = defineEmits<{
	(e: "task-created", taskId: number): void
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
const nativeProtocol = ref(false)
const hasTerminalControl = ref(true)
const reconnecting = ref(false)
const connectionFailed = ref(false)
const runtimeState = ref<CodexRuntimeState | null>(null)
const runtimeLoading = ref(false)
const runtimeError = ref(false)
const runtimeSupported = ref(true)
const executorId = ref("")
let runtimePollInterval: ReturnType<typeof setInterval> | null = null

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
		data: JSON.stringify({ sequence: lastSequence, requestId: pendingResyncId })
	}))
}

const runtimeTagType = computed(() => {
	if (runtimeState.value?.responseState === "failed") return "error"
	if (runtimeState.value?.responseState === "needsInput") return "warning"
	if (runtimeState.value?.responseState === "completed") return "success"
	return "info"
})

const formatTokens = (count: number) => new Intl.NumberFormat().format(count)

const loadRuntimeState = async () => {
	if (!props.sessionId || runtimeLoading.value || !runtimeSupported.value) return
	runtimeLoading.value = true
	try {
		if (!executorId.value) {
			const sessionResponse = await getCodeSession(props.sessionId)
			executorId.value = sessionResponse.data.session.agentName || ""
		}
		if (executorId.value !== "codex") {
			runtimeSupported.value = false
			return
		}
		const response = await getCodexRuntimeState(props.sessionId)
		runtimeState.value = response.data
		runtimeSupported.value = response.data !== null
		runtimeError.value = false
	} catch {
		runtimeError.value = true
	} finally {
		runtimeLoading.value = false
	}
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
			reconnectTimer = setTimeout(() => {
				reconnectTimer = null
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
	if (fitAddon) {
		try {
			fitAddon.fit()
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: "resize", data: JSON.stringify({ cols: term.cols, rows: term.rows }) }))
			}
		} catch (e) {
			console.warn("Fit addon resize error", e)
		}
	}
}

onMounted(() => {
	initTerminal()
	void loadRuntimeState()
	runtimePollInterval = setInterval(() => void loadRuntimeState(), 3000)
	window.addEventListener("resize", handleResize)
	if (terminalRef.value && typeof ResizeObserver !== "undefined") {
		resizeObserver = new ResizeObserver(() => {
			handleResize()
		})
		resizeObserver.observe(terminalRef.value)
	}
})

onBeforeUnmount(() => {
	window.removeEventListener("resize", handleResize)
	resizeObserver?.disconnect()
	if (pingInterval) clearInterval(pingInterval)
	if (runtimePollInterval) clearInterval(runtimePollInterval)
	if (reconnectTimer) clearTimeout(reconnectTimer)
	intentionalClose = true
	if (ws) ws.close()
	if (term) term.dispose()
})
</script>

<template>
	<div class="flex h-full min-h-0 w-full flex-col bg-[#1e1e1e]">
		<div
			v-if="sessionId && runtimeSupported"
			class="flex min-h-12 items-center justify-between gap-3 border-b border-slate-700 bg-slate-900 px-4 py-2 text-slate-300"
		>
			<div class="flex min-w-0 items-center gap-3">
				<n-tag v-if="runtimeState" :type="runtimeTagType" size="small" round :bordered="false">
					{{ t(`code.codexState_${runtimeState.responseState}`) }}
				</n-tag>
				<span v-else class="text-xs text-slate-400">
					{{ t(runtimeError ? "code.codexRuntimeUnavailable" : "code.codexRuntimeStarting") }}
				</span>
				<span v-if="runtimeState?.awaitingApproval" class="truncate text-xs font-medium text-amber-300">
					{{ t("code.codexApprovalHint") }}
				</span>
				<span v-else-if="runtimeState?.lastAssistantPreview" class="truncate text-xs text-slate-400">
					{{ runtimeState.lastAssistantPreview }}
				</span>
			</div>
			<div class="flex shrink-0 items-center gap-3 text-xs text-slate-400">
				<n-button v-if="connectionFailed && !reconnecting" size="tiny" type="warning" @click="reconnectTerminal">
					{{ t("code.reconnectTerminal") }}
				</n-button>
				<n-tag v-else-if="reconnecting" size="small" type="warning" :bordered="false">
					{{ t("code.terminalReconnecting") }}
				</n-tag>
				<n-tag v-else-if="nativeProtocol && hasTerminalControl" size="small" type="success" :bordered="false">
					{{ t("code.terminalControlling") }}
				</n-tag>
				<n-button v-else-if="nativeProtocol" size="tiny" type="warning" @click="takeTerminalControl">
					{{ t("code.takeTerminalControl") }}
				</n-button>
				<template v-if="runtimeState">
					<span v-if="runtimeState.model">{{ runtimeState.model }}</span>
					<span v-if="runtimeState.totalTokens">
						{{ t("code.codexTokenUsage", { count: formatTokens(runtimeState.totalTokens) }) }}
					</span>
					<span v-if="runtimeState.cachedInputTokens" class="hidden md:inline">
						{{ t("code.codexCachedTokens", { count: formatTokens(runtimeState.cachedInputTokens) }) }}
					</span>
				</template>
			</div>
		</div>
		<div ref="terminalRef" class="min-h-0 w-full flex-1"></div>
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
