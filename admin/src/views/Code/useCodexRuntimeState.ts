import type { Ref } from "vue"
import { ref } from "vue"
import { getCodeSession, getCodexRuntimeState } from "@/api/modules/code"
import type { CodexRuntimeState } from "@/api/interface/code"

/**
 * Codex 运行状态轮询。
 *
 * 从 CodeTerminal 拆出来：它跟 xterm / WebSocket 那套没有任何耦合，
 * 走的是普通 HTTP 轮询，混在一个文件里只会把终端组件顶过 500 行门禁。
 *
 * `active` 传 ref 而不是布尔：终端实例会被 KeepAlive 缓存，
 * 切走的实例必须停止轮询（池子里挂 8 个就是 8 份 3 秒轮询），切回来再自己续上。
 */
export function useCodexRuntimeState(sessionId: () => number | null | undefined, active: Ref<boolean>) {
	const runtimeState = ref<CodexRuntimeState | null>(null)
	const runtimeLoading = ref(false)
	const runtimeError = ref(false)
	const runtimeSupported = ref(true)
	const executorId = ref("")
	let pollInterval: ReturnType<typeof setInterval> | null = null

	const loadRuntimeState = async () => {
		if (!active.value) return
		const currentSessionId = sessionId()
		if (!currentSessionId || runtimeLoading.value || !runtimeSupported.value) return
		runtimeLoading.value = true
		try {
			if (!executorId.value) {
				const sessionResponse = await getCodeSession(currentSessionId)
				executorId.value = sessionResponse.data.session.agentName || ""
			}
			// 只有 codex 有运行时状态文件，其它执行器直接关掉这块 UI，别白轮询。
			if (executorId.value !== "codex") {
				runtimeSupported.value = false
				return
			}
			const response = await getCodexRuntimeState(currentSessionId)
			runtimeState.value = response.data
			runtimeSupported.value = response.data !== null
			runtimeError.value = false
		} catch {
			runtimeError.value = true
		} finally {
			runtimeLoading.value = false
		}
	}

	const startRuntimePolling = () => {
		if (pollInterval) return
		void loadRuntimeState()
		pollInterval = setInterval(() => void loadRuntimeState(), 3000)
	}

	const stopRuntimePolling = () => {
		if (!pollInterval) return
		clearInterval(pollInterval)
		pollInterval = null
	}

	/** 会话已统一交付：终端不再连接，这块状态也就没有意义了。 */
	const disableRuntimeState = () => {
		runtimeSupported.value = false
		stopRuntimePolling()
	}

	return {
		runtimeState,
		runtimeError,
		runtimeSupported,
		executorId,
		loadRuntimeState,
		startRuntimePolling,
		stopRuntimePolling,
		disableRuntimeState,
	}
}
