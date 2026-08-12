<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import CodeTaskUserSnippet from "./CodeTaskUserSnippet.vue"
import CodeTaskFocusMarker from "./CodeTaskFocusMarker.vue"
import CodeTaskMetaLine from "./CodeTaskMetaLine.vue"
import ProjectBranchManager from "./ProjectBranchManager.vue"
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
	createTask: []
}>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
    <section class="ai-workspace-task-history mt-3 flex min-h-0 flex-1 flex-col overflow-hidden">
      <div class="flex items-center justify-between px-4 pb-2 pt-1">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
          {{ t("code.taskHistory") }}
        </div>
        <div class="flex items-center gap-2">
          <n-button
            size="small"
            type="primary"
            class="!rounded-lg"
            @click="emit('createTask')"
          >
            <template #icon>
              <Icon
                name="mdi:robot-outline"
                :size="14"
              />
            </template>
            {{ t("code.aiTaskShort") }}
          </n-button>
          <div class="text-xs text-slate-400">
            {{ t("code.taskCount", { count: taskTotal }) }}
          </div>
        </div>
      </div>
      <n-scrollbar
        trigger="none"
        class="ai-workspace-task-scrollbar min-h-0 flex-1"
      >
        <div class="px-2.5 pb-3 pr-3.5">
          <div
            v-if="tasks.length === 0"
            class="ai-workspace-task-empty flex min-h-[180px] items-center justify-center"
          >
            <n-empty :description="t('code.noProjectHistory')" />
          </div>
          <div
            v-else
            class="space-y-1"
          >
            <div
              v-for="task in tasks"
              :key="task.id"
              class="ai-workspace-task-row group/task relative flex cursor-pointer items-stretch justify-between gap-2.5 rounded-xl py-2.5 pl-2 pr-3 transition-colors duration-200"
              :class="currentTaskId === task.id ? 'ai-workspace-task-row--active' : ''"
              @click="emit('selectTask', task)"
            >
              <CodeTaskFocusMarker :active="currentTaskId === task.id" />
              <div class="min-w-0 flex-1">
                <div
                  class="flex min-w-0 items-center gap-1.5 text-sm font-semibold text-slate-800"
                  :title="task.title"
                >
                  <Icon
                    v-if="task.agentName === 'terminal'"
                    name="mdi:console-line"
                    :size="14"
                    class="shrink-0 text-slate-500"
                  />
                  <span class="truncate">{{ task.title }}</span>
                </div>
                <CodeTaskMetaLine
                  :task="task"
                  class="mt-1.5"
                >
                  <template #lead>
                    <TaskApprovalAction
                      class="shrink-0"
                      :task="task"
                      @approved="emit('refreshTasks')"
                    />
                  </template>
                </CodeTaskMetaLine>

                <CodeTaskUserSnippet :task="task" />
              </div>
              <!-- self-start：行改成 items-stretch 让竖线通高后，这里要单独顶对齐 -->
              <div
                class="self-start opacity-100 transition-opacity md:opacity-0 md:group-hover/task:opacity-100"
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
                    <template #icon>
                      <Icon name="mdi:dots-horizontal" />
                    </template>
                  </n-button>
                </n-dropdown>
              </div>
            </div>
          </div>
        </div>
      </n-scrollbar>
    </section>

    <ProjectBranchManager :project-id="projectId" />
  </div>
</template>

<style scoped>
.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical) {
	right: 3px !important;
	width: 3px !important;
}

.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail__scrollbar) {
	width: 3px !important;
	background-color: rgba(100, 116, 139, 0.28) !important;
}

/*
	竖线已经交给 CodeTaskFocusMarker（flex 里的真实元素），
	原来的 ::before 贴在 left:0，被行的 rounded-xl 吞掉了，所以一直看不出选中在哪。
	底色只是陪衬，主信号是那条竖线。
*/
.ai-workspace-task-row--active {
	background-color: color-mix(in srgb, var(--primary-color) 12%, transparent);
}

/* 悬停时先给一条淡的，提示「点这里会选中」 */
.ai-workspace-task-row:hover :deep(.code-task-focus-marker:not(.code-task-focus-marker--on)) {
	background: color-mix(in srgb, var(--primary-color) 28%, transparent);
}

/*
	:not(--active) 是必要的：原来用的是 Tailwind 的 hover:bg-slate-200/45，
	hover 变体在生成顺序上排在 bg-blue-50/80 之后，
	鼠标划过选中行会把它洗白 —— 正好和「锁住焦点」相反。
*/
.ai-workspace-task-row:not(.ai-workspace-task-row--active):hover {
	background-color: rgb(226 232 240 / 45%);
}

:global(.theme-dark) .ai-workspace-task-row:not(.ai-workspace-task-row--active):hover {
	background-color: color-mix(in srgb, var(--fg-secondary-color) 10%, transparent) !important;
}

:global(.theme-dark) .ai-workspace-task-row--active {
	background-color: color-mix(in srgb, var(--primary-color) 16%, transparent) !important;
}

:global(.theme-dark) .ai-workspace-task-btn {
	background-color: color-mix(in srgb, var(--fg-secondary-color) 15%, transparent) !important;
}
</style>
