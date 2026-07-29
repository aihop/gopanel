<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getAITaskMessages, getCodeExecutionRun, getCodeSessionHistory } from "@/api/modules/code"
import type { AIMessage, CodeExecutionRun } from "@/api/interface/code"

const props = defineProps<{
	show: boolean
	sessionId: number | null
	taskId: number | null
}>()

const emit = defineEmits<{
	(event: "update:show", value: boolean): void
}>()

const { t } = useI18n()
const message = useMessage()
const loading = ref(false)
const detailLoading = ref(false)
const messages = ref<AIMessage[]>([])
const runs = ref<CodeExecutionRun[]>([])
const rawOutput = ref("")
const showRawOutput = ref(false)

const loadHistory = async () => {
	if (!props.sessionId && !props.taskId) return
	loading.value = true
	messages.value = []
	runs.value = []
	try {
		if (props.sessionId) {
			const response = await getCodeSessionHistory(props.sessionId)
			messages.value = response.data.messages || []
			runs.value = response.data.runs || []
		} else if (props.taskId) {
			const response = await getAITaskMessages(props.taskId)
			messages.value = response.data || []
			runs.value = []
		}
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.historyLoadFailed"))
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
		message.error(error instanceof Error ? error.message : t("code.historyLoadFailed"))
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
				<n-empty v-if="!loading && messages.length === 0" :description="t('code.noConversationHistory')" />
				<div v-else class="space-y-4">
					<div
						v-for="item in messages"
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
						<pre class="whitespace-pre-wrap break-words font-sans text-sm leading-6 text-slate-700">{{ item.content }}</pre>
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
