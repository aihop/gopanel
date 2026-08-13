import { onBeforeUnmount, ref } from "vue"
import { useAuthStore } from "@/store/auth"

export interface PipelineLogStreamOptions {
	canReconnect?: () => boolean
	onDisconnected?: () => void
	onFinished?: () => void
	onLog?: (line: string) => void
	reconnectDelay?: number
	truncatedMessage?: string
}

export function usePipelineLogStream(options: PipelineLogStreamOptions = {}) {
	const authStore = useAuthStore()
	const logs = ref<string[]>([])
	const connecting = ref(false)
	const connected = ref(false)
	const streamError = ref(false)
	const currentRecordId = ref(0)
	let eventSource: EventSource | null = null
	let reconnectTimer: number | null = null

	function clearReconnectTimer() {
		if (reconnectTimer !== null) window.clearTimeout(reconnectTimer)
		reconnectTimer = null
	}

	function close() {
		clearReconnectTimer()
		eventSource?.close()
		eventSource = null
		connecting.value = false
		connected.value = false
	}

	function append(line: string) {
		if (logs.value.length >= 3000) {
			logs.value.splice(0, logs.value.length - 1999)
			logs.value.unshift(options.truncatedMessage || "... older logs were collapsed ...")
		}
		logs.value.push(line)
		options.onLog?.(line)
	}

	function scheduleReconnect() {
		clearReconnectTimer()
		if (!currentRecordId.value || options.canReconnect?.() === false) return
		reconnectTimer = window.setTimeout(() => connect(currentRecordId.value), options.reconnectDelay ?? 2000)
	}

	function connect(recordId: number, resetLogs = true) {
		close()
		currentRecordId.value = recordId
		if (resetLogs) logs.value = []
		streamError.value = false
		if (!recordId) return
		connecting.value = true
		const apiUrl = (window as typeof window & { __VITE_API_URL__?: string }).__VITE_API_URL__ || "/api"
		const token = encodeURIComponent(authStore.auth || "")
		eventSource = new EventSource(`${apiUrl}/pipeline/logs?recordId=${recordId}&token=${token}`)
		eventSource.onopen = () => {
			connecting.value = false
			connected.value = true
		}
		eventSource.onmessage = event => {
			if (!event.data || event.data === "ping" || event.data === ":") return
			if (event.data === "EOF" || event.data === '["EOF"]' || event.data === "connection closed") {
				close()
				options.onFinished?.()
				if (options.canReconnect?.() === true) scheduleReconnect()
				return
			}
			append(event.data)
		}
		eventSource.onerror = () => {
			eventSource?.close()
			eventSource = null
			connecting.value = false
			connected.value = false
			streamError.value = true
			options.onDisconnected?.()
			scheduleReconnect()
		}
	}

	onBeforeUnmount(close)

	return { logs, connecting, connected, streamError, connect, close }
}
