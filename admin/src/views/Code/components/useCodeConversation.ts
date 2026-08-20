import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { createCodeInstruction, getCodeSessionHistory, stopCodeSession } from "@/api/modules/code"
import type { AIMessage, CodeExecutionRun, CodeSession } from "@/api/interface/code"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import {
	conversationRunRunning,
	conversationSessionClosed,
	conversationSessionInitializing,
} from "./codeConversationThread"

const conversationPollMs = 3000

export function useCodeConversation(sessionId: () => number | null, taskId: () => number | null) {
	const { t } = useI18n({ messages: codeWorkspaceMessages })
	const toast = useMessage()
	const loading = ref(false)
	const sending = ref(false)
	const stopping = ref(false)
	const loadError = ref(false)
	const draft = ref("")
	const messages = ref<AIMessage[]>([])
	const runs = ref<CodeExecutionRun[]>([])
	const session = ref<CodeSession | null>(null)
	const expandedMessageIds = ref(new Set<number>())
	let pollTimer: ReturnType<typeof setInterval> | null = null
	let requestVersion = 0

	const closed = computed(() => conversationSessionClosed(session.value?.status))
	const initializing = computed(() => conversationSessionInitializing(session.value?.status))
	const running = computed(() => conversationRunRunning(runs.value))
	const canSend = computed(() => Boolean(sessionId()) && !closed.value && !initializing.value && !sending.value)

	const loadHistory = async (silent = false) => {
		const id = sessionId()
		if (!id) {
			messages.value = []
			runs.value = []
			session.value = null
			return
		}
		const version = ++requestVersion
		if (!silent) loading.value = true
		try {
			const response = await getCodeSessionHistory(id, taskId() ?? undefined)
			if (version !== requestVersion) return
			messages.value = response.data.messages || []
			runs.value = response.data.runs || []
			session.value = response.data.session || null
			loadError.value = false
		} catch {
			if (version !== requestVersion) return
			loadError.value = true
			if (!silent) toast.error(t("code.historyLoadFailed"))
		} finally {
			if (version === requestVersion) loading.value = false
		}
	}

	const sendInstruction = async () => {
		const id = sessionId()
		const content = draft.value.trim()
		if (!id || !content || !canSend.value) return
		sending.value = true
		try {
			const response = await createCodeInstruction(id, content)
			if (response.code !== 0) throw new Error(response.message)
			draft.value = ""
			await loadHistory(true)
			return response.data.task?.id || 0
		} catch (error) {
			toast.error(error instanceof Error && error.message ? error.message : t("code.instructionSubmitFailed"))
			return 0
		} finally {
			sending.value = false
		}
	}

	const stopExecution = async () => {
		const id = sessionId()
		if (!id || stopping.value) return
		stopping.value = true
		try {
			await stopCodeSession(id)
			toast.success(t("code.stopRequested"))
			await loadHistory(true)
		} catch {
			toast.error(t("code.stopFailed"))
		} finally {
			stopping.value = false
		}
	}

	const toggleMessageExpanded = (messageId: number) => {
		const next = new Set(expandedMessageIds.value)
		if (next.has(messageId)) next.delete(messageId)
		else next.add(messageId)
		expandedMessageIds.value = next
	}

	const stopPolling = () => {
		if (!pollTimer) return
		clearInterval(pollTimer)
		pollTimer = null
	}

	const startPolling = () => {
		stopPolling()
		pollTimer = setInterval(() => void loadHistory(true), conversationPollMs)
	}

	watch(
		[sessionId, taskId],
		() => {
			expandedMessageIds.value = new Set()
			void loadHistory()
			if (sessionId()) startPolling()
			else stopPolling()
		},
		{ immediate: true },
	)
	onBeforeUnmount(stopPolling)

	return {
		t,
		loading,
		sending,
		stopping,
		loadError,
		draft,
		messages,
		runs,
		session,
		expandedMessageIds,
		closed,
		initializing,
		running,
		canSend,
		loadHistory,
		sendInstruction,
		stopExecution,
		toggleMessageExpanded,
	}
}
