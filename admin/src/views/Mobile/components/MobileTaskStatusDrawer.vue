<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeSessionState } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"

const props = defineProps<{ show: boolean; state: CodeSessionState | null; loading: boolean }>()
const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "approve", reason: string): void
	(event: "reject", reason: string): void
	(event: "stop"): void
	(event: "retry"): void
}>()
const { t } = useI18n({ messages: mobileMessages })
const approvalReason = ref("")
const canRetry = computed(() => ["failed", "cancelled"].includes(props.state?.latestInstruction?.status || ""))
const isRunning = computed(() => props.state?.currentStage === "executing" || props.state?.latestRun?.status === "running")
const knownStages = ["idle", "interactive", "task_ready", "instruction_queued", "awaiting_approval", "executing", "completed", "preview_ready", "failed", "cancelled", "approval_rejected"]
const stageLabel = computed(() => {
	const stage = props.state?.currentStage || "idle"
	return t(`mobile.stage_${knownStages.includes(stage) ? stage : "unknown"}`)
})
const tokenLabel = (count: number) => new Intl.NumberFormat().format(count || 0)

function previewCanOpen(status: string) {
	return status === "ready"
}

watch(() => props.state?.pendingApproval?.id, () => { approvalReason.value = "" })
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="min(78dvh, 720px)" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('mobile.taskStatus')" closable>
			<n-spin :show="loading">
				<div class="space-y-4">
					<n-alert v-if="state?.pendingApproval" type="warning" :title="state.pendingApproval.title">
						<div class="whitespace-pre-wrap text-sm">{{ state.pendingApproval.content }}</div>
						<n-input v-model:value="approvalReason" class="mt-3" :placeholder="t('mobile.approvalReason')" />
						<div class="mt-3 flex gap-2"><n-button type="warning" :loading="loading" @click="emit('approve', approvalReason)">{{ t("mobile.approve") }}</n-button><n-button :disabled="loading" @click="emit('reject', approvalReason)">{{ t("mobile.reject") }}</n-button></div>
					</n-alert>
					<n-alert v-if="state?.errorSummary" type="error">{{ state.errorSummary }}</n-alert>
					<section class="rounded-2xl border border-slate-200 bg-white p-4">
						<div class="flex items-center justify-between"><strong>{{ t("mobile.latestResult") }}</strong><n-tag size="small" :bordered="false">{{ stageLabel }}</n-tag></div>
						<p class="mt-3 whitespace-pre-wrap text-sm leading-6 text-slate-600">{{ state?.recentOutput || t("mobile.noResult") }}</p>
						<div v-if="state?.latestRun" class="mt-3 flex flex-wrap gap-3 text-xs text-slate-400"><span>{{ state.latestRun.executorId }}</span><span>{{ Math.round(state.latestRun.durationMs / 1000) }}s</span><span>{{ t("mobile.exitCode", { code: state.latestRun.exitCode }) }}</span></div>
					</section>
					<section v-if="state?.tokenUsage.project.runs" class="rounded-2xl border border-slate-200 bg-white p-4"><strong>{{ t("mobile.tokenUsage") }}</strong><div class="mt-3 grid grid-cols-3 gap-2 text-center"><div class="rounded-xl bg-slate-50 p-2"><div class="font-semibold">{{ tokenLabel(state.latestRun?.totalTokens || 0) }}</div><div class="text-[11px] text-slate-500">{{ t("mobile.latestRunTokens") }}</div></div><div class="rounded-xl bg-blue-50 p-2"><div class="font-semibold text-blue-700">{{ tokenLabel(state.tokenUsage.session.totalTokens) }}</div><div class="text-[11px] text-slate-500">{{ t("mobile.sessionTokens") }}</div></div><div class="rounded-xl bg-emerald-50 p-2"><div class="font-semibold text-emerald-700">{{ tokenLabel(state.tokenUsage.project.totalTokens) }}</div><div class="text-[11px] text-slate-500">{{ t("mobile.projectTokens") }}</div></div></div></section>
					<n-alert v-if="state && !state.tokenUsage.budget.unlimited" :type="state.tokenUsage.budget.exceeded ? 'error' : 'info'" :show-icon="false">{{ state.tokenUsage.budget.exceeded ? t("mobile.tokenBudgetExceeded") : t("mobile.tokenBudget", { percent: Math.round(state.tokenUsage.budget.usagePercent) }) }}</n-alert>
					<section v-if="state?.changedFiles.length" class="rounded-2xl border border-slate-200 bg-white p-4"><strong>{{ t("mobile.changedFiles") }}</strong><div class="mt-3 space-y-2"><div v-for="file in state.changedFiles" :key="file" class="flex items-center gap-2 text-sm text-slate-600"><Icon name="mdi:file-code-outline" :size="17" /><span class="truncate">{{ file }}</span></div></div></section>
					<section v-if="state?.previews.length" class="rounded-2xl border border-slate-200 bg-white p-4"><strong>{{ t("mobile.previews") }}</strong><a v-for="preview in state.previews" :key="preview.id" :href="previewCanOpen(preview.status) ? preview.url : undefined" :target="previewCanOpen(preview.status) ? '_blank' : undefined" rel="noopener noreferrer" class="mt-3 flex w-full items-center gap-3 rounded-xl bg-slate-50 p-3 text-left" :class="previewCanOpen(preview.status) ? '' : 'cursor-not-allowed opacity-60'"><Icon :name="previewCanOpen(preview.status) ? 'mdi:open-in-new' : 'mdi:link-off'" :size="18" /><span class="min-w-0 flex-1 truncate text-sm">{{ preview.title }}</span><n-tag size="small" :bordered="false">{{ preview.status }}</n-tag></a><div v-if="state.previews.some(preview => !previewCanOpen(preview.status))" class="mt-2 text-xs text-slate-500">{{ t("mobile.previewUnavailableHint") }}</div></section>
					<section v-if="state?.timelineEvents.length" class="rounded-2xl border border-slate-200 bg-white p-4"><strong>{{ t("mobile.timeline") }}</strong><div class="mt-4 space-y-4"><div v-for="event in state.timelineEvents" :key="event.id" class="border-l-2 border-blue-200 pl-3"><div class="text-sm font-medium text-slate-800">{{ event.title }}</div><div v-if="event.content" class="mt-1 whitespace-pre-wrap text-xs leading-5 text-slate-500">{{ event.content }}</div></div></div></section>
				</div>
			</n-spin>
			<template #footer><div class="flex w-full gap-2"><n-button v-if="isRunning" type="error" secondary :loading="loading" @click="emit('stop')">{{ t("mobile.stop") }}</n-button><n-button v-if="canRetry" secondary :loading="loading" @click="emit('retry')">{{ t("mobile.retryExecution") }}</n-button></div></template>
		</n-drawer-content>
	</n-drawer>
</template>
