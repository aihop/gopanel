<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeExecutor, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { getCodeExecutors } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { executorSupportsStructuredTurn, findExecutorById } from "../codeStructuredTurn"
import { useCodeWorkspaceFullscreen } from "../useCodeWorkspaceFullscreen"
import CodeConversationPanel from "./CodeConversationPanel.vue"
import CodeProjectIdentity from "./CodeProjectIdentity.vue"
import CodeTerminal from "./CodeTerminal.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"
import { CODE_TERMINAL_POOL_SIZE, codeTerminalIdentity } from "./codeTerminalSession"

const props = defineProps<{
	task: CodeTaskListItem | null
	session?: CodeSession | null
	projectName?: string
	showHeader?: boolean
}>()
const emit = defineEmits<{
	openWorkspace: [task: CodeTaskListItem]
	openHistory: [task: CodeTaskListItem]
	taskCreated: [taskId: number]
}>()
const { t } = useI18n({ messages: codeProjectMessages })

// 复用工作台那套全屏（含 Escape 退出），不另写一份。
const { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen } = useCodeWorkspaceFullscreen(t)

// 和工作台用同一套身份算法，但两边各有各的 KeepAlive 树——key 相同也不共享实例，
// 面板和工作台各有自己的 xterm 缓存；隐藏实例会释放控制并断开，重新显示时增量接回，
// 避免两棵 KeepAlive 树同时占着同一个 PTY 的控制权。
const terminalIdentity = computed(() =>
	codeTerminalIdentity(props.session?.id || props.task?.sessionId || null, props.task?.id || null),
)
const sessionId = computed(() => props.session?.id || props.task?.sessionId || null)
const executorId = computed(() => props.session?.agentName || props.task?.agentName || "")
const executors = ref<CodeExecutor[]>([])
const showNativeCli = ref(false)
const structuredTurn = computed(() =>
	executorSupportsStructuredTurn(findExecutorById(executors.value, executorId.value)),
)
const showConversation = computed(() => {
	if (showNativeCli.value || !executorId.value) return false
	// 执行器列表还没回来时先不挂 PTY，避免 Grok 详情一打开就拉起它自己的 TUI。
	if (!executors.value.length) return executorId.value !== "terminal"
	return structuredTurn.value
})

watch(executorId, () => {
	showNativeCli.value = false
})
onMounted(async () => {
	try {
		const response = await getCodeExecutors()
		if (response.code === 0) executors.value = response.data || []
	} catch {
		void 0
	}
})
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
    class="detail-pane flex min-h-0 flex-col overflow-hidden"
    :class="isWorkspaceFullscreen ? 'detail-pane--fullscreen' : ''"
  >
    <div
      v-if="!task && !session"
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
          v-if="task?.agentName === 'terminal'"
          name="mdi:console-line"
          :size="14"
          class="shrink-0 text-slate-400"
        />
        <span
          class="truncate text-sm font-semibold text-[var(--n-text-color)]"
          :title="task?.title || session?.title"
        >
          {{ task?.title || session?.title }}
        </span>
        <TaskStatusBadge
          v-if="task"
          :status="task.status"
          class="shrink-0"
        />
        <CodeProjectIdentity
          v-if="projectName"
          class="shrink-0 text-[11px] text-[var(--n-text-color-3)]"
          :project-id="task?.projectId || session?.projectId || 0"
          :name="projectName"
        />
      </header>

      <div
        class="relative min-h-0 flex-1"
        :class="showConversation ? 'bg-white' : 'bg-[#1e1e1e]'"
      >
        <!-- 对话和全屏都属于当前执行任务，浮在内容上避免占用高度。 -->
        <div
          class="detail-pane__floating absolute right-3 top-2 z-[2] flex items-center gap-1"
          :class="{ 'detail-pane__floating--light': showConversation }"
        >
          <n-button
            v-if="structuredTurn"
            quaternary
            circle
            size="tiny"
            :title="t(showNativeCli ? 'code.conversationMode' : 'code.advancedTerminal')"
            @click="showNativeCli = !showNativeCli"
          >
            <template #icon>
              <Icon
                :name="showNativeCli ? 'mdi:message-outline' : 'mdi:console-line'"
                :size="16"
              />
            </template>
          </n-button>
          <n-button
            v-if="task && task.agentName !== 'terminal'"
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

        <CodeConversationPanel
          v-if="showConversation && sessionId"
          :session-id="sessionId"
          :task-id="task?.id || null"
          @task-created="emit('taskCreated', $event)"
        />
        <KeepAlive v-else :max="CODE_TERMINAL_POOL_SIZE">
          <CodeTerminal
            :key="terminalIdentity"
            :task-id="task?.id || null"
            :session-id="sessionId"
            reserve-top-right-actions
            @task-created="emit('taskCreated', $event)"
            @new-session="task && emit('openWorkspace', task)"
          />
        </KeepAlive>
      </div>
    </template>
  </section>
</template>

<style scoped>
.detail-pane {
	background: color-mix(in srgb, var(--n-color) 97%, transparent);
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

.detail-pane__floating--light :deep(.n-button) {
	color: rgb(51 65 85);
	background: rgb(255 255 255 / 86%);
}

.detail-pane__floating--light :deep(.n-button:hover) {
	color: rgb(15 23 42);
	background: #fff;
}
</style>
