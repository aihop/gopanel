<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import {
	approveCodeInstruction,
	createCodeInstruction,
	getCodeSessionState,
	rejectCodeInstruction,
	retryCodeInstruction,
	stopCodeSession
} from "@/api/modules/code"
import type { CodeSessionState } from "@/api/interface/code"

const props = defineProps<{ sessionId: number }>()
const emit = defineEmits<{
	(event: "task-created", taskId: number): void
	(event: "show-history"): void
}>()

const { t } = useI18n()
const message = useMessage()
const state = ref<CodeSessionState | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const loadError = ref("")
const prompt = ref("")
let pollTimer: ReturnType<typeof setInterval> | null = null
let loadingSessionId: number | null = null

const isRunning = computed(
	() => state.value?.currentStage === "executing" || state.value?.latestRun?.status === "running"
)
const canRetry = computed(() => ["failed", "cancelled"].includes(state.value?.latestInstruction?.status || ""))
const stageType = computed(() => {
	if (state.value?.currentStage === "failed") return "error"
	if (state.value?.currentStage === "awaiting_approval") return "warning"
	if (["completed", "preview_ready"].includes(state.value?.currentStage || "")) return "success"
	return "info"
})

const loadState = async (silent = false) => {
	const sessionId = props.sessionId
	if (loadingSessionId === sessionId) return
	loadingSessionId = sessionId
	if (!silent) loading.value = true
	try {
		const response = await getCodeSessionState(sessionId)
		if (props.sessionId !== sessionId) return
		state.value = response.data
		loadError.value = ""
		if (response.data.currentTask?.id) emit("task-created", response.data.currentTask.id)
	} catch (error) {
		if (props.sessionId !== sessionId) return
		loadError.value = error instanceof Error ? error.message : t("code.stateLoadFailed")
	} finally {
		if (loadingSessionId === sessionId) loadingSessionId = null
		if (!silent && props.sessionId === sessionId) loading.value = false
	}
}

const startPolling = () => {
	if (pollTimer) clearInterval(pollTimer)
	pollTimer = setInterval(() => void loadState(true), 2000)
}

watch(
	() => props.sessionId,
	() => {
		state.value = null
		prompt.value = ""
		void loadState()
		startPolling()
	},
	{ immediate: true }
)

onBeforeUnmount(() => {
	if (pollTimer) clearInterval(pollTimer)
})

const submitPrompt = async () => {
	const content = prompt.value.trim()
	if (!content || actionLoading.value) return
	actionLoading.value = true
	try {
		const response = await createCodeInstruction(props.sessionId, content)
		prompt.value = ""
		emit("task-created", response.data.task.id)
		message.success(response.data.approval ? t("code.approvalRequired") : t("code.instructionQueued"))
		await loadState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.instructionSubmitFailed"))
	} finally {
		actionLoading.value = false
	}
}

const decideApproval = async (approved: boolean) => {
	const approvalId = state.value?.pendingApproval?.id
	if (!approvalId) return
	actionLoading.value = true
	try {
		if (approved) await approveCodeInstruction(approvalId)
		else await rejectCodeInstruction(approvalId)
		message.success(t(approved ? "code.approvalApproved" : "code.approvalRejected"))
		await loadState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.approvalFailed"))
	} finally {
		actionLoading.value = false
	}
}

const stopExecution = async () => {
	actionLoading.value = true
	try {
		await stopCodeSession(props.sessionId)
		message.success(t("code.stopRequested"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.stopFailed"))
	} finally {
		actionLoading.value = false
	}
}

const retryInstruction = async () => {
	const instructionId = state.value?.latestInstruction?.id
	if (!instructionId) return
	actionLoading.value = true
	try {
		await retryCodeInstruction(instructionId)
		message.success(t("code.retryQueued"))
		await loadState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.retryFailed"))
	} finally {
		actionLoading.value = false
	}
}
</script>

<template>
	<div class="flex h-full min-h-0 flex-col bg-slate-50">
		<div class="flex items-center justify-between border-b border-slate-200 bg-white px-5 py-3">
			<div class="flex min-w-0 items-center gap-3">
				<n-tag :type="stageType" round :bordered="false">
					{{ t(`code.stage_${state?.currentStage || 'idle'}`) }}
				</n-tag>
				<span class="truncate text-sm text-slate-500">{{ state?.session.agentName }}</span>
			</div>
			<div class="flex items-center gap-2">
				<n-button v-if="isRunning" size="small" type="error" secondary :loading="actionLoading" @click="stopExecution">
					{{ t("code.stopExecution") }}
				</n-button>
				<n-button v-if="canRetry" size="small" secondary :loading="actionLoading" @click="retryInstruction">
					{{ t("code.retryExecution") }}
				</n-button>
				<n-button size="small" quaternary @click="emit('show-history')">{{ t("code.rawAndHistory") }}</n-button>
			</div>
		</div>

		<n-spin :show="loading" class="min-h-0 flex-1">
			<n-scrollbar class="h-full px-5 py-5">
				<n-alert v-if="loadError" type="error" class="mb-4" :title="t('code.stateLoadFailed')">
					<div class="flex items-center justify-between gap-3">
						<span>{{ loadError }}</span>
						<n-button text type="primary" @click="loadState()">{{ t("code.retry") }}</n-button>
					</div>
				</n-alert>

				<div v-if="state?.recentMessages.length" class="mx-auto max-w-4xl space-y-4">
					<div v-for="item in state.recentMessages" :key="item.id" class="flex" :class="item.role === 'user' ? 'justify-end' : 'justify-start'">
						<div class="max-w-[88%] rounded-2xl px-4 py-3 shadow-sm" :class="item.role === 'user' ? 'bg-blue-600 text-white' : 'border border-slate-200 bg-white text-slate-700'">
							<div class="mb-1 text-xs opacity-70">{{ item.role === "user" ? t("code.userMessage") : t("code.executorMessage") }}</div>
							<pre class="whitespace-pre-wrap break-words font-sans text-sm leading-6">{{ item.content }}</pre>
						</div>
					</div>
				</div>
				<n-empty v-else-if="!loading && !loadError" :description="t('code.conversationEmpty')" class="py-20" />

				<div v-if="state?.pendingApproval" class="mx-auto mt-5 max-w-4xl rounded-2xl border border-amber-200 bg-amber-50 p-4">
					<div class="font-semibold text-amber-900">{{ t("code.riskApproval") }}</div>
					<div class="mt-2 whitespace-pre-wrap text-sm text-amber-800">{{ state.pendingApproval.content }}</div>
					<div class="mt-4 flex gap-2">
						<n-button type="warning" :loading="actionLoading" @click="decideApproval(true)">{{ t("code.approveExecution") }}</n-button>
						<n-button :disabled="actionLoading" @click="decideApproval(false)">{{ t("code.rejectExecution") }}</n-button>
					</div>
				</div>

				<div v-if="state?.errorSummary" class="mx-auto mt-5 max-w-4xl">
					<n-alert type="error" :title="t('code.executionError')">{{ state.errorSummary }}</n-alert>
				</div>
				<div v-if="state?.changedFiles.length" class="mx-auto mt-5 max-w-4xl rounded-2xl border border-slate-200 bg-white p-4">
					<div class="mb-3 text-sm font-semibold text-slate-700">{{ t("code.changedFiles") }}</div>
					<div class="flex flex-wrap gap-2"><n-tag v-for="file in state.changedFiles" :key="file" size="small">{{ file }}</n-tag></div>
				</div>
				<div v-if="state?.previews.length" class="mx-auto mt-5 max-w-4xl rounded-2xl border border-slate-200 bg-white p-4">
					<div class="mb-3 text-sm font-semibold text-slate-700">{{ t("code.previewLinks") }}</div>
					<a v-for="preview in state.previews" :key="preview.id" :href="preview.url" target="_blank" class="mr-3 text-sm text-blue-600 hover:underline">{{ preview.title }}</a>
				</div>
			</n-scrollbar>
		</n-spin>

		<div class="border-t border-slate-200 bg-white p-4">
			<div class="mx-auto flex max-w-4xl items-end gap-3 rounded-2xl border border-slate-200 bg-slate-50 p-3 focus-within:border-blue-300">
				<n-input v-model:value="prompt" type="textarea" :placeholder="t('code.promptPlaceholder')" :autosize="{ minRows: 2, maxRows: 6 }" :bordered="false" :disabled="actionLoading" @keydown.ctrl.enter.prevent="submitPrompt" @keydown.meta.enter.prevent="submitPrompt" />
				<n-button type="primary" :loading="actionLoading" :disabled="!prompt.trim()" @click="submitPrompt">{{ t("code.sendInstruction") }}</n-button>
			</div>
			<div class="mx-auto mt-2 max-w-4xl text-xs text-slate-400">{{ t("code.promptHint") }}</div>
		</div>
	</div>
</template>
