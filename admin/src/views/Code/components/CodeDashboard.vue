<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useStorage } from "@vueuse/core"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { AIProject, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { CodeProjectDropPosition } from "../codeProjectOrder"
import { getAITasks, setAITaskArchived } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { mergeCodeDashboardTasks, sortCodeTasksStably } from "../codeDashboardBuckets"
import { useCodeTaskPolling } from "../useCodeTaskPolling"
import CodeDashboardProjectList from "./CodeDashboardProjectList.vue"
import CodeDashboardArchivedTasks from "./CodeDashboardArchivedTasks.vue"
import SessionHistoryDrawer from "./SessionHistoryDrawer.vue"
import CodeTaskDetailPane from "./CodeTaskDetailPane.vue"

const props = defineProps<{
	projects: AIProject[]
	loading: boolean
	loadError: boolean
	immersive?: boolean
	pendingSession?: CodeSession | null
}>()
const emit = defineEmits<{
	retry: []
	createProject: []
	createTask: [projectId: number]
	projectAction: [action: string, projectId: number]
	reorderProject: [projectId: number, targetProjectId: number, position: CodeProjectDropPosition]
	openTask: [task: CodeTaskListItem]
	sessionResolved: []
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()

const tasks = ref<CodeTaskListItem[]>([])
const recentTaskCandidates = ref<CodeTaskListItem[]>([])
const taskTotal = ref(0)
const tasksInitialLoading = ref(true)
const tasksLoadError = ref(false)
const selectedTaskId = ref<number | null>(null)
const pendingRestoredTaskId = ref<number | null>(null)
const pendingCreatedTaskId = ref<number | null>(null)
const showHistoryDrawer = ref(false)
// 折叠状态存起来：习惯用宽终端的人不该每次进页面都再折一次。
const listCollapsed = useStorage("code-dashboard-list-collapsed", false)

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
		idleIntervalMs: 20000,
		selectedTaskId,
	},
)

let recentRequest: Promise<void> | null = null
const fetchRecentTasks = (silent = true) => {
	if (recentRequest) return recentRequest
	recentRequest = getAITasks({ page: 1, limit: 7, projectId: 0, includeGit: false, order: "recent" })
		.then(response => {
			if (response.code !== 0) throw new Error(response.message)
			recentTaskCandidates.value = response.data.items || []
		})
		.catch(error => {
			if (!silent) {
				tasksLoadError.value = true
				message.error(error instanceof Error ? error.message : t("code.dashboardTaskLoadFailed"))
			}
		})
		.finally(() => {
			recentRequest = null
		})
	return recentRequest
}

const refreshTasks = async (silent = true) => {
	// 失败标记在每次显式刷新前清掉，成功后 onError 不会再置回来，横幅自己就消失了。
	if (!silent) tasksLoadError.value = false
	await Promise.all([fetchTasks(silent, "full"), fetchRecentTasks(silent)])
}

onMounted(async () => {
	tasksLoadError.value = false
	await Promise.all([fetchTasksFast(false), fetchRecentTasks(false)])
	tasksInitialLoading.value = false
})

watch(tasks, () => void fetchRecentTasks(true))

const initialLoading = computed(() => props.loading || tasksInitialLoading.value)

const projectNameById = computed(() => {
	const map = new Map<number, string>()
	for (const project of props.projects) map.set(project.id, project.name)
	return map
})

const allTasks = computed(() => mergeCodeDashboardTasks(tasks.value, recentTaskCandidates.value))

// 仍按稳定规则排序，不按状态拆成多个区块。
// 任务状态变化时只更新行内徽标，不会在“运行中/今日完成”之间跳来跳去。
const visibleTasks = computed(() => {
	return sortCodeTasksStably(allTasks.value)
})

const selectedTask = computed(() => visibleTasks.value.find(task => task.id === selectedTaskId.value) || null)

watch(
	() => props.pendingSession?.id,
	sessionId => {
		if (sessionId) selectedTaskId.value = null
	},
)

// 选中的任务被筛掉、归档或删掉时，落到当前可见的第一条，右边不会停在一个看不见的任务上。
// 只在「没有选中」或「选中的已不可见」时才动，避免每轮轮询把用户的选择顶掉。
watch(
	visibleTasks,
	list => {
		if (pendingCreatedTaskId.value) {
			const createdTask = list.find(task => task.id === pendingCreatedTaskId.value)
			if (createdTask) {
				selectedTaskId.value = createdTask.id
				pendingCreatedTaskId.value = null
				emit("sessionResolved")
			}
			return
		}
		if (props.pendingSession) return
		if (pendingRestoredTaskId.value) {
			if (list.some(task => task.id === pendingRestoredTaskId.value)) {
				selectedTaskId.value = pendingRestoredTaskId.value
				pendingRestoredTaskId.value = null
			}
			return
		}
		if (!list.length) {
			selectedTaskId.value = null
			return
		}
		if (list.some(task => task.id === selectedTaskId.value)) return
		selectedTaskId.value = list[0].id
	},
	{ immediate: true },
)

const archiving = ref<number | null>(null)

const toggleArchived = async (task: CodeTaskListItem) => {
	if (archiving.value) return
	archiving.value = task.id
	try {
		const response = await setAITaskArchived(task.id, true)
		if (response.code !== 0) throw new Error(response.message)
		message.success(t("code.taskArchived"))
		await refreshTasks(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.taskArchiveFailed"))
	} finally {
		archiving.value = null
	}
}

const restoreArchivedTask = async (task: CodeTaskListItem) => {
	pendingRestoredTaskId.value = task.id
	await refreshTasks(true)
}

const activateCreatedTask = async (taskId: number) => {
	pendingCreatedTaskId.value = taskId
	await refreshTasks(true)
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
    <!-- 标题后放视图工具，项目管理与新建项目靠右保持主操作层级。 -->
    <div
      class="flex flex-wrap items-center gap-1.5"
      :class="immersive ? 'mb-2 px-3' : 'mb-3 px-4 md:px-5'"
    >
      <span class="shrink-0 text-sm font-medium tracking-[0.01em] text-[var(--n-text-color-2)]">
        {{ t("code.workspace") }}
      </span>
      <n-button
        quaternary
        circle
        size="tiny"
        :title="listCollapsed ? t('code.dashboardExpandList') : t('code.dashboardCollapseList')"
        @click="listCollapsed = !listCollapsed"
      >
        <template #icon>
          <Icon
            :name="listCollapsed ? 'mdi:chevron-right' : 'mdi:chevron-left'"
            :size="16"
            class="opacity-50"
          />
        </template>
      </n-button>

      <CodeDashboardArchivedTasks
        :projects="projects"
        @restored="restoreArchivedTask"
      />

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

      <div class="ml-auto flex items-center gap-2">
        <!-- 项目管理和新建项目由 Index 注入：它们属于项目，不属于任务列表 -->
        <slot name="toolbar" />
      </div>
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

    <!-- 主从：左边会话轨，右边就是工作台。切任务不跳页。 -->
    <div
      class="dashboard-workbench grid min-h-0 flex-1 overflow-hidden border-t border-[color-mix(in_srgb,var(--n-border-color)_70%,transparent)]"
      :class="[
        listCollapsed
          ? 'grid-cols-1 grid-rows-1'
          : 'dashboard-workbench--split grid-cols-1 grid-rows-[minmax(180px,1fr)_minmax(320px,2fr)] lg:grid-rows-1',
      ]"
      :style="{ '--dashboard-sidebar-width': '260px' }"
    >
      <section
        v-if="!listCollapsed"
        class="dashboard-panel flex min-h-0 flex-col overflow-hidden"
      >
        <n-scrollbar class="min-h-0 flex-1">
          <div class="py-1">
            <div
              v-if="!visibleTasks.length && !projects.length"
              class="flex min-h-[200px] items-center justify-center"
            >
              <n-empty
                size="small"
                :description="t('code.dashboardNoTasks')"
              >
                <template
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
              :projects="projects"
              :tasks="visibleTasks"
              :empty-label="t('code.dashboardNoProjectTasks')"
              :selected-task-id="selectedTaskId"
              :archived="false"
              :archiving-task-id="archiving"
              @open="selectedTaskId = $event.id"
              @archive="toggleArchived"
              @open-workspace="emit('openTask', $event)"
              @create-task="emit('createTask', $event)"
              @project-action="(action, projectId) => emit('projectAction', action, projectId)"
              @reorder-project="(projectId, targetProjectId, position) => emit('reorderProject', projectId, targetProjectId, position)"
              @refresh="refreshTasks(true)"
            />
          </div>
        </n-scrollbar>
      </section>

      <CodeTaskDetailPane
        :task="pendingSession ? null : selectedTask"
        :session="pendingSession"
        :project-name="pendingSession
          ? projectNameById.get(pendingSession.projectId) || ''
          : selectedTask
            ? projectNameById.get(selectedTask.projectId) || ''
            : ''"
        @open-history="showHistoryDrawer = true"
        @task-created="activateCreatedTask"
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
	background: transparent;
	border-bottom: 1px solid color-mix(in srgb, var(--n-border-color) 70%, transparent);
}
@media (min-width: 1024px) {
	.dashboard-workbench--split {
		grid-template-columns: var(--dashboard-sidebar-width) minmax(0, 1fr);
	}

	.dashboard-panel {
		border-right: 1px solid var(--n-border-color);
		border-bottom: 0;
	}
}
</style>
