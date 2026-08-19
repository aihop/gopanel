<script setup lang="ts">
import { computed, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { AIProject } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { getAITasks, setAITaskArchived } from "@/api/modules/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { codeDashboardRecentStatus, codeTaskTimestamp } from "../codeDashboardBuckets"
import { codeProjectColor } from "../projectColor"
import Icon from "@/components/common/Icon.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"

const props = defineProps<{ projects: AIProject[] }>()
const emit = defineEmits<{ restored: [task: CodeTaskListItem] }>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()
const show = ref(false)
const tasks = ref<CodeTaskListItem[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadError = ref(false)
const restoringTaskId = ref<number | null>(null)
const projectNames = computed(() => new Map(props.projects.map(project => [project.id, project.name])))

const taskTime = (task: CodeTaskListItem) =>
	new Date(codeTaskTimestamp(task)).toLocaleString(undefined, {
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
	})

const loadArchivedTasks = async (reset = true) => {
	if (loading.value) return
	const nextPage = reset ? 1 : page.value + 1
	loading.value = true
	loadError.value = false
	try {
		const response = await getAITasks({
			page: nextPage,
			limit: 20,
			projectId: 0,
			includeGit: false,
			archived: 1,
		})
		if (response.code !== 0) throw new Error(response.message)
		const items = response.data.items || []
		tasks.value = reset ? items : [...tasks.value, ...items]
		total.value = response.data.total || 0
		page.value = nextPage
	} catch (error) {
		if (reset) loadError.value = true
		else message.error(error instanceof Error && error.message ? error.message : t("code.dashboardTaskLoadFailed"))
	} finally {
		loading.value = false
	}
}

const updateShow = (value: boolean) => {
	show.value = value
	if (value) void loadArchivedTasks(true)
}

const restoreTask = async (task: CodeTaskListItem) => {
	if (restoringTaskId.value) return
	restoringTaskId.value = task.id
	try {
		const response = await setAITaskArchived(task.id, false)
		if (response.code !== 0) throw new Error(response.message)
		tasks.value = tasks.value.filter(item => item.id !== task.id)
		total.value = Math.max(0, total.value - 1)
		show.value = false
		message.success(t("code.taskUnarchived"))
		emit("restored", task)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.taskArchiveFailed"))
	} finally {
		restoringTaskId.value = null
	}
}
</script>

<template>
  <n-popover
    :show="show"
    trigger="click"
    placement="bottom-start"
    :show-arrow="false"
    style="width: min(360px, calc(100vw - 24px))"
    @update:show="updateShow"
  >
    <template #trigger>
      <n-button
        quaternary
        circle
        size="small"
        :title="t('code.dashboardViewArchived')"
      >
        <template #icon>
          <Icon
            name="mdi:archive-outline"
            :size="16"
          />
        </template>
      </n-button>
    </template>

    <div class="p-1">
      <div class="mb-2 flex items-center justify-between gap-3 px-1">
        <span class="text-sm font-semibold text-[var(--n-text-color)]">
          {{ t("code.dashboardViewArchived") }}
        </span>
        <span
          v-if="tasks.length"
          class="text-xs text-[var(--n-text-color-3)]"
        >{{ t("code.dashboardArchivedCount", { loaded: tasks.length, total }) }}</span>
      </div>

      <n-spin :show="loading">
        <div
          v-if="loadError"
          class="flex min-h-28 flex-col items-center justify-center gap-2"
        >
          <span class="text-xs text-red-500">{{ t("code.dashboardTaskLoadFailed") }}</span>
          <n-button
            size="tiny"
            @click="loadArchivedTasks(true)"
          >
            {{ t("code.retry") }}
          </n-button>
        </div>
        <n-empty
          v-else-if="!loading && !tasks.length"
          size="small"
          class="py-6"
          :description="t('code.dashboardNoArchived')"
        />
        <n-scrollbar
          v-else
          style="max-height: min(420px, calc(100vh - 160px))"
        >
          <div class="space-y-1 pr-1">
            <button
              v-for="task in tasks"
              :key="task.id"
              type="button"
              class="group/archive flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left transition-colors hover:bg-[var(--n-color-embedded)]"
              :disabled="restoringTaskId !== null"
              :aria-label="t('code.taskUnarchive')"
              @click="restoreTask(task)"
            >
              <n-tooltip>
                <template #trigger>
                  <span
                    class="h-2.5 w-2.5 shrink-0 rounded-[2px]"
                    :style="{ backgroundColor: codeProjectColor(task.projectId) }"
                  />
                </template>
                {{ projectNames.get(task.projectId) || t("code.projectFallback") }}
              </n-tooltip>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-xs font-medium text-[var(--n-text-color)]">
                  {{ task.title }}
                </span>
                <span class="mt-0.5 block text-[10px] text-[var(--n-text-color-3)]">
                  {{ taskTime(task) }}
                </span>
              </span>
              <TaskStatusBadge
                :status="codeDashboardRecentStatus(task)"
                compact
              />
              <n-spin
                v-if="restoringTaskId === task.id"
                :size="13"
              />
              <Icon
                v-else
                name="mdi:archive-arrow-up-outline"
                :size="15"
                class="shrink-0 text-[var(--n-text-color-3)] opacity-50 transition-opacity group-hover/archive:opacity-100"
              />
            </button>
            <n-button
              v-if="tasks.length < total"
              block
              quaternary
              size="small"
              :loading="loading"
              @click="loadArchivedTasks(false)"
            >
              {{ t("code.dashboardLoadMoreArchived") }}
            </n-button>
          </div>
        </n-scrollbar>
      </n-spin>
    </div>
  </n-popover>
</template>
