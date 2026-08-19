<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import CodeTerminal from "./CodeTerminal.vue"
import ProjectNativeTerminalPanel from "./ProjectNativeTerminalPanel.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import { CODE_TERMINAL_POOL_SIZE } from "./codeTerminalSession"

const props = defineProps<{
	active: boolean
	isProjectTerminalActive: boolean
	projectTerminalSessionId: number | null
	projectTerminalWorkDir: string
	projectTerminalOpening: boolean
	taskId: number | null
	sessionId: number | null
	taskTitle: string
	taskWorkDir: string
	sessionWorkDir: string
	terminalTakeoverRequested: boolean
	terminalIdentity: string
}>()

const emit = defineEmits<{
	openProjectTerminal: []
	reopenProjectTerminal: []
	closeProjectTerminal: []
	takeTaskTerminal: []
	taskCreated: [taskId: number]
	newSession: []
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })

const canTaskTerminal = computed(() => props.sessionId !== null || props.taskId !== null)
const taskTerminalLabel = computed(() =>
	props.taskId ? t("code.taskTerminal") : props.sessionId ? t("code.sessionTerminal") : t("code.terminalSession"),
)
const activeDirectory = computed(() => {
	if (props.isProjectTerminalActive) return props.projectTerminalWorkDir
	return props.taskWorkDir || props.sessionWorkDir || ""
})

function onProjectTabClick() {
	if (!props.isProjectTerminalActive) emit("openProjectTerminal")
}

function onTaskTabClick() {
	if (props.isProjectTerminalActive) emit("takeTaskTerminal")
}
</script>

<template>
  <div
    v-show="active"
    class="ai-workspace-terminal-panel flex min-h-0 flex-1 flex-col overflow-hidden rounded border border-slate-700 bg-[#1e1e1e] shadow-lg"
  >
    <div class="flex shrink-0 items-center gap-2 border-b border-slate-700 bg-slate-900 px-3 py-1.5">
      <button
        class="flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors"
        :class="
          isProjectTerminalActive
            ? 'bg-blue-600 text-white'
            : 'text-slate-300 hover:bg-slate-800 hover:text-white'
        "
        @click="onProjectTabClick"
      >
        <Icon
          name="mdi:console-line"
          :size="14"
        />
        <span>{{ t("code.projectTerminal") }}</span>
      </button>
      <button
        v-if="canTaskTerminal"
        class="flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors"
        :class="
          !isProjectTerminalActive
            ? 'bg-blue-600 text-white'
            : 'text-slate-300 hover:bg-slate-800 hover:text-white'
        "
        @click="onTaskTabClick"
      >
        <Icon
          name="mdi:robot-outline"
          :size="14"
        />
        <span>{{ taskTerminalLabel }}</span>
      </button>
      <div
        v-if="activeDirectory"
        class="ml-auto min-w-0 truncate text-xs text-slate-400"
        :title="activeDirectory"
      >
        {{ t("code.currentDirectory") }}: {{ activeDirectory }}
      </div>
    </div>

    <!--
			终端实例池：切任务、在「项目终端 / 任务终端」之间来回切，都只是换显示，
			被切走的实例保留 xterm 和滚屏，但释放控制并断开 WebSocket；切回来按 sequence 增量接回。
			只有超出 CODE_TERMINAL_POOL_SIZE 被 LRU 淘汰时才真正销毁 xterm。

			KeepAlive 的插槽里只能有一个节点，注释也算一个，所以说明都写在外面。
			项目终端用固定 key：换会话由内层 HostTerminalPanel 自己的 :key 处理，
			这里再按会话分身份只会让缓存里堆一堆已经死掉的项目终端。
		-->
    <div class="min-h-0 flex-1">
      <KeepAlive :max="CODE_TERMINAL_POOL_SIZE">
        <ProjectNativeTerminalPanel
          v-if="isProjectTerminalActive"
          key="project-terminal"
          :session-id="projectTerminalSessionId"
          :opening="projectTerminalOpening"
          @closed="emit('closeProjectTerminal')"
          @reopen="emit('reopenProjectTerminal')"
        />
        <CodeTerminal
          v-else-if="canTaskTerminal"
          :key="terminalIdentity"
          :task-id="taskId"
          :session-id="sessionId"
          :auto-take-control="terminalTakeoverRequested"
          @task-created="emit('taskCreated', $event)"
          @new-session="emit('newSession')"
        />
      </KeepAlive>
    </div>
  </div>
</template>
