<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeSession, CodeSessionState } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import { mobileTaskDeliveryMessages } from "../mobileTaskDeliveryMessages"
import MobileGitReviewDrawer from "./MobileGitReviewDrawer.vue"
import CodeDeliveryFacts from "@/views/Code/components/CodeDeliveryFacts.vue"

const props = defineProps<{
	show: boolean
	session: CodeSession | null
	state: CodeSessionState | null
	loading: boolean
}>()
const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "approve", reason: string): void
	(event: "reject", reason: string): void
	(event: "stop"): void
	(event: "retry"): void
	(event: "deliveryUpdated"): void
}>()
const taskStatusMessages = {
	zh: { mobile: { ...mobileMessages.zh.mobile, ...mobileTaskDeliveryMessages.zh.mobile } },
	en: { mobile: { ...mobileMessages.en.mobile, ...mobileTaskDeliveryMessages.en.mobile } }
}
const { t } = useI18n({ messages: taskStatusMessages })
const approvalReason = ref("")
const showGitReview = ref(false)
const canRetry = computed(() => ["failed", "cancelled"].includes(props.state?.latestInstruction?.status || ""))
const isRunning = computed(
	() => props.state?.currentStage === "executing" || props.state?.latestRun?.status === "running"
)
const knownStages = [
	"idle",
	"interactive",
	"task_ready",
	"instruction_queued",
	"awaiting_approval",
	"executing",
	"completed",
	"preview_ready",
	"failed",
	"cancelled",
	"approval_rejected"
]
const stageLabel = computed(() => {
	const stage = props.state?.currentStage || "idle"
	return t(`mobile.stage_${knownStages.includes(stage) ? stage : "unknown"}`)
})
const deliveryStageLabel = computed(() => t(`mobile.deliveryStage_${props.state?.delivery?.stage || "queued"}`))
const deliveryStatusType = computed(() => {
	const status = props.state?.delivery?.status
	if (status === "completed") return "success"
	if (status === "failed" || status === "conflict" || status === "partial") return "error"
	return "info"
})
const deliverySession = computed(() => props.state?.session || props.session)
const deliveryAvailable = computed(() =>
	Boolean(deliverySession.value?.worktreeBranch || deliverySession.value?.isolationMode === "multi_worktree")
)
const tokenLabel = (count: number) => new Intl.NumberFormat().format(count || 0)
const latestRunTokenLabel = computed(() => {
	if (props.state?.latestRun?.tokenUsageStatus === "unavailable") return t("mobile.tokenNotRecorded")
	if (props.state?.latestRun?.tokenUsageStatus === "pending") return t("mobile.tokenPending")
	return tokenLabel(props.state?.latestRun?.totalTokens || 0)
})
const runtimeProgress = computed(() => props.state?.codexRuntime?.progress)
const runtimePercentage = computed(() => {
	const progress = runtimeProgress.value
	if (!progress?.totalSteps) return 0
	return Math.round((progress.completedSteps / progress.totalSteps) * 100)
})

function previewCanOpen(status: string) {
	return status === "ready"
}

watch(
	() => props.state?.pendingApproval?.id,
	() => {
		approvalReason.value = ""
	}
)
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="min(78dvh, 720px)" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('mobile.taskStatus')" closable>
			<n-spin :show="loading">
				<div class="space-y-4">
					<section
						v-if="deliverySession && (deliveryAvailable || state?.delivery)"
						class="rounded-2xl border border-slate-200 bg-white p-4"
					>
						<div class="flex items-center justify-between gap-3">
							<strong>{{ t("mobile.deliveryTitle") }}</strong>
							<n-tag v-if="state?.delivery" size="small" :type="deliveryStatusType" :bordered="false">
								{{ deliveryStageLabel }}
							</n-tag>
						</div>
						<template v-if="state?.delivery">
							<n-progress
								class="mt-3"
								type="line"
								:percentage="state.delivery.progress"
								:status="
									state.delivery.status === 'failed' ||
									state.delivery.status === 'conflict' ||
									state.delivery.status === 'partial'
										? 'error'
										: state.delivery.status === 'completed'
											? 'success'
											: 'default'
								"
							/>
							<div class="mt-2 flex justify-between text-xs text-slate-500">
								<span>
									{{
										t(`mobile.deliveryStatus_${state.delivery.status}`, {
											position: state.delivery.queuePosition
										})
									}}
								</span>
								<span>{{ t("mobile.deliveryAttempt", { count: state.delivery.attempt }) }}</span>
							</div>
							<n-alert v-if="state.delivery.errorMessage" class="mt-3" type="error" :show-icon="false">
								{{ state.delivery.errorMessage }}
							</n-alert>
							<CodeDeliveryFacts :facts="state.delivery.facts" :job-status="state.delivery.status" />
						</template>
						<n-button
							block
							size="large"
							type="primary"
							secondary
							class="mt-3 !h-11 !rounded-xl"
							@click="showGitReview = true"
						>
							<template #icon><Icon name="mdi:source-branch-check" :size="18" /></template>
							{{ t("mobile.gitReviewOpen") }}
						</n-button>
						<p class="mt-2 text-[11px] leading-5 text-slate-500">
							{{ t("mobile.gitReviewHint") }}
						</p>
					</section>
					<n-alert v-if="state?.pendingApproval" type="warning" :title="state.pendingApproval.title">
						<div class="whitespace-pre-wrap text-sm">{{ state.pendingApproval.content }}</div>
						<n-input
							v-model:value="approvalReason"
							class="mt-3"
							:placeholder="t('mobile.approvalReason')"
						/>
						<div class="mt-3 flex gap-2">
							<n-button type="warning" :loading="loading" @click="emit('approve', approvalReason)">
								{{ t("mobile.approve") }}
							</n-button>
							<n-button :disabled="loading" @click="emit('reject', approvalReason)">
								{{ t("mobile.reject") }}
							</n-button>
						</div>
					</n-alert>
					<n-alert v-if="state?.errorSummary" type="error">{{ state.errorSummary }}</n-alert>
					<section class="rounded-2xl border border-slate-200 bg-white p-4">
						<div class="flex items-center justify-between">
							<strong>{{ t("mobile.latestResult") }}</strong>
							<n-tag size="small" :bordered="false">{{ stageLabel }}</n-tag>
						</div>
						<div
							v-if="runtimeProgress?.totalSteps || runtimeProgress?.changedFiles"
							class="mt-3 rounded-xl bg-slate-50 p-3"
						>
							<div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
								<span v-if="runtimeProgress.totalSteps" class="font-semibold text-blue-700">
									{{
										t("mobile.runtimeStep", {
											current: runtimeProgress.currentStep,
											total: runtimeProgress.totalSteps
										})
									}}
								</span>
								<span v-if="runtimeProgress.changedFiles" class="font-medium text-emerald-700">
									{{ t("mobile.runtimeChangedFiles", { count: runtimeProgress.changedFiles }) }}
								</span>
								<span v-if="runtimeProgress.additions || runtimeProgress.deletions" class="text-slate-500">
									<span class="text-emerald-600">+{{ runtimeProgress.additions }}</span>
									<span class="ml-1 text-rose-600">-{{ runtimeProgress.deletions }}</span>
								</span>
							</div>
							<p v-if="runtimeProgress.stepTitle" class="mt-1 truncate text-sm text-slate-700">
								{{ runtimeProgress.stepTitle }}
							</p>
							<n-progress
								v-if="runtimeProgress.totalSteps"
								class="mt-2"
								type="line"
								:percentage="runtimePercentage"
								:show-indicator="false"
								:height="5"
							/>
						</div>
						<p class="mt-3 whitespace-pre-wrap text-sm leading-6 text-slate-600">
							{{ state?.recentOutput || t("mobile.noResult") }}
						</p>
						<div v-if="state?.latestRun" class="mt-3 flex flex-wrap gap-3 text-xs text-slate-400">
							<span>{{ state.latestRun.executorId }}</span>
							<span>{{ Math.round(state.latestRun.durationMs / 1000) }}s</span>
							<span>{{ t("mobile.exitCode", { code: state.latestRun.exitCode }) }}</span>
						</div>
					</section>
					<section
						v-if="state?.tokenUsage.project.runs"
						class="rounded-2xl border border-slate-200 bg-white p-4"
					>
						<strong>{{ t("mobile.tokenUsage") }}</strong>
						<div class="mt-3 grid grid-cols-3 gap-2 text-center">
							<div class="rounded-xl bg-slate-50 p-2">
								<div class="font-semibold">{{ latestRunTokenLabel }}</div>
								<div class="text-[11px] text-slate-500">{{ t("mobile.latestRunTokens") }}</div>
							</div>
							<div class="rounded-xl bg-blue-50 p-2">
								<div class="font-semibold text-blue-700">
									{{ tokenLabel(state.tokenUsage.session.totalTokens) }}
								</div>
								<div class="text-[11px] text-slate-500">{{ t("mobile.sessionTokens") }}</div>
							</div>
							<div class="rounded-xl bg-emerald-50 p-2">
								<div class="font-semibold text-emerald-700">
									{{ tokenLabel(state.tokenUsage.project.totalTokens) }}
								</div>
								<div class="text-[11px] text-slate-500">{{ t("mobile.projectTokens") }}</div>
							</div>
						</div>
						<div v-if="state.tokenUsage.project.recoveredRuns" class="mt-2 text-xs text-emerald-600">
							{{ t("mobile.tokenUsageRecovered", { count: state.tokenUsage.project.recoveredRuns }) }}
						</div>
						<div v-if="state.tokenUsage.project.unavailableRuns" class="mt-2 text-xs text-amber-600">
							{{ t("mobile.tokenUsageIncomplete", { count: state.tokenUsage.project.unavailableRuns }) }}
						</div>
						<div v-if="state.tokenUsage.project.pendingRuns" class="mt-2 text-xs text-slate-500">
							{{ t("mobile.tokenUsagePending", { count: state.tokenUsage.project.pendingRuns }) }}
						</div>
					</section>
					<n-alert
						v-if="state && !state.tokenUsage.budget.unlimited"
						:type="state.tokenUsage.budget.exceeded ? 'error' : 'info'"
						:show-icon="false"
					>
						{{
							state.tokenUsage.budget.exceeded
								? t("mobile.tokenBudgetExceeded")
								: t("mobile.tokenBudget", { percent: Math.round(state.tokenUsage.budget.usagePercent) })
						}}
					</n-alert>
					<section v-if="state?.changedFiles.length" class="rounded-2xl border border-slate-200 bg-white p-4">
						<strong>{{ t("mobile.changedFiles") }}</strong>
						<div class="mt-3 space-y-2">
							<div
								v-for="file in state.changedFiles"
								:key="file"
								class="flex items-center gap-2 text-sm text-slate-600"
							>
								<Icon name="mdi:file-code-outline" :size="17" />
								<span class="truncate">{{ file }}</span>
							</div>
						</div>
					</section>
					<section v-if="state?.previews.length" class="rounded-2xl border border-slate-200 bg-white p-4">
						<strong>{{ t("mobile.previews") }}</strong>
						<a
							v-for="preview in state.previews"
							:key="preview.id"
							:href="previewCanOpen(preview.status) ? preview.url : undefined"
							:target="previewCanOpen(preview.status) ? '_blank' : undefined"
							rel="noopener noreferrer"
							class="mt-3 flex w-full items-center gap-3 rounded-xl bg-slate-50 p-3 text-left"
							:class="previewCanOpen(preview.status) ? '' : 'cursor-not-allowed opacity-60'"
						>
							<Icon
								:name="previewCanOpen(preview.status) ? 'mdi:open-in-new' : 'mdi:link-off'"
								:size="18"
							/>
							<span class="min-w-0 flex-1 truncate text-sm">{{ preview.title }}</span>
							<n-tag size="small" :bordered="false">{{ preview.status }}</n-tag>
						</a>
						<div
							v-if="state.previews.some(preview => !previewCanOpen(preview.status))"
							class="mt-2 text-xs text-slate-500"
						>
							{{ t("mobile.previewUnavailableHint") }}
						</div>
					</section>
					<section
						v-if="state?.timelineEvents.length"
						class="rounded-2xl border border-slate-200 bg-white p-4"
					>
						<strong>{{ t("mobile.timeline") }}</strong>
						<div class="mt-4 space-y-4">
							<div
								v-for="event in state.timelineEvents"
								:key="event.id"
								class="border-l-2 border-blue-200 pl-3"
							>
								<div class="text-sm font-medium text-slate-800">{{ event.title }}</div>
								<div
									v-if="event.content"
									class="mt-1 whitespace-pre-wrap text-xs leading-5 text-slate-500"
								>
									{{ event.content }}
								</div>
							</div>
						</div>
					</section>
				</div>
			</n-spin>
			<template #footer>
				<div class="flex w-full gap-2">
					<n-button v-if="isRunning" type="error" secondary :loading="loading" @click="emit('stop')">
						{{ t("mobile.stop") }}
					</n-button>
					<n-button v-if="canRetry" secondary :loading="loading" @click="emit('retry')">
						{{ t("mobile.retryExecution") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
	<MobileGitReviewDrawer
		v-if="deliverySession"
		v-model:show="showGitReview"
		:session="deliverySession"
		:delivery="state?.delivery"
		@updated="emit('deliveryUpdated')"
	/>
</template>
