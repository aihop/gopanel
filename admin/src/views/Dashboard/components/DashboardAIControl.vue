<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useDocumentVisibility, useIntervalFn } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useRouter } from "vue-router"
import type { AIProject, CodeAttentionItem } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { getAIProjects, getAITasks, getCodeAttention } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import CodeProjectIdentity from "@/views/Code/components/CodeProjectIdentity.vue"
import CodeTaskMetaLine from "@/views/Code/components/CodeTaskMetaLine.vue"
import TaskStatusBadge from "@/views/Code/components/TaskStatusBadge.vue"
import { codeTaskBucket, groupCodeDashboardTasks } from "@/views/Code/codeDashboardBuckets"
import { dashboardMessages } from "../dashboardMessages"

const { t } = useI18n({ messages: dashboardMessages })
const router = useRouter()
const visibility = useDocumentVisibility()
const tasks = ref<CodeTaskListItem[]>([])
const projects = ref<AIProject[]>([])
const attentionItems = ref<CodeAttentionItem[]>([])
const tasksLoading = ref(true)
const attentionLoading = ref(true)
const tasksError = ref(false)
const attentionError = ref(false)
const refreshing = ref(false)
const now = ref(new Date())

const projectNames = computed(() => new Map(projects.value.map(project => [project.id, project.name])))
const groupedTasks = computed(() => groupCodeDashboardTasks(tasks.value, now.value))
const taskRank = (task: CodeTaskListItem) => {
	const bucket = codeTaskBucket(task, now.value)
	return bucket === "attention" ? 0 : bucket === "active" ? 1 : bucket === "doneToday" ? 2 : 3
}
const visibleTasks = computed(() =>
	[...tasks.value]
		.filter(task => codeTaskBucket(task, now.value))
		.sort((left, right) => taskRank(left) - taskRank(right) || right.id - left.id)
		.slice(0, 8)
)
const stats = computed(() => [
	{
		key: "running",
		label: t("dashboardControl.running"),
		count: groupedTasks.value.active.length,
		icon: "mdi:play-circle-outline",
		tone: "text-blue-600"
	},
	{
		key: "attention",
		label: t("dashboardControl.needsAttention"),
		count: groupedTasks.value.attention.length,
		icon: "mdi:shield-alert-outline",
		tone: "text-amber-600"
	},
	{
		key: "delivering",
		label: t("dashboardControl.delivering"),
		count: groupedTasks.value.deliveringCount,
		icon: "mdi:source-merge",
		tone: "text-violet-600"
	},
	{
		key: "done",
		label: t("dashboardControl.doneToday"),
		count: groupedTasks.value.doneToday.length,
		icon: "mdi:check-circle-outline",
		tone: "text-emerald-600"
	}
])

function projectName(task: CodeTaskListItem) {
	return projectNames.value.get(task.projectId) || t("dashboardControl.projectFallback", { id: task.projectId })
}

function formatTime(value: string) {
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) return "--"
	return date.toLocaleString(undefined, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })
}

function openTask(task: CodeTaskListItem) {
	void router.push({ path: `/code/project/${task.projectId}`, query: { taskId: String(task.id) } })
}

function openAttention(item: CodeAttentionItem) {
	const task = tasks.value.find(candidate => candidate.id === item.taskId || candidate.sessionId === item.sessionId)
	if (task) {
		openTask(task)
		return
	}
	if (item.projectId) {
		void router.push({ path: `/code/project/${item.projectId}` })
		return
	}
	void router.push({ name: "Code-Index" })
}

async function loadTasks() {
	try {
		const [taskResponse, projectResponse] = await Promise.all([
			getAITasks({ page: 1, limit: 50, projectId: 0, includeGit: false }),
			getAIProjects({ page: 1, limit: 100 })
		])
		tasks.value = taskResponse.data.items || []
		projects.value = projectResponse.data.items || []
		tasksError.value = false
	} catch {
		tasksError.value = true
	} finally {
		tasksLoading.value = false
	}
}

async function loadAttention() {
	try {
		const response = await getCodeAttention(8)
		attentionItems.value = response.data.items || []
		attentionError.value = false
	} catch {
		attentionError.value = true
	} finally {
		attentionLoading.value = false
	}
}

async function refresh() {
	refreshing.value = true
	now.value = new Date()
	await Promise.all([loadTasks(), loadAttention()])
	refreshing.value = false
}

useIntervalFn(() => {
	if (visibility.value === "visible") void refresh()
}, 10000)

onMounted(refresh)
</script>

<template>
	<section class="grid grid-cols-1 items-start gap-8 2xl:contents">
		<div class="space-y-5 self-start 2xl:col-start-1">
			<div class="bg-base-100 rounded-2xl border border-slate-200 p-6 shadow-sm">
				<div class="mb-5 flex flex-wrap items-start justify-between gap-4">
					<div>
						<div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">
							{{ t("dashboardControl.aiDevelopment") }}
						</div>
						<div class="fg-base-100 mt-2 text-xl font-semibold">
							{{ t("dashboardControl.recentTasks") }}
						</div>
						<div class="mt-1 text-sm text-slate-500">{{ t("dashboardControl.aiDevelopmentHint") }}</div>
					</div>
					<div class="flex items-center gap-2">
						<n-button quaternary circle :loading="refreshing" @click="refresh">
							<template #icon><Icon name="mdi:refresh" /></template>
						</n-button>
						<n-button type="primary" @click="router.push({ name: 'Code-Index' })">
							{{ t("dashboardControl.startDevelopment") }}
						</n-button>
					</div>
				</div>

				<n-alert v-if="tasksError" type="error" :show-icon="false" class="mb-4">
					<div class="flex items-center justify-between gap-3">
						<span>{{ t("dashboardControl.taskLoadFailed") }}</span>
						<n-button text type="primary" @click="loadTasks">{{ t("dashboardControl.retry") }}</n-button>
					</div>
				</n-alert>
				<n-spin :show="tasksLoading">
					<div v-if="!tasksLoading && visibleTasks.length" class="divide-y divide-slate-200">
						<button
							v-for="task in visibleTasks"
							:key="task.id"
							type="button"
							class="group flex w-full items-start gap-3 px-2 py-4 text-left transition hover:bg-slate-50"
							@click="openTask(task)"
						>
							<div class="min-w-0 flex-1">
								<div class="flex min-w-0 flex-wrap items-center gap-2">
									<CodeProjectIdentity
										class="max-w-[180px] text-[11px]"
										:project-id="task.projectId"
										:name="projectName(task)"
									/>
									<span class="fg-base-100 min-w-0 flex-1 truncate text-sm font-semibold">
										{{ task.title }}
									</span>
									<TaskStatusBadge :status="task.status" />
								</div>
								<CodeTaskMetaLine :task="task" class="mt-2" />
								<div
									v-if="task.summary.lastAgentMessage"
									class="mt-2 line-clamp-1 text-xs text-slate-500"
								>
									{{ task.summary.lastAgentMessage }}
								</div>
							</div>
							<div class="flex shrink-0 items-center gap-2 pt-1 text-xs text-slate-400">
								<span>{{ formatTime(task.updatedAt || task.createdAt) }}</span>
								<Icon name="mdi:chevron-right" class="transition group-hover:translate-x-0.5" />
							</div>
						</button>
					</div>
					<div
						v-else-if="!tasksLoading && !tasksError"
						class="flex items-center justify-between gap-4 rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 py-5"
					>
						<n-empty :description="t('dashboardControl.noTasks')"></n-empty>
						<n-button type="primary" secondary @click="router.push({ name: 'Code-Index' })">
							{{ t("dashboardControl.startDevelopment") }}
						</n-button>
					</div>
				</n-spin>
			</div>

			<slot />
		</div>

		<div class="space-y-8 self-start 2xl:sticky 2xl:top-6 2xl:col-start-2 2xl:row-start-2">
			<div class="bg-base-accent border-base-accent rounded-2xl p-5 shadow-sm">
				<div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">
					{{ t("dashboardControl.panelGuide") }}
				</div>
				<div class="fg-base-100 mt-3 text-lg font-semibold">{{ t("home.homeHelper") }}</div>
				<div class="mt-2 text-sm leading-6 text-slate-500">{{ t("home.homeHelperDesc") }}</div>
			</div>

			<div class="bg-base-accent border-base-accent rounded-2xl p-5 shadow-sm">
				<div class="mb-4 flex items-center justify-between gap-3">
					<div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">
						{{ t("dashboardControl.aiDevelopment") }}
					</div>
					<n-button text type="primary" size="small" @click="router.push({ name: 'Code-Index' })">
						{{ t("dashboardControl.enterWorkspace") }}
					</n-button>
				</div>
				<div class="grid grid-cols-2 gap-3">
					<div
						v-for="stat in stats"
						:key="stat.key"
						class="rounded-xl border border-[rgba(147,197,253,0.45)] bg-white/90 p-3"
					>
						<div class="flex items-center gap-2" :class="stat.tone">
							<Icon :name="stat.icon" :size="16" />
							<span class="text-xl font-bold">{{ stat.count }}</span>
						</div>
						<div class="mt-1 text-xs text-slate-500">{{ stat.label }}</div>
					</div>
				</div>
			</div>

			<div class="bg-base-100 rounded-2xl border border-slate-200 p-5 shadow-sm">
				<div class="mb-4">
					<div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">
						{{ t("dashboardControl.attentionTitle") }}
					</div>
					<div class="mt-2 text-sm leading-6 text-slate-500">{{ t("dashboardControl.attentionHint") }}</div>
				</div>
				<n-alert v-if="attentionError" type="error" :show-icon="false">
					<div class="flex items-center justify-between gap-2">
						<span>{{ t("dashboardControl.attentionLoadFailed") }}</span>
						<n-button text type="primary" size="small" @click="loadAttention">
							{{ t("dashboardControl.retry") }}
						</n-button>
					</div>
				</n-alert>
				<n-spin v-else :show="attentionLoading">
					<div v-if="!attentionLoading && attentionItems.length" class="space-y-3">
						<button
							v-for="item in attentionItems"
							:key="item.id"
							type="button"
							class="w-full rounded-xl border border-slate-200 bg-slate-50/80 p-3 text-left transition hover:border-blue-300 hover:bg-white"
							@click="openAttention(item)"
						>
							<div class="flex items-start gap-2">
								<Icon
									:name="
										item.severity === 'error'
											? 'mdi:alert-circle-outline'
											: 'mdi:shield-alert-outline'
									"
									:class="item.severity === 'error' ? 'text-red-500' : 'text-amber-500'"
								/>
								<div class="min-w-0 flex-1">
									<div class="fg-base-100 text-sm font-semibold">
										{{ t(`dashboardControl.attentionType_${item.type}`) }}
									</div>
									<div v-if="item.summary" class="mt-1 line-clamp-2 text-xs leading-5 text-slate-500">
										{{ item.summary }}
									</div>
									<div class="mt-2 flex items-center justify-between gap-2 text-xs text-slate-400">
										<span>{{ formatTime(item.updatedAt) }}</span>
										<span class="text-blue-600">{{ t("dashboardControl.openItem") }}</span>
									</div>
								</div>
							</div>
						</button>
					</div>
					<div
						v-else-if="!attentionLoading"
						class="rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 py-8 text-center text-sm text-slate-400"
					>
						{{ t("dashboardControl.attentionEmpty") }}
					</div>
				</n-spin>
			</div>
		</div>
	</section>
</template>
