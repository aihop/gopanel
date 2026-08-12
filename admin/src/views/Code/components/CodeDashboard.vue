<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useIntervalFn, useStorage } from "@vueuse/core"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { AIProject } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { setAITaskArchived } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import {
	filterCodeDashboardTasksByProject,
	groupCodeDashboardTasks,
	matchesCodeDashboardFilter,
	sortCodeTasksStably,
	type CodeDashboardBucket,
} from "../codeDashboardBuckets"
import { useCodeTaskPolling } from "../useCodeTaskPolling"
import CodeDashboardProjectList from "./CodeDashboardProjectList.vue"
import SessionHistoryDrawer from "./SessionHistoryDrawer.vue"
import CodeTaskDetailPane from "./CodeTaskDetailPane.vue"

const props = defineProps<{ projects: AIProject[]; loading: boolean; loadError: boolean }>()
const emit = defineEmits<{
	retry: []
	createProject: []
	createTask: [projectId: number]
	projectAction: [action: string, projectId: number]
	openTask: [task: CodeTaskListItem]
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()

const tasks = ref<CodeTaskListItem[]>([])
const taskTotal = ref(0)
const tasksInitialLoading = ref(true)
const tasksLoadError = ref(false)
const activeFilter = ref<CodeDashboardBucket | "delivering" | null>(null)
const selectedProjectId = ref<number | null>(null)
const showArchived = ref(false)
const selectedTaskId = ref<number | null>(null)
const showHistoryDrawer = ref(false)
// 折叠状态存起来：习惯用宽终端的人不该每次进页面都再折一次。
const listCollapsed = useStorage("code-dashboard-list-collapsed", false)
// 分组要按「现在」判断今天，用一个随轮询推进的时间戳，跨零点也不会停在昨天。
const now = ref(new Date())

// 面板是常驻页面，比工作台松：5 秒一轮，每 6 轮（30 秒）才带一次 git 汇总。
// git 汇总要按会话读工作区算 diff，跨项目每轮都拉会把低配机器压垮。
const { fetchTasks, fetchTasksFast } = useCodeTaskPolling(
	computed(() => 0),
	tasks,
	taskTotal,
	() => {
		tasksLoadError.value = true
	},
	// 全空闲时降到 20 秒，页面切走时不发请求 —— 面板是常驻页，不能一直空转。
	{
		intervalMs: 5000,
		gitEveryPolls: 6,
		limit: 50,
		allProjects: true,
		archived: showArchived,
		idleIntervalMs: 20000,
	},
)

const refreshTasks = async (silent = true) => {
	// 失败标记在每次显式刷新前清掉，成功后 onError 不会再置回来，横幅自己就消失了。
	if (!silent) tasksLoadError.value = false
	now.value = new Date()
	await fetchTasks(silent, true)
}

// 「今天」是按本地日历判断的，时间戳得自己往前走，否则跨零点会一直停在昨天。
useIntervalFn(() => (now.value = new Date()), 30000)

onMounted(async () => {
	tasksLoadError.value = false
	await fetchTasksFast(false)
	tasksInitialLoading.value = false
})

const initialLoading = computed(() => props.loading || tasksInitialLoading.value)

const projectNameById = computed(() => {
	const map = new Map<number, string>()
	for (const project of props.projects) map.set(project.id, project.name)
	return map
})

const selectedProjectName = computed(
	() => props.projects.find(project => project.id === selectedProjectId.value)?.name || "",
)
const handleProjectFilterSelect = (key: string) => {
	const id = Number(key)
	selectedProjectId.value = id > 0 ? id : null
}
const visibleProjects = computed(() =>
	selectedProjectId.value ? props.projects.filter(project => project.id === selectedProjectId.value) : props.projects,
)
const projectTasks = computed(() => filterCodeDashboardTasksByProject(tasks.value, selectedProjectId.value))
const grouped = computed(() => groupCodeDashboardTasks(projectTasks.value, now.value))

const stats = computed(() => [
	{
		key: "active" as const,
		labelKey: "code.dashboardActive",
		icon: "mdi:play-circle-outline",
		count: grouped.value.active.length,
		tone: "running",
	},
	{
		key: "attention" as const,
		labelKey: "code.dashboardAttention",
		icon: "mdi:shield-alert-outline",
		count: grouped.value.attention.length,
		tone: "attention",
	},
	{
		key: "delivering" as const,
		labelKey: "code.dashboardDelivering",
		icon: "mdi:source-merge",
		count: grouped.value.deliveringCount,
		tone: "delivering",
	},
	{
		key: "doneToday" as const,
		labelKey: "code.dashboardDoneToday",
		icon: "mdi:check-circle-outline",
		count: grouped.value.doneToday.length,
		tone: "done",
	},
])

// 项目内仍按稳定规则排序，不按状态拆成多个区块。
// 任务状态变化时只更新行内徽标，不会在“运行中/今日完成”之间跳来跳去。
const visibleTasks = computed(() => {
	const filter = activeFilter.value
	const scoped = filter
		? projectTasks.value.filter(task => matchesCodeDashboardFilter(task, filter, now.value))
		: projectTasks.value
	return sortCodeTasksStably(scoped)
})

const selectedTask = computed(() => visibleTasks.value.find(task => task.id === selectedTaskId.value) || null)

// 选中的任务被筛掉、归档或删掉时，落到当前可见的第一条，右边不会停在一个看不见的任务上。
// 只在「没有选中」或「选中的已不可见」时才动，避免每轮轮询把用户的选择顶掉。
watch(
	visibleTasks,
	list => {
		if (!list.length) {
			selectedTaskId.value = null
			return
		}
		if (list.some(task => task.id === selectedTaskId.value)) return
		selectedTaskId.value = list[0].id
	},
	{ immediate: true },
)

const toggleFilter = (key: CodeDashboardBucket | "delivering") => {
	activeFilter.value = activeFilter.value === key ? null : key
}

// 切归档视图要立刻换列表，不能等下一轮轮询（最长 5 秒）才刷。
// 顺手清掉状态筛选：归档列表里「今日完成」这种口径没有意义。
const toggleArchivedView = () => {
	showArchived.value = !showArchived.value
	activeFilter.value = null
	tasks.value = []
	void refreshTasks(false)
}

const archiving = ref<number | null>(null)

const toggleArchived = async (task: CodeTaskListItem) => {
	if (archiving.value) return
	archiving.value = task.id
	const nextArchived = !showArchived.value
	try {
		const response = await setAITaskArchived(task.id, nextArchived)
		if (response.code !== 0) throw new Error(response.message)
		message.success(t(nextArchived ? "code.taskArchived" : "code.taskUnarchived"))
		await refreshTasks(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.taskArchiveFailed"))
	} finally {
		archiving.value = null
	}
}
</script>

<template>
  <div
    class="relative flex min-h-0 flex-1 flex-col"
    :aria-busy="initialLoading"
  >
    <div
      v-if="initialLoading"
      class="flex min-h-0 flex-1 items-center justify-center"
      aria-hidden="true"
    >
      <n-spin
        size="small"
        :delay="150"
      />
    </div>
    <template v-else>
    <!--
      顶部只留一行：标题 + 状态数字（既是概览也是筛选器）+ 工具栏插槽。
      大标题和副标题去掉了 —— 面包屑已经写了「开发工作台」，
      在一个天天用的工作页面上再占 80px 讲一遍是纯装饰，那 80px 归终端。
      项目筛选紧跟标题，保持低视觉权重；默认仍展示所有项目的任务。
    -->
    <div class="mb-4 flex flex-wrap items-center gap-x-4 gap-y-2 px-5 md:px-7">
      <span class="shrink-0 text-base font-semibold tracking-[-0.01em] text-[var(--n-text-color)]">
        {{ t("code.workspace") }}
      </span>
      <div
        v-show="!showArchived"
        class="flex flex-1 flex-wrap items-center gap-2"
      >
        <button
          v-for="stat in stats"
          :key="stat.key"
          type="button"
          class="dashboard-stat flex items-center gap-2 rounded-full px-3.5 py-2"
          :class="[`dashboard-stat--${stat.tone}`, activeFilter === stat.key ? 'dashboard-stat--selected' : '']"
          @click="toggleFilter(stat.key)"
        >
          <Icon
            :name="stat.icon"
            :size="15"
            class="dashboard-stat__icon shrink-0"
          />
          <span class="dashboard-stat__count text-sm font-bold">{{ stat.count }}</span>
          <span class="text-xs text-[var(--n-text-color-3)]">{{ t(stat.labelKey) }}</span>
        </button>
        <n-button
          v-if="activeFilter"
          text
          size="tiny"
          type="primary"
          @click="activeFilter = null"
        >
          {{ t("code.dashboardClearFilter") }}
        </n-button>
      </div>
      <div
        v-show="showArchived"
        class="flex flex-1 items-center gap-2 text-sm text-[var(--n-text-color-2)]"
      >
        <Icon
          name="mdi:archive-outline"
          :size="16"
        />
        {{ t("code.dashboardArchivedTitle") }}
      </div>

      <n-button
        size="small"
        :secondary="showArchived"
        :quaternary="!showArchived"
        @click="toggleArchivedView()"
      >
        <template #icon>
          <Icon
            :name="showArchived ? 'mdi:arrow-left' : 'mdi:archive-outline'"
            :size="15"
          />
        </template>
        {{ showArchived ? t("code.dashboardBackToActive") : t("code.dashboardViewArchived") }}
      </n-button>

      <n-button
        quaternary
        circle
        size="small"
        :title="listCollapsed ? t('code.dashboardExpandList') : t('code.dashboardCollapseList')"
        @click="listCollapsed = !listCollapsed"
      >
        <template #icon>
          <Icon
            :name="listCollapsed ? 'mdi:dock-left' : 'mdi:dock-window'"
            :size="16"
          />
        </template>
      </n-button>

      <n-button
        quaternary
        circle
        size="small"
        :title="t('code.refresh')"
        @click="refreshTasks(false)"
      >
        <template #icon>
          <Icon
            name="mdi:refresh"
            :size="16"
          />
        </template>
      </n-button>

      <!-- 项目管理和新建项目由 Index 注入：它们属于项目，不属于任务列表 -->
      <slot name="toolbar" />
    </div>

    <n-alert
      v-if="loadError"
      type="error"
      :show-icon="false"
      class="mx-5 mb-3 md:mx-7"
    >
      <div class="flex items-center justify-between gap-3">
        <span>{{ t("code.projectLoadFailed") }}</span>
        <n-button
          text
          type="primary"
          @click="emit('retry')"
        >
          {{ t("code.retry") }}
        </n-button>
      </div>
    </n-alert>
    <n-alert
      v-else-if="tasksLoadError"
      type="error"
      :show-icon="false"
      class="mx-5 mb-3 md:mx-7"
    >
      <div class="flex items-center justify-between gap-3">
        <span>{{ t("code.dashboardTaskLoadFailed") }}</span>
        <n-button
          text
          type="primary"
          @click="refreshTasks(false)"
        >
          {{ t("code.retry") }}
        </n-button>
      </div>
    </n-alert>

    <!-- 主从：左边所有任务，右边选中任务的终端。切任务不跳页，只换右边。 -->
    <div
      class="dashboard-workbench grid min-h-0 flex-1 overflow-hidden border-t "
      :class="
        listCollapsed
          ? 'grid-cols-1 grid-rows-1'
          : 'grid-cols-1 grid-rows-[minmax(220px,2fr)_minmax(260px,3fr)] xl:grid-cols-[400px_minmax(0,1fr)] xl:grid-rows-1'
      "
    >
      <section
        v-if="!listCollapsed"
        class="dashboard-panel flex min-h-0 flex-col overflow-hidden"
      >
        <n-scrollbar class="min-h-0 flex-1">
          <!-- 左列独立滚动，所以这里的密度不吃终端高度，可以给足 -->
          <div class="py-2">
            <div
              v-if="!visibleTasks.length && (showArchived || !projects.length)"
              class="flex min-h-[200px] items-center justify-center"
            >
              <n-empty
                size="small"
                :description="showArchived ? t('code.dashboardNoArchived') : t('code.dashboardNoTasks')"
              >
                <template
                  v-if="!showArchived"
                  #extra
                >
                  <n-button
                    size="tiny"
                    type="primary"
                    @click="emit('createProject')"
                  >
                    {{ projects.length ? t("code.dashboardNoTasksHint") : t("code.createProject") }}
                  </n-button>
                </template>
              </n-empty>
            </div>
            <CodeDashboardProjectList
              v-else
              :projects="visibleProjects"
              :tasks="visibleTasks"
              :selected-task-id="selectedTaskId"
              :archived="showArchived"
              :archiving-task-id="archiving"
              @open="selectedTaskId = $event.id"
              @archive="toggleArchived"
              @open-workspace="emit('openTask', $event)"
              @create-task="emit('createTask', $event)"
              @project-action="(action, projectId) => emit('projectAction', action, projectId)"
              @refresh="refreshTasks(true)"
            />
          </div>
        </n-scrollbar>
      </section>

      <!--
        列表折叠后左边就没有「现在看的是哪条」了，这时才让详情区补一条紧凑的头。
        展开时不显示 —— 那会和选中行的信息重复，也白吃终端高度。
      -->
      <CodeTaskDetailPane
        :task="selectedTask"
        :project-name="selectedTask ? projectNameById.get(selectedTask.projectId) || '' : ''"
        :show-header="listCollapsed"
        @open-workspace="emit('openTask', $event)"
        @open-history="showHistoryDrawer = true"
        @task-created="refreshTasks(true)"
      />
    </div>

    <SessionHistoryDrawer
      v-model:show="showHistoryDrawer"
      :session-id="selectedTask?.sessionId || null"
      :task-id="selectedTask?.id || null"
    />
    </template>
  </div>
</template>

<style scoped>
.dashboard-panel {
	background: color-mix(in srgb, var(--n-color) 97%, transparent);
	border-bottom: 1px solid var(--n-border-color);
}
@media (min-width: 1280px) {
	.dashboard-panel {
		border-right: 1px solid var(--n-border-color);
		border-bottom: 0;
	}
}
.dashboard-stat {
	--stat-accent: #94a3b8;
	background: color-mix(in srgb, var(--n-color) 97%, transparent);
	border: 1px solid color-mix(in srgb, var(--n-border-color) 92%, transparent);
	box-shadow: 0 4px 12px rgb(15 23 42 / 3.5%);
	transition:
		border-color 0.18s ease,
		box-shadow 0.18s ease,
		background-color 0.18s ease;
}
.dashboard-stat:hover {
	border-color: color-mix(in srgb, var(--stat-accent) 42%, var(--n-border-color));
	box-shadow: 0 8px 20px rgb(15 23 42 / 7%);
}
.dashboard-stat--selected {
	border-color: var(--stat-accent);
	background: color-mix(in srgb, var(--stat-accent) 10%, var(--n-color));
}
.dashboard-stat--running {
	--stat-accent: #10b981;
}
.dashboard-stat--attention {
	--stat-accent: #f59e0b;
}
.dashboard-stat--delivering {
	--stat-accent: #3b82f6;
}
.dashboard-stat--done {
	--stat-accent: #64748b;
}
.dashboard-stat__icon,
.dashboard-stat__count {
	color: var(--stat-accent);
}
</style>
