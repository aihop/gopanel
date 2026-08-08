<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { getCodeProjectBranches } from "@/api/modules/code"
import type { CodeProjectBranch, CodeProjectBranches } from "@/api/interface/codeBranches"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import TaskApprovalAction from "./TaskApprovalAction.vue"

const props = defineProps<{
	projectId: number
	tasks: CodeTaskListItem[]
	taskTotal: number
	currentTaskId: number | null
	taskActionOptions: Array<Record<string, unknown>>
}>()
const emit = defineEmits<{
	selectTask: [task: CodeTaskListItem]
	taskAction: [key: string, task: CodeTaskListItem]
	refreshTasks: []
}>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const message = useMessage()
const branchState = ref<CodeProjectBranches>({ repositories: [], totalBranches: 0 })
const branchesLoading = ref(false)
const branchesError = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | undefined

const repositoryCountLabel = computed(() => t("code.repositoryCount", { count: branchState.value.repositories.length }))

const fetchBranches = async (silent = false) => {
	if (!props.projectId || branchesLoading.value) return
	branchesLoading.value = true
	if (!silent) branchesError.value = false
	try {
		const response = await getCodeProjectBranches(props.projectId)
		if (response.code !== 0) throw new Error(response.message || t("code.branchLoadFailed"))
		branchState.value = response.data
		branchesError.value = false
	} catch (error) {
		branchesError.value = true
	} finally {
		branchesLoading.value = false
	}
}

const branchScopeLabel = (branch: CodeProjectBranch) =>
	t(branch.scope === "remote" ? "code.remoteBranch" : "code.localBranch")

const formatTaskDuration = (durationMs: number) => {
	const seconds = Math.max(1, Math.round(durationMs / 1000))
	if (seconds < 60) return t("code.taskDurationSeconds", { count: seconds })
	const minutes = Math.floor(seconds / 60)
	const remainingSeconds = seconds % 60
	if (minutes < 60)
		return remainingSeconds
			? t("code.taskDurationMinutesSeconds", { minutes, seconds: remainingSeconds })
			: t("code.taskDurationMinutes", { count: minutes })
	const hours = Math.floor(minutes / 60)
	const remainingMinutes = minutes % 60
	return remainingMinutes
		? t("code.taskDurationHoursMinutes", { hours, minutes: remainingMinutes })
		: t("code.taskDurationHours", { count: hours })
}

const taskGitMeta = (task: CodeTaskListItem) =>
	task.summary.deliveryStatus === "queued"
		? { icon: "mdi:clock-outline", color: "text-amber-500" }
		: task.summary.deliveryStatus === "running"
			? { icon: "mdi:cloud-sync-outline", color: "text-blue-500" }
			: task.summary.deliveryStatus === "failed"
				? { icon: "mdi:cloud-alert-outline", color: "text-red-500" }
				: task.summary.deliveryStatus === "conflict"
					? { icon: "mdi:source-branch-alert", color: "text-red-500" }
					:
	({
		working: { icon: "mdi:source-branch", color: "text-slate-400" },
		committed: { icon: "mdi:source-commit", color: "text-blue-500" },
		merged: { icon: "mdi:source-merge", color: "text-emerald-600" },
		pushed: { icon: "mdi:cloud-check-outline", color: "text-emerald-600" },
		push_failed: { icon: "mdi:cloud-alert-outline", color: "text-red-500" },
		conflict: { icon: "mdi:source-branch-alert", color: "text-red-500" }
	})[task.summary.gitStatus || ""]

const taskTooltip = (task: CodeTaskListItem) =>
	[
		task.summary.deliveryStatus
			? t(`code.deliveryStatus_${task.summary.deliveryStatus}`, {
				position: task.summary.deliveryQueuePosition,
				progress: task.summary.deliveryProgress
			})
			: "",
		task.summary.deliveryStage ? t(`code.deliveryStage_${task.summary.deliveryStage}`) : "",
		task.summary.deliveryError,
		task.summary.gitStatus ? t(`code.taskGitStatus_${task.summary.gitStatus}`) : "",
		task.summary.gitError,
		task.summary.executor,
		task.summary.model,
		task.summary.branch,
		new Date(task.createdAt).toLocaleString()
	]
		.filter(Boolean)
		.join(" · ")

onMounted(() => {
	void fetchBranches()
	refreshTimer = setInterval(() => void fetchBranches(true), 60000)
})
onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
watch(
	() => props.projectId,
	() => {
		branchState.value = { repositories: [], totalBranches: 0 }
		void fetchBranches()
	}
)
</script>

<template>
	<div class="flex min-h-0 flex-1 flex-col overflow-hidden">
		<section class="ai-workspace-task-history mt-3 flex min-h-0 flex-1 flex-col overflow-hidden">
			<div class="flex items-center justify-between px-4 pb-2 pt-1">
				<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
					{{ t("code.taskHistory") }}
				</div>
				<div class="text-xs text-slate-400">{{ t("code.taskCount", { count: taskTotal }) }}</div>
			</div>
			<n-scrollbar trigger="none" class="ai-workspace-task-scrollbar min-h-0 flex-1">
				<div class="px-2.5 pb-3 pr-3.5">
					<div
						v-if="tasks.length === 0"
						class="ai-workspace-task-empty flex min-h-[180px] items-center justify-center"
					>
						<n-empty :description="t('code.noProjectHistory')" />
					</div>
					<div v-else class="space-y-1">
						<div
							v-for="task in tasks"
							:key="task.id"
							class="ai-workspace-task-row group/task relative flex cursor-pointer items-start justify-between gap-3 rounded-xl px-3 py-2.5 transition-colors duration-200 hover:bg-slate-200/45"
							:class="currentTaskId === task.id ? 'ai-workspace-task-row--active bg-blue-50/80' : ''"
							@click="emit('selectTask', task)"
						>
							<div class="min-w-0 flex-1">
								<div class="flex min-w-0 items-center gap-1.5 text-sm font-semibold text-slate-800" :title="task.title">
									<Icon
										v-if="task.agentName === 'terminal'"
										name="mdi:console-line"
										:size="14"
										class="shrink-0 text-slate-500"
									/>
									<span class="truncate">{{ task.title }}</span>
								</div>
								<div
									class="mt-1.5 flex min-w-0 items-center gap-1.5 overflow-hidden whitespace-nowrap text-[10px] text-slate-400"
									:title="taskTooltip(task)"
								>
									<TaskApprovalAction
										class="shrink-0"
										:task="task"
										@approved="emit('refreshTasks')"
									/>
									<template v-if="taskGitMeta(task)">
										<span class="text-slate-300">·</span>
										<Icon
											:name="taskGitMeta(task)!.icon"
											:size="13"
											class="shrink-0"
											:class="taskGitMeta(task)!.color"
										/>
									</template>
									<template v-if="task.summary.hasDiff">
										<span class="text-slate-300">·</span>
										<span class="font-medium text-emerald-600">+{{ task.summary.additions }}</span>
										<span class="font-medium text-red-500">-{{ task.summary.deletions }}</span>
										<span>
											{{ t("code.taskChangedFiles", { count: task.summary.changedFiles }) }}
										</span>
									</template>
									<template v-if="task.summary.durationMs > 0">
										<span class="text-slate-300">·</span>
										<span class="whitespace-nowrap">
											{{ formatTaskDuration(task.summary.durationMs) }}
										</span>
									</template>
								</div>
							</div>
							<div
								class="opacity-100 transition-opacity md:opacity-0 md:group-hover/task:opacity-100"
								@click.stop
							>
								<n-dropdown
									trigger="click"
									:options="taskActionOptions"
									@select="key => emit('taskAction', String(key), task)"
								>
									<n-button
										quaternary
										circle
										size="small"
										class="ai-workspace-task-btn !bg-transparent"
									>
										<template #icon><Icon name="mdi:dots-horizontal" /></template>
									</n-button>
								</n-dropdown>
							</div>
						</div>
					</div>
				</div>
			</n-scrollbar>
		</section>

		<section
			class="ai-workspace-branches flex max-h-[42%] min-h-[168px] shrink-0 flex-col border-t border-slate-200/80 bg-white/65"
		>
			<div class="flex shrink-0 items-center justify-between gap-2 px-4 py-2.5">
				<div class="min-w-0">
					<div class="mt-0.5 text-[11px] text-slate-400">
						{{
							t("code.branchSummary", {
								branches: branchState.totalBranches,
								repositories: repositoryCountLabel
							})
						}}
					</div>
				</div>
				<n-button
					quaternary
					circle
					size="tiny"
					:loading="branchesLoading"
					:aria-label="t('code.refreshBranches')"
					@click="fetchBranches()"
				>
					<template #icon><Icon name="mdi:refresh" :size="15" /></template>
				</n-button>
			</div>
			<div
				v-if="branchesLoading && branchState.repositories.length === 0"
				class="flex min-h-0 flex-1 items-center justify-center"
			>
				<n-spin size="small" />
			</div>
			<div
				v-else-if="branchesError && branchState.repositories.length === 0"
				class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-4 text-center text-xs text-red-500"
			>
				<span>{{ t("code.branchLoadFailed") }}</span>
				<n-button text type="primary" size="tiny" @click="fetchBranches()">{{ t("code.retry") }}</n-button>
			</div>
			<div
				v-else-if="branchState.repositories.length === 0"
				class="flex min-h-0 flex-1 items-center justify-center px-4 text-center text-xs text-slate-400"
			>
				{{ t("code.noGitRepositories") }}
			</div>
			<n-scrollbar v-else trigger="none" class="ai-workspace-branch-scrollbar min-h-0 flex-1">
				<div class="space-y-3 px-4 pb-3">
					<div v-for="repository in branchState.repositories" :key="repository.path">
						<div class="flex items-center justify-between gap-2">
							<div
								class="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-slate-700"
								:title="repository.path"
							>
								<Icon name="mdi:source-repository" :size="14" class="shrink-0 text-slate-400" />
								<span class="truncate">{{ repository.name }}</span>
							</div>
							<span
								v-if="repository.dirty"
								class="shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-700"
							>
								{{ t("code.changedFiles", { count: repository.changedFiles }) }}
							</span>
						</div>
						<div class="mt-1.5 space-y-0.5 pl-1">
							<div
								v-for="branch in repository.branches"
								:key="branch.ref"
								class="py-1.5 pl-4"
								:class="branch.current ? 'text-blue-600' : ''"
							>
								<div class="flex items-center justify-between gap-2">
									<div
										class="flex min-w-0 items-center gap-1.5 text-[11px] font-medium"
										:class="branch.current ? 'text-blue-700' : 'text-slate-700'"
										:title="branch.name"
									>
										<Icon
											:name="branch.current ? 'mdi:source-branch-check' : 'mdi:source-branch'"
											:size="13"
											:class="branch.current ? 'text-blue-600' : 'text-slate-400'"
										/>
										<span class="truncate">{{ branch.name }}</span>
									</div>
									<span class="shrink-0 text-[9px] uppercase tracking-wide text-slate-400">
										{{ branchScopeLabel(branch) }}
									</span>
								</div>
								<div class="mt-1 flex items-center gap-1.5 text-[10px] text-slate-400">
									<span class="font-mono">{{ branch.commit }}</span>
									<span class="font-medium text-emerald-600">+{{ branch.additions }}</span>
									<span class="font-medium text-red-500">-{{ branch.deletions }}</span>
									<span class="truncate" :title="branch.subject">{{ branch.subject }}</span>
									<span
										v-if="branch.scope === 'local' && branch.merged"
										class="ml-auto shrink-0 text-emerald-600"
									>
										{{ t("code.branchMerged") }}
									</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</n-scrollbar>
		</section>
	</div>
</template>

<style scoped>
.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical),
.ai-workspace-branch-scrollbar :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical) {
	right: 3px !important;
	width: 3px !important;
}

.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail__scrollbar),
.ai-workspace-branch-scrollbar :deep(.n-scrollbar-rail__scrollbar) {
	width: 3px !important;
	background-color: rgba(100, 116, 139, 0.28) !important;
}

.ai-workspace-task-row--active::before {
	position: absolute;
	top: 8px;
	bottom: 8px;
	left: 0;
	width: 4px;
	border-radius: 0 9999px 9999px 0;
	background-color: rgb(29, 78, 216);
	box-shadow: 0 2px 8px rgba(29, 78, 216, 0.32);
	content: "";
}

:global(.theme-dark) .ai-workspace-branches {
	border-top-color: color-mix(in srgb, var(--border-color) 80%, transparent);
	background-color: color-mix(in srgb, var(--bg-default-color) 76%, transparent);
}

:global(.theme-dark) .ai-workspace-task-row:hover {
	background-color: color-mix(in srgb, var(--fg-secondary-color) 10%, transparent) !important;
}

:global(.theme-dark) .ai-workspace-task-row--active {
	background-color: color-mix(in srgb, var(--primary-color) 14%, transparent) !important;
}

:global(.theme-dark) .ai-workspace-task-row--active::before {
	background-color: rgb(96, 165, 250);
	box-shadow: 0 2px 8px rgba(96, 165, 250, 0.28);
}

:global(.theme-dark) .ai-workspace-task-btn {
	background-color: color-mix(in srgb, var(--fg-secondary-color) 15%, transparent) !important;
}
</style>
