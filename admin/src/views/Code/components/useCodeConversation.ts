import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { createCodeInstruction, getCodeSessionHistory, stopCodeSession } from "@/api/modules/code"
import type { AIMessage, CodeExecutionRun, CodeSession } from "@/api/interface/code"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import type { ComposerAttachment } from "./codeConversationAttachments"
import { attachmentIdentity, serializeInstructionContent } from "./codeConversationAttachments"
import { streamCodeConversation, type ConversationStreamPayload } from "./codeConversationStream"
import {
	conversationRunRunning,
	conversationSessionClosed,
	conversationSessionInitializing,
	visibleConversationThread,
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
	const attachments = ref<ComposerAttachment[]>([])
	const messages = ref<AIMessage[]>([])
	const runs = ref<CodeExecutionRun[]>([])
	const session = ref<CodeSession | null>(null)
	const expandedMessageIds = ref(new Set<number>())
	const streaming = ref<ConversationStreamPayload | null>(null)
	let pollTimer: ReturnType<typeof setInterval> | null = null
	let streamAbort: AbortController | null = null
	let requestVersion = 0

	const closed = computed(() => conversationSessionClosed(session.value?.status))
	const initializing = computed(() => conversationSessionInitializing(session.value?.status))
	const running = computed(() => conversationRunRunning(runs.value))
	const canSend = computed(() => Boolean(sessionId()) && !closed.value && !initializing.value && !sending.value)
	const hasComposerContent = computed(() => Boolean(draft.value.trim() || attachments.value.length))
	const workDir = computed(() => session.value?.workDir || "")
	const displayMessages = computed(() => {
		const items = visibleConversationThread(messages.value)
		const live = streaming.value
		const liveContent = live?.content || ""
		const activeRun = runs.value.find(run => run.status === "running" || run.status === "queued")
		const liveRunId = live?.runId || activeRun?.id || 0
		if (!liveContent && !activeRun) return items
		const index = items.findLastIndex(item => item.role !== "user" && Boolean(liveRunId) && item.runId === liveRunId)
		if (index >= 0) {
			if (!liveContent || (items[index].content || "").length >= liveContent.length) return items
			const next = items.slice()
			next[index] = { ...next[index], content: liveContent }
			return next
		}
		return [
			...items,
			{
				id: -1,
				createdAt: "",
				sessionId: sessionId() || 0,
				taskId: taskId() || 0,
				runId: liveRunId,
				role: "agent",
				content: liveContent,
			},
		]
	})

	const revokeAttachmentPreview = (item: ComposerAttachment) => {
		if (item.previewUrl?.startsWith("blob:")) URL.revokeObjectURL(item.previewUrl)
	}

	const clearAttachments = () => {
		attachments.value.forEach(revokeAttachmentPreview)
		attachments.value = []
	}

	const addAttachments = (items: ComposerAttachment[]) => {
		if (!items.length) return
		const next = [...attachments.value]
		for (const item of items) {
			if (next.some(existing => attachmentIdentity(existing) === attachmentIdentity(item))) {
				revokeAttachmentPreview(item)
				continue
			}
			next.push(item)
		}
		attachments.value = next
	}

	const removeAttachment = (path: string) => {
		attachments.value = attachments.value.filter(item => {
			if (attachmentIdentity(item) !== path && item.path !== path) return true
			revokeAttachmentPreview(item)
			return false
		})
	}

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
		const content = serializeInstructionContent(draft.value, attachments.value)
		if (!id || !content || !canSend.value) return
		sending.value = true
		try {
			const response = await createCodeInstruction(id, content)
			if (response.code !== 0) throw new Error(response.message)
			draft.value = ""
			clearAttachments()
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

	const stopStream = () => {
		streamAbort?.abort()
		streamAbort = null
		streaming.value = null
	}

	const applyStreamPayload = (payload: ConversationStreamPayload, append = false) => {
		const current = streaming.value
		const nextContent = append
			? `${current?.content || ""}${payload.content || ""}`
			: payload.content !== undefined
				? payload.content
				: current?.content || ""
		streaming.value = {
			...current,
			...payload,
			content: nextContent,
		}
	}

	const startStream = () => {
		stopStream()
		const id = sessionId()
		if (!id) return
		const abort = new AbortController()
		streamAbort = abort
		void streamCodeConversation(
			id,
			{
				onSnapshot: payload => applyStreamPayload(payload),
				onDelta: payload => applyStreamPayload(payload, true),
				onDone: payload => {
					applyStreamPayload(payload)
					void loadHistory(true)
				},
			},
			abort.signal,
		)
			.then(() => {
				if (abort.signal.aborted) return
				if (conversationRunRunning(runs.value)) {
					window.setTimeout(() => {
						if (streamAbort === abort) startStream()
					}, 400)
					return
				}
				startPolling()
			})
			.catch(() => {
				if (abort.signal.aborted) return
				startPolling()
				if (!conversationRunRunning(runs.value)) return
				window.setTimeout(() => {
					if (streamAbort === abort) startStream()
				}, 800)
			})
	}

	const startPolling = () => {
		stopPolling()
		pollTimer = setInterval(() => void loadHistory(true), conversationPollMs)
	}

	watch(
		[sessionId, taskId],
		() => {
			expandedMessageIds.value = new Set()
			clearAttachments()
			void loadHistory()
			stopPolling()
			if (sessionId()) startStream()
			else stopStream()
		},
		{ immediate: true },
	)
	onBeforeUnmount(() => {
		stopPolling()
		stopStream()
		clearAttachments()
	})

	return {
		t,
		loading,
		sending,
		stopping,
		loadError,
		draft,
		attachments,
		messages,
		displayMessages,
		streaming,
		runs,
		session,
		expandedMessageIds,
		closed,
		initializing,
		running,
		canSend,
		hasComposerContent,
		workDir,
		loadHistory,
		addAttachments,
		removeAttachment,
		sendInstruction,
		stopExecution,
		toggleMessageExpanded,
	}
}
