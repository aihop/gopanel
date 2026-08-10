<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { AIProject } from "@/api/interface/code"
import type { CodeProjectOverview } from "@/api/interface/codeOverview"
import type { CodeProjectBranches } from "@/api/interface/codeBranches"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { getCodeProjectBranches } from "@/api/modules/code"
import { getCodeProjectOverview } from "@/api/modules/codeOverview"
import Icon from "@/components/common/Icon.vue"
import ProjectRepositorySync from "./ProjectRepositorySync.vue"
import { projectOverviewMessages } from "../projectOverviewMessages"

const props = defineProps<{
	project: AIProject | null
	projectId: number
	tasks: CodeTaskListItem[]
	terminalAvailable: boolean
}>()
const emit = defineEmits<{ createTask: []; openTerminal: []; selectTask: [task: CodeTaskListItem] }>()
const { t } = useI18n({ messages: projectOverviewMessages })
const message = useMessage()
const overview = ref<CodeProjectOverview | null>(null)
const branches = ref<CodeProjectBranches>({ repositories: [], totalBranches: 0 })
const loading = ref(false)
const loadError = ref(false)
const branchesError = ref(false)
let overviewPending = false
let overviewRequest = 0
let branchesRequest = 0

const summary = computed(() => overview.value?.executionSummary || props.project?.executionSummary)
const activeTask = computed(() => props.tasks.find(task => task.id === summary.value?.currentTaskId) || null)
const changedFiles = computed(() =>
	branches.value.repositories.reduce((total, repository) => total + repository.changedFiles, 0)
)
const currentBranches = computed(() =>
	branches.value.repositories.map(repository => repository.currentBranch).filter(Boolean)
)
const status = computed(() =>
	summary.value?.pendingApprovalCount ? "pending_approval" : summary.value?.status || "idle"
)
const statusMeta = computed(() => {
	if (status.value === "pending_approval") return { type: "warning" as const, icon: "mdi:shield-alert-outline" }
	if (status.value === "delivering") return { type: "info" as const, icon: "mdi:source-merge" }
	if (status.value === "running") return { type: "info" as const, icon: "mdi:progress-clock" }
	if (status.value === "queued") return { type: "default" as const, icon: "mdi:clock-outline" }
	return { type: "success" as const, icon: "mdi:check-circle-outline" }
})
const budgetPercentage = computed(() => Math.min(100, Math.round(overview.value?.budget.usagePercent || 0)))

const formatTokens = (count: number) => new Intl.NumberFormat().format(count || 0)
const formatDuration = (milliseconds: number) =>
	milliseconds < 1000
		? t("code.durationMilliseconds", { count: milliseconds })
		: t("code.durationSeconds", { count: (milliseconds / 1000).toFixed(milliseconds < 10000 ? 1 : 0) })
const formatDate = (value?: string) => (value ? new Date(value).toLocaleString() : t("code.noRecentActivity"))
const runStatusLabel = (value: string) =>
	["completed", "running", "failed", "cancelled"].includes(value)
		? t(`code.runStatus_${value}`)
		: t("code.runStatus_unknown")
const stageLabel = (value?: string) => {
	const stages = [
		"idle",
		"interactive",
		"task_ready",
		"instruction_queued",
		"awaiting_approval",
		"executing",
		"completed",
		"cancelled",
		"failed"
	]
	return value && stages.includes(value) ? t(`code.stage_${value}`) : t("code.overviewUnknownStage")
}

const loadOverview = async (notify = false) => {
	if (!props.projectId || overviewPending) return
	const projectId = props.projectId
	const requestId = ++overviewRequest
	overviewPending = true
	if (!overview.value) loading.value = true
	try {
		const response = await getCodeProjectOverview(projectId)
		if (response.code !== 0) throw new Error(response.message)
		if (requestId !== overviewRequest || projectId !== props.projectId) return
		overview.value = response.data
		loadError.value = false
	} catch (error) {
		if (requestId !== overviewRequest || projectId !== props.projectId) return
		if (notify || !overview.value) loadError.value = true
	} finally {
		if (requestId === overviewRequest) {
			loading.value = false
			overviewPending = false
		}
	}
}

const loadBranches = async (notify = false) => {
	if (!props.projectId) return
	const projectId = props.projectId
	const requestId = ++branchesRequest
	try {
		const response = await getCodeProjectBranches(projectId)
		if (response.code !== 0) throw new Error(response.message)
		if (requestId !== branchesRequest || projectId !== props.projectId) return
		branches.value = response.data
		branchesError.value = false
	} catch (error) {
		if (requestId !== branchesRequest || projectId !== props.projectId) return
		branchesError.value = true
		branches.value = { repositories: [], totalBranches: 0 }
	}
}

const refresh = () => {
	void loadOverview(true)
	void loadBranches(true)
}

const continueTask = () => {
	if (activeTask.value) emit("selectTask", activeTask.value)
	else emit("createTask")
}

onMounted(refresh)
watch(
	() => props.projectId,
	() => {
		overviewRequest++
		branchesRequest++
		overviewPending = false
		overview.value = null
		branches.value = { repositories: [], totalBranches: 0 }
		loading.value = false
		loadError.value = false
		branchesError.value = false
		refresh()
	}
)
useIntervalFn(() => void loadOverview(), 5000)
</script>

<template>
	<div
		class="min-h-0 flex-1 overflow-auto rounded-[20px] border border-slate-200/80 p-4 shadow-[0_24px_50px_rgba(15,23,42,0.10)] md:p-6 dark:border-[var(--border-color)]"
	>
		<n-spin :show="loading">
			<n-alert v-if="loadError && !overview" type="error" :title="t('code.overviewLoadFailed')">
				<n-button size="small" @click="refresh">{{ t("code.retry") }}</n-button>
			</n-alert>
			<div v-else class="mx-auto max-w-6xl space-y-5">
				<section
					class="rounded-[22px] border border-slate-200 bg-white p-5 md:p-6 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
				>
					<div class="flex flex-col justify-between gap-5 md:flex-row md:items-center">
						<div class="min-w-0">
							<div class="mb-3 flex flex-wrap items-center gap-2">
								<n-tag :type="statusMeta.type" :bordered="false" round>
									<template #icon><Icon :name="statusMeta.icon" :size="15" /></template>
									{{ t(`code.overviewStatus_${status}`) }}
								</n-tag>
								<span class="text-xs text-[var(--n-text-color-3)]">
									{{ formatDate(summary?.updatedAt) }}
								</span>
							</div>
							<h2 class="truncate text-xl font-semibold text-[var(--n-text-color)] md:text-2xl">
								{{ summary?.currentTaskTitle || t("code.overviewReadyTitle") }}
							</h2>
							<p class="mt-2 max-w-2xl text-sm leading-6 text-[var(--n-text-color-2)]">
								{{
									summary?.currentTaskTitle
										? t("code.overviewActiveHint", {
												stage: stageLabel(summary.currentStage)
											})
										: t("code.overviewReadyHint")
								}}
							</p>
						</div>
						<div class="flex shrink-0 flex-wrap gap-2 rounded-2xl bg-slate-50 p-1.5 dark:bg-white/5">
							<n-button v-if="activeTask" type="primary" size="large" @click="continueTask">
								<template #icon><Icon name="mdi:play-outline" /></template>
								{{ t("code.continueTask") }}
							</n-button>
							<n-button
								:type="activeTask ? 'default' : 'primary'"
								size="large"
								class="!rounded-xl"
								@click="emit('createTask')"
							>
								<template #icon><Icon name="mdi:robot-outline" /></template>
								{{ t("code.newAiTask") }}
							</n-button>
							<n-button
								v-if="terminalAvailable"
								size="large"
								secondary
								class="!rounded-xl"
								@click="emit('openTerminal')"
							>
								<template #icon><Icon name="mdi:console-line" /></template>
								{{ t("code.projectTerminal") }}
							</n-button>
						</div>
					</div>
				</section>
				<ProjectRepositorySync :project-id="projectId" />

				<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
					<section
						class="rounded-2xl border border-slate-200 bg-white p-4 sm:col-span-2 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
					>
						<div class="flex items-start justify-between gap-3">
							<div>
								<div class="text-xs font-medium text-[var(--n-text-color-3)]">
									{{ t("code.monthlyTokenUsage") }}
								</div>
								<div class="mt-1 text-2xl font-semibold text-[var(--n-text-color)]">
									{{ formatTokens(overview?.budget.usedTokens || 0) }}
								</div>
							</div>
							<Icon name="mdi:gauge" :size="24" class="text-blue-500" />
						</div>
						<div v-if="overview?.budget.unlimited" class="mt-3 text-xs text-[var(--n-text-color-3)]">
							{{ t("code.unlimitedTokenBudget") }}
						</div>
						<div v-else class="mt-3">
							<div class="mb-1.5 flex justify-between text-xs text-[var(--n-text-color-3)]">
								<span>
									{{
										t("code.tokenBudgetUsed", {
											percent: Math.round(overview?.budget.usagePercent || 0)
										})
									}}
								</span>
								<span>
									{{ formatTokens(overview?.budget.remainingTokens || 0) }} {{ t("code.remaining") }}
								</span>
							</div>
							<n-progress
								type="line"
								:percentage="budgetPercentage"
								:status="overview?.budget.exceeded ? 'error' : 'default'"
								:show-indicator="false"
							/>
						</div>
					</section>
					<section
						class="rounded-2xl border border-slate-200 bg-white p-4 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
					>
						<div class="flex justify-between">
							<div>
								<div class="text-xs text-[var(--n-text-color-3)]">{{ t("code.activeTasks") }}</div>
								<div class="mt-2 text-2xl font-semibold">{{ summary?.activeTaskCount || 0 }}</div>
							</div>
							<Icon name="mdi:progress-wrench" :size="23" class="text-sky-500" />
						</div>
					</section>
					<section
						class="rounded-2xl border border-slate-200 bg-white p-4 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
					>
						<div class="flex justify-between">
							<div>
								<div class="text-xs text-[var(--n-text-color-3)]">
									{{ t("code.awaitingApprovals") }}
								</div>
								<div
									class="mt-2 text-2xl font-semibold"
									:class="summary?.pendingApprovalCount ? 'text-amber-600' : ''"
								>
									{{ summary?.pendingApprovalCount || 0 }}
								</div>
							</div>
							<Icon name="mdi:shield-check-outline" :size="23" class="text-amber-500" />
						</div>
					</section>
				</div>

				<div class="grid gap-4 lg:grid-cols-2">
					<section
						class="rounded-2xl border border-slate-200 bg-white p-5 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
					>
						<div class="flex items-center justify-between gap-3">
							<h3 class="font-semibold">{{ t("code.projectUsage") }}</h3>
							<span class="text-xs text-[var(--n-text-color-3)]">
								{{ t("code.executionCount", { count: overview?.tokenUsage.runs || 0 }) }}
							</span>
						</div>
						<div class="mt-4 grid grid-cols-2 gap-3">
							<div class="rounded-xl p-3">
								<div class="text-xs text-slate-500">{{ t("code.historicalTokens") }}</div>
								<div class="mt-1 text-lg font-semibold text-slate-800">
									{{ formatTokens(overview?.tokenUsage.totalTokens || 0) }}
								</div>
							</div>
							<div class="rounded-xl p-3">
								<div class="text-xs text-slate-500">{{ t("code.projectTasks") }}</div>
								<div class="mt-1 text-lg font-semibold text-slate-800">
									{{ overview?.taskCount || tasks.length }}
								</div>
							</div>
						</div>
						<div v-if="overview?.latestRun" class="mt-4 rounded-xl border border-slate-100 p-3 text-sm">
							<div class="flex items-center justify-between gap-3">
								<span class="font-medium">
									{{ overview.latestRun.model || overview.latestRun.executorId }}
								</span>
								<n-tag size="small" :bordered="false">
									{{ runStatusLabel(overview.latestRun.status) }}
								</n-tag>
							</div>
							<div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--n-text-color-3)]">
								<span>
									{{ t("code.tokenCount", { count: formatTokens(overview.latestRun.totalTokens) }) }}
								</span>
								<span>{{ formatDuration(overview.latestRun.durationMs) }}</span>
								<span>{{ formatDate(overview.latestRun.createdAt) }}</span>
							</div>
						</div>
						<n-empty v-else size="small" :description="t('code.noExecutionHistory')" class="py-5" />
					</section>

					<section
						class="rounded-2xl border border-slate-200 bg-white p-5 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
					>
						<div class="flex items-center justify-between gap-3">
							<h3 class="font-semibold">{{ t("code.projectHealth") }}</h3>
							<n-button
								quaternary
								circle
								size="small"
								:aria-label="t('code.refreshOverview')"
								@click="refresh"
							>
								<template #icon><Icon name="mdi:refresh" /></template>
							</n-button>
						</div>
						<n-alert v-if="branchesError" type="warning" :show-icon="false" class="mt-3">
							{{ t("code.branchLoadFailed") }}
						</n-alert>
						<div class="mt-4 space-y-3 text-sm">
							<div class="flex items-center justify-between gap-3 rounded-xl p-3">
								<span class="flex items-center gap-2 text-slate-600">
									<Icon name="mdi:source-repository" />
									{{ t("code.repositories") }}
								</span>
								<strong class="text-slate-800">{{ branches.repositories.length }}</strong>
							</div>
							<div class="flex items-center justify-between gap-3 rounded-xl p-3">
								<span class="flex items-center gap-2 text-slate-600">
									<Icon name="mdi:file-document-edit-outline" />
									{{ t("code.uncommittedChanges") }}
								</span>
								<strong :class="changedFiles ? 'text-amber-600' : 'text-emerald-600'">
									{{ changedFiles }}
								</strong>
							</div>
							<div class="flex items-center justify-between gap-3 rounded-xl p-3">
								<span class="flex items-center gap-2 text-slate-600">
									<Icon name="mdi:test-tube" />
									{{ t("code.qualityGate") }}
								</span>
								<n-tag
									size="small"
									:type="project?.requireQualityGate ? 'success' : 'default'"
									:bordered="false"
								>
									{{ t(project?.requireQualityGate ? "code.enabled" : "code.disabled") }}
								</n-tag>
							</div>
						</div>
						<div
							v-if="currentBranches.length"
							class="mt-3 truncate text-xs text-[var(--n-text-color-3)]"
							:title="currentBranches.join(', ')"
						>
							{{ t("code.currentBranches", { branches: currentBranches.join(", ") }) }}
						</div>
					</section>
				</div>
			</div>
		</n-spin>
	</div>
</template>
