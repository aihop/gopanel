import { computed, ref, nextTick, type Ref } from "vue"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { useAuthStore } from "@/store/auth"
import { useI18n } from "vue-i18n"
import { getCodeSession } from "@/api/modules/code"
import { codeTerminalMessages } from "../codeTerminalMessages"
import {
	authoritativeTerminalSize,
	CodeTerminalInputFallback,
	codeTerminalOptions,
	isDeliveredCodeSession,
	isTerminalViewportAtBottom,
	keepingTerminalBottom,
	shouldAutoAcquireTerminalControl,
	shouldAttachOnlyToTerminal,
	terminalWebSocketUrl,
	terminalInputIntent,
	terminalReleaseControlMessage,
	terminalSizeData,
	terminalTakeControlMessage
} from "./codeTerminalSession"

interface UseCodeTerminalConnectionOptions {
	sessionId: Ref<number | null>
	taskId: Ref<number | null>
	autoTakeControl: Ref<boolean>
	onTaskCreated: (taskId: number) => void
	onWriteTerminalData: (data: string, forceBottom?: boolean) => void
	onSyncScrollAnchor: () => void
	onJumpToTerminalBottom: () => void
	onTerminalReady?: (terminal: Terminal) => void
}

export function useCodeTerminalConnection(options: UseCodeTerminalConnectionOptions) {
	const authStore = useAuthStore()
	const { t } = useI18n({ messages: codeTerminalMessages })

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
	let autoTakeControlPending = false
	let forceStart = false
	let connectionErrorObserved = false
	let inputFallback: CodeTerminalInputFallback | null = null

	const isActive = ref(true)
	const nativeProtocol = ref(false)
	const hasTerminalControl = ref(true)
	const reconnecting = ref(false)
	const connectionFailed = ref(false)
	const terminalInactive = ref(false)
	const sessionDelivered = ref(false)

	const markSessionDelivered = () => {
		sessionDelivered.value = true
		hasTerminalControl.value = false
		reconnecting.value = false
		if (pingInterval) clearInterval(pingInterval)
		pingInterval = null
		resizeObserver?.disconnect()
		if (term) term.dispose()
	}

	const refreshDeliveredSession = async () => {
		if (!options.sessionId.value) return false
		try {
			const response = await getCodeSession(options.sessionId.value)
			const session = response.data.session
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
		const shouldFollow = forceBottom || isTerminalViewportAtBottom(buffer.baseY, buffer.viewportY)
		term.write(data, () => {
			if (shouldFollow) term.scrollToBottom()
			options.onSyncScrollAnchor()
		})
	}

	const sendTerminalInput = (data: string) => {
		const intent = terminalInputIntent(
			terminalInactive.value,
			Boolean(ws) && ws?.readyState === WebSocket.OPEN,
			hasTerminalControl.value,
		)
		if (intent === "resume") {
			writeTerminalData(`\r\n\x1b[36m[GoPanel] ${t("code.terminalResumingByInput")}\x1b[0m\r\n`)
			resumeTerminal()
			return
		}
		if (intent === "send") ws?.send(JSON.stringify({ type: "cmd", data }))
	}

	const requestTerminalResync = () => {
		if (ws?.readyState !== WebSocket.OPEN || pendingResyncId) return
		pendingResyncId = `${Date.now()}-${++resyncRequest}`
		ws.send(
			JSON.stringify({
				type: "resync",
				data: JSON.stringify({ sequence: lastSequence, requestId: pendingResyncId })
			})
		)
	}

	const applyAuthoritativeSize = (cols: unknown, rows: unknown) => {
		const size = authoritativeTerminalSize(cols, rows)
		if (!size) return
		if (term.cols === size.cols && term.rows === size.rows) return
		keepingTerminalBottom(term, () => term.resize(size.cols, size.rows))
	}

	const connectWebSocket = () => {
		ws = new WebSocket(
			terminalWebSocketUrl({
				host: window.location.host,
				secure: window.location.protocol === "https:",
				token: authStore.auth || "",
				cols: term.cols,
				rows: term.rows,
				sessionId: options.sessionId.value,
				taskId: options.taskId.value,
				attachOnly: shouldAttachOnlyToTerminal(options.taskId.value, forceStart),
				afterSequence: lastSequence,
				takeControl: autoTakeControlPending,
			}),
		)

		ws.onopen = () => {
			reconnecting.value = false
			connectionErrorObserved = false
			if (options.sessionId.value) {
				term.writeln(`\x1b[32m[GoPanel] ${t("code.openingSession", { id: options.sessionId.value })}\x1b[0m\r\n`)
			} else if (options.taskId.value) {
				term.writeln(`\x1b[32m[GoPanel] 正在恢复历史任务 #${options.taskId.value} 的上下文...\x1b[0m\r\n`)
			}
		}

		ws.onmessage = event => {
			receivedServerMessage = true
			initialReconnectAttempts = 0
			try {
				const msg = JSON.parse(event.data)
				if (msg.type === "baseline") {
					nativeProtocol.value = true
					connectionFailed.value = false
					terminalInactive.value = false
					forceStart = false
					if (pendingResyncId && msg.requestId !== pendingResyncId) return
					applyAuthoritativeSize(msg.cols, msg.rows)
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
						if (hasTerminalControl.value) {
							void nextTick(() => {
								handleResize()
								term.focus()
							})
						} else if (shouldAutoAcquireTerminalControl(false, msg.controlReason, msg.leaseExpiresAt)) {
							void nextTick(takeTerminalControl)
						}
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
					applyAuthoritativeSize(msg.cols, msg.rows)
					const gainedControl = !hasTerminalControl.value && Boolean(msg.hasControl)
					hasTerminalControl.value = Boolean(msg.hasControl)
					if (gainedControl) void nextTick(handleResize)
					else if (shouldAutoAcquireTerminalControl(Boolean(msg.hasControl), msg.controlReason, msg.leaseExpiresAt)) {
						void nextTick(takeTerminalControl)
					}
					if (msg.controlReason) term.writeln(`\r\n\x1b[33m[GoPanel] ${t("code.terminalControlBusy")}\x1b[0m`)
				} else if (msg.type === "closed") {
					intentionalClose = true
					hasTerminalControl.value = false
					lastSequence = 0
				} else if (msg.type === "inactive") {
					intentionalClose = true
					serverErrorShown = true
					terminalInactive.value = true
					hasTerminalControl.value = false
					writeTerminalData(`\r\n\x1b[36m[GoPanel] ${t("code.terminalSessionInactive")}\x1b[0m\r\n`)
				} else if (msg.type === "error") {
					serverErrorShown = true
					hasTerminalControl.value = false
					if (msg.code === "workspace_busy") {
						intentionalClose = true
						forceStart = false
						terminalInactive.value = true
						connectionFailed.value = false
						writeTerminalData(`\r\n\x1b[33m[GoPanel] ${t("code.terminalWorkspaceBusy")}\x1b[0m\r\n`)
					} else {
						connectionFailed.value = true
						writeTerminalData(`\r\n\x1b[31m[GoPanel] ${t("code.terminalStartFailed")}\x1b[0m\r\n`)
					}
				} else if (msg.type === "cmd") {
					writeTerminalData(msg.data)
				} else if (msg.type === "meta" && msg.task_id) {
					options.onTaskCreated(msg.task_id)
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
			connectionErrorObserved = true
		}

		ws.onclose = () => {
			pendingResyncId = ""
			if (intentionalClose) {
				return
			}
			if (!serverErrorShown) {
				const messageKey = connectionErrorObserved ? "code.terminalWebSocketFailed" : "code.terminalDisconnected"
				writeTerminalData(`\r\n\x1b[33m[GoPanel] ${t(messageKey)}\x1b[0m\r\n`)
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
		terminalInactive.value = false
		connectionErrorObserved = false
		reconnecting.value = true
		connectWebSocket()
	}

	const resumeTerminal = () => {
		if (ws && ws.readyState !== WebSocket.CLOSED) {
			ws.onclose = null
			ws.close()
		}
		forceStart = true
		intentionalClose = false
		terminalInactive.value = false
		serverErrorShown = false
		receivedServerMessage = false
		connectionFailed.value = false
		reconnecting.value = true
		connectWebSocket()
	}

	const takeTerminalControl = () => {
		if (ws && ws.readyState === WebSocket.OPEN) {
			const dimensions = fitAddon.proposeDimensions() || { cols: term.cols, rows: term.rows }
			ws.send(terminalTakeControlMessage(dimensions.cols, dimensions.rows))
			term.focus()
		}
	}

	const disconnectTerminal = (releaseControl: boolean) => {
		if (reconnectTimer) {
			clearTimeout(reconnectTimer)
			reconnectTimer = null
		}
		intentionalClose = true
		reconnecting.value = false
		hasTerminalControl.value = false
		if (!ws || (ws.readyState !== WebSocket.OPEN && ws.readyState !== WebSocket.CONNECTING)) return
		if (releaseControl && ws.readyState === WebSocket.OPEN) ws.send(terminalReleaseControlMessage())
		ws.onclose = null
		ws.onerror = null
		ws.onmessage = null
		ws.close()
	}

	const handleResize = () => {
		if (!fitAddon) return
		if (!isActive.value) return
		const element = term.element
		if (!element || element.clientWidth === 0 || element.clientHeight === 0) return
		if (nativeProtocol.value && !hasTerminalControl.value) return
		try {
			keepingTerminalBottom(term, () => fitAddon.fit())
			if (ws && ws.readyState === WebSocket.OPEN && (!nativeProtocol.value || hasTerminalControl.value)) {
				ws.send(JSON.stringify({ type: "resize", data: terminalSizeData(term.cols, term.rows) }))
			}
		} catch (e) {
			console.warn("Fit addon resize error", e)
		}
	}

	const initTerminal = (terminalRef: Ref<HTMLElement | null>) => {
		if (!terminalRef.value) return

		term = new Terminal(codeTerminalOptions())

		fitAddon = new FitAddon()
		term.loadAddon(fitAddon)
		term.open(terminalRef.value)
		term.onScroll(options.onSyncScrollAnchor)
		options.onTerminalReady?.(term)
		if (term.textarea) {
			inputFallback = new CodeTerminalInputFallback()
			term.textarea.addEventListener("compositionstart", () => inputFallback?.startComposition())
			term.textarea.addEventListener("compositionend", () => inputFallback?.endComposition())
			term.textarea.addEventListener("input", event => {
				inputFallback?.queueInput(event as InputEvent, sendTerminalInput)
			})
		}

		nextTick(() => {
			try {
				fitAddon.fit()
			} catch (e) {
				console.warn("Fit addon error on init", e)
			}
			connectWebSocket()
		})

		term.onData(data => {
			inputFallback?.recordTerminalData(data)
			sendTerminalInput(data)
		})

		pingInterval = setInterval(() => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: "ping" }))
			}
		}, 15000)
	}

	const cleanup = () => {
		if (reconnectTimer) {
			clearTimeout(reconnectTimer)
			reconnectTimer = null
		}
		if (pingInterval) clearInterval(pingInterval)
		resizeObserver?.disconnect()
		disconnectTerminal(true)
		inputFallback?.dispose()
		if (term) term.dispose()
	}

	const updateAutoTakeControl = (requested: boolean) => {
		autoTakeControlPending = requested
		if (!requested || sessionDelivered.value) return
		if (ws?.readyState === WebSocket.OPEN) takeTerminalControl()
	}

	const activate = () => {
		isActive.value = true
		if (intentionalClose && !sessionDelivered.value && !terminalInactive.value) {
			intentionalClose = false
			autoTakeControlPending = true
			serverErrorShown = false
			receivedServerMessage = false
			connectionFailed.value = false
			reconnecting.value = true
			connectWebSocket()
		}
	}

	const deactivate = () => {
		isActive.value = false
		disconnectTerminal(true)
	}

	const setupResizeObserver = (terminalRef: Ref<HTMLElement | null>) => {
		if (terminalRef.value && typeof ResizeObserver !== "undefined") {
			resizeObserver = new ResizeObserver(() => {
				handleResize()
			})
			resizeObserver.observe(terminalRef.value)
		}
	}

	return {
		isActive,
		nativeProtocol,
		hasTerminalControl,
		reconnecting,
		connectionFailed,
		terminalInactive,
		sessionDelivered,
		initTerminal,
		reconnectTerminal,
		resumeTerminal,
		takeTerminalControl,
		disconnectTerminal,
		handleResize,
		cleanup,
		updateAutoTakeControl,
		activate,
		deactivate,
		setupResizeObserver,
		refreshDeliveredSession,
		sendTerminalInput,
	}
}
