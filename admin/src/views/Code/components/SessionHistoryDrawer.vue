<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getAITaskMessages, getCodeExecutionRun, getCodeSessionHistory } from "@/api/modules/code"
import type { AIMessage, CodeExecutionRun } from "@/api/interface/code"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	show: boolean
	sessionId: number | null
	taskId: number | null
}>()

const emit = defineEmits<{
	(event: "update:show", value: boolean): void
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const message = useMessage()
const loading = ref(false)
const detailLoading = ref(false)
const messages = ref<AIMessage[]>([])
const runs = ref<CodeExecutionRun[]>([])
const rawOutput = ref("")
const showRawOutput = ref(false)
const hideExecutorMessages = ref(false)
const expandedMessageIds = ref(new Set<number>())

const messagePreviewMaxCharacters = 800
const messagePreviewMaxLines = 8

const visibleMessages = computed(() =>
	hideExecutorMessages.value ? messages.value.filter(item => item.role === "user") : messages.value
)

const isLongMessage = (content: string) =>
	content.length > messagePreviewMaxCharacters || content.split("\n").length > messagePreviewMaxLines

const isMessageExpanded = (messageId: number) => expandedMessageIds.value.has(messageId)

const messageContent = (item: AIMessage) => {
	if (!isLongMessage(item.content) || isMessageExpanded(item.id)) return item.content
	const lines = item.content.split("\n").slice(0, messagePreviewMaxLines).join("\n")
	const preview = lines.slice(0, messagePreviewMaxCharacters).trimEnd()
	return `${preview}\n…`
}

const toggleMessageExpanded = (messageId: number) => {
	const next = new Set(expandedMessageIds.value)
	if (next.has(messageId)) next.delete(messageId)
	else next.add(messageId)
	expandedMessageIds.value = next
}

const loadHistory = async () => {
	if (!props.sessionId && !props.taskId) return
	loading.value = true
	messages.value = []
	runs.value = []
	expandedMessageIds.value = new Set()
	try {
		if (props.sessionId) {
			// 带上 taskId：同一会话下可能有多个任务，历史应只显示当前任务的对话。
			const response = await getCodeSessionHistory(props.sessionId, props.taskId ?? undefined)
			messages.value = response.data.messages || []
			runs.value = response.data.runs || []
		} else if (props.taskId) {
			const response = await getAITaskMessages(props.taskId)
			messages.value = response.data || []
			runs.value = []
		}
	} catch (error) {
		void 0
	} finally {
		loading.value = false
	}
}

watch(
	() => [props.show, props.sessionId, props.taskId] as const,
	([show]) => {
		if (show) void loadHistory()
	}
)

const showRunRawOutput = async (runId: number) => {
	detailLoading.value = true
	showRawOutput.value = true
	rawOutput.value = ""
	try {
		const response = await getCodeExecutionRun(runId)
		rawOutput.value = response.data.rawOutput || response.data.output || ""
	} catch (error) {
		void 0
	} finally {
		detailLoading.value = false
	}
}

const runForMessage = (runId: number) => runs.value.find(run => run.id === runId)
const runStatusLabel = (status: string) => t(`code.runStatus_${status}`)
</script>

<template>
	<n-drawer :show="show" :width="720" placement="right" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('code.conversationHistory')" closable body-content-style="padding: 16px;">
			<n-spin :show="loading">
				<div class="mb-4 flex items-center justify-end gap-2 border-b border-slate-100 pb-3">
					<span class="text-sm text-slate-500">{{ t("code.hideExecutorMessages") }}</span>
					<n-switch v-model:value="hideExecutorMessages" size="small" />
				</div>
				<n-empty
					v-if="!loading && visibleMessages.length === 0"
					:description="hideExecutorMessages ? t('code.noUserInstructions') : t('code.noConversationHistory')"
				/>
				<div v-else class="space-y-4">
					<div
						v-for="item in visibleMessages"
						:key="item.id"
						class="rounded-2xl border border-slate-200 p-4"
						:class="item.role === 'user' ? 'bg-blue-50' : 'bg-white'"
					>
						<div class="mb-2 flex items-center justify-between gap-3">
							<n-tag size="small" :type="item.role === 'user' ? 'info' : 'success'" :bordered="false">
								{{ item.role === "user" ? t("code.userMessage") : t("code.executorMessage") }}
							</n-tag>
							<span class="text-xs text-slate-400">{{ new Date(item.createdAt).toLocaleString() }}</span>
						</div>
						<pre class="whitespace-pre-wrap break-words font-sans text-sm leading-6 text-slate-700">{{ messageContent(item) }}</pre>
						<n-button
							v-if="isLongMessage(item.content)"
							text
							type="primary"
							size="small"
							class="mt-2"
							:aria-expanded="isMessageExpanded(item.id)"
							@click="toggleMessageExpanded(item.id)"
						>
							{{ isMessageExpanded(item.id) ? t("code.collapseMessage") : t("code.expandMessage") }}
						</n-button>
						<div v-if="item.runId && runForMessage(item.runId)" class="mt-3 flex flex-wrap items-center gap-2">
							<n-tag size="small" :type="runForMessage(item.runId)?.status === 'completed' ? 'success' : 'error'">
								{{ runStatusLabel(runForMessage(item.runId)?.status || "failed") }}
							</n-tag>
							<span class="text-xs text-slate-400">
								{{ t("code.runDuration", { duration: runForMessage(item.runId)?.durationMs || 0 }) }}
							</span>
							<n-button text type="primary" size="small" @click="showRunRawOutput(item.runId)">
								{{ t("code.viewRawOutput") }}
							</n-button>
						</div>
					</div>
				</div>
			</n-spin>
		</n-drawer-content>
	</n-drawer>

	<n-modal v-model:show="showRawOutput" preset="card" style="width: 860px" :title="t('code.rawOutput')">
		<n-spin :show="detailLoading">
			<n-empty v-if="!detailLoading && !rawOutput" :description="t('code.noRawOutput')" />
			<pre v-else class="max-h-[65vh] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-slate-950 p-4 text-xs leading-5 text-slate-100">{{ rawOutput }}</pre>
		</n-spin>
	</n-modal>
</template>
