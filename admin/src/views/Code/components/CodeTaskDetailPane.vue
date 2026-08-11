<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { useCodeWorkspaceFullscreen } from "../useCodeWorkspaceFullscreen"
import CodeTerminal from "./CodeTerminal.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"
import { CODE_TERMINAL_POOL_SIZE, codeTerminalIdentity } from "./codeTerminalSession"

const props = defineProps<{ task: CodeTaskListItem | null; projectName?: string; showHeader?: boolean }>()
const emit = defineEmits<{
	openWorkspace: [task: CodeTaskListItem]
	openHistory: [task: CodeTaskListItem]
	taskCreated: [taskId: number]
}>()
const { t } = useI18n({ messages: codeProjectMessages })

// 复用工作台那套全屏（含 Escape 退出），不另写一份。
const { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen } = useCodeWorkspaceFullscreen(t)

// 和工作台用同一套身份：在面板里看过的任务，进工作台再看还是同一条终端实例。
const terminalIdentity = computed(() =>
	props.task ? codeTerminalIdentity(props.task.sessionId || null, props.task.id) : "",
)
</script>

<template>
  <!--
		展开状态下刻意没有头部。任务标题、项目、状态、token、diff 在左边选中的那一行上全都有，
		在终端上面再摆一条只是把同样的信息讲第二遍，还要吃掉 ~64px 的终端高度。
		只有左边列表被折叠起来、看不到选中行时，才补一条紧凑的单行头。
		终端自己的工具条（codex 状态 / 接管终端 / 重连）保留 —— 那些是操作，不是重复的状态。
		全屏按钮浮在右上角，不占垂直空间。
	-->
  <section
    class="detail-pane flex min-h-0 flex-col overflow-hidden rounded-[24px]"
    :class="isWorkspaceFullscreen ? 'detail-pane--fullscreen' : ''"
  >
    <div
      v-if="!task"
      class="flex min-h-0 flex-1 items-center justify-center p-8"
    >
      <n-empty :description="t('code.detailNoSelection')">
        <template #extra>
          <span class="text-xs text-[var(--n-text-color-3)]">{{ t("code.detailNoSelectionHint") }}</span>
        </template>
      </n-empty>
    </div>

    <template v-else>
      <header
        v-if="showHeader"
        class="detail-pane__header flex shrink-0 items-center gap-2 px-4 py-2"
      >
        <Icon
          v-if="task.agentName === 'terminal'"
          name="mdi:console-line"
          :size="14"
          class="shrink-0 text-slate-400"
        />
        <span
          class="truncate text-sm font-semibold text-[var(--n-text-color)]"
          :title="task.title"
        >
          {{ task.title }}
        </span>
        <TaskStatusBadge
          :status="task.status"
          class="shrink-0"
        />
        <span
          v-if="projectName"
          class="shrink-0 text-[11px] text-[var(--n-text-color-3)]"
        >
          {{ projectName }}
        </span>
      </header>

      <div class="relative min-h-0 flex-1 bg-[#1e1e1e]">
        <!-- 对话和全屏都属于当前执行任务，浮在终端上避免占用输出高度。 -->
        <div class="detail-pane__floating absolute right-3 top-2 z-[2] flex items-center gap-1">
          <n-button
            v-if="task.agentName !== 'terminal'"
            quaternary
            circle
            size="tiny"
            :title="t('code.conversationHistory')"
            @click="emit('openHistory', task)"
          >
            <template #icon>
              <Icon
                name="mdi:message-text-clock-outline"
                :size="16"
              />
            </template>
          </n-button>
          <n-button
            quaternary
            circle
            size="tiny"
            :title="fullscreenLabel"
            @click="toggleWorkspaceFullscreen"
          >
            <template #icon>
              <Icon
                :name="isWorkspaceFullscreen ? 'mdi:fullscreen-exit' : 'mdi:fullscreen'"
                :size="16"
              />
            </template>
          </n-button>
        </div>

        <KeepAlive :max="CODE_TERMINAL_POOL_SIZE">
          <CodeTerminal
            :key="terminalIdentity"
            :task-id="task.id"
            :session-id="task.sessionId || null"
            @task-created="emit('taskCreated', $event)"
            @new-session="emit('openWorkspace', task)"
          />
        </KeepAlive>
      </div>
    </template>
  </section>
</template>

<style scoped>
/* 和左列同一套：97% 底色 + 92% 边框 + 柔和投影，两栏才像一对 */
.detail-pane {
	background: color-mix(in srgb, var(--n-color) 97%, transparent);
	border: 1px solid color-mix(in srgb, var(--n-border-color) 92%, transparent);
	box-shadow: 0 8px 24px rgb(15 23 42 / 4.5%);
}

.detail-pane__header {
	border-bottom: 1px solid color-mix(in srgb, var(--n-border-color) 82%, transparent);
}

.detail-pane--fullscreen {
	position: fixed;
	z-index: 1000;
	border-radius: 0;
	box-shadow: none;
	inset: 0;
}

/* 浮在终端上，所以要自己有底色才看得清，否则压在深色输出上 */
.detail-pane__floating :deep(.n-button) {
	color: rgb(203 213 225);
	background: rgb(30 41 59 / 72%);
}

.detail-pane__floating :deep(.n-button:hover) {
	color: #fff;
	background: rgb(51 65 85 / 92%);
}
</style>
