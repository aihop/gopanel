<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeExecutor, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { getCodeExecutors } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { defaultDashboardWorkbenchMode, executorSupportsStructuredTurn, findExecutorById } from "../codeStructuredTurn"
import type { CodeWorkspaceMode } from "../codeStructuredTurn"
import { useCodeWorkspaceFullscreen } from "../useCodeWorkspaceFullscreen"
import CodeConversationPanel from "./CodeConversationPanel.vue"
import CodeGitReview from "./CodeGitReview.vue"
import CodeProjectIdentity from "./CodeProjectIdentity.vue"
import CodeTaskDeliveryButton from "./CodeTaskDeliveryButton.vue"
import CodeTerminal from "./CodeTerminal.vue"
import ProjectStructurePanel from "./ProjectStructurePanel.vue"
import SessionFileEditor from "./SessionFileEditor.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"
import WorkspaceModeSwitch from "./WorkspaceModeSwitch.vue"
import { CODE_TERMINAL_POOL_SIZE, codeTerminalIdentity } from "./codeTerminalSession"

const props = defineProps<{
	task: CodeTaskListItem | null
	session?: CodeSession | null
	projectName?: string
}>()
const emit = defineEmits<{
	openHistory: [task: CodeTaskListItem]
	taskCreated: [taskId: number]
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen } = useCodeWorkspaceFullscreen(t)

const terminalIdentity = computed(() =>
	codeTerminalIdentity(props.session?.id || props.task?.sessionId || null, props.task?.id || null),
)
const sessionId = computed(() => props.session?.id || props.task?.sessionId || null)
const executorId = computed(() => props.session?.agentName || props.task?.agentName || "")
const executors = ref<CodeExecutor[]>([])
const workspaceMode = ref<CodeWorkspaceMode>("conversation")
const selectedFile = ref({ path: "", extension: "" })
const activeFilePath = ref("")
const structuredTurn = computed(() =>
	executorSupportsStructuredTurn(findExecutorById(executors.value, executorId.value)),
)
const defaultMode = computed(() =>
	defaultDashboardWorkbenchMode({
		executorId: executorId.value,
		structuredTurn: structuredTurn.value,
		executorsLoaded: executors.value.length > 0,
	}),
)
const executorLabel = computed(() =>
	[props.task?.summary.executor || executorId.value, props.task?.summary.model].filter(Boolean).join(" · "),
)

const applyDefaultMode = () => {
	workspaceMode.value = defaultMode.value
	selectedFile.value = { path: "", extension: "" }
	activeFilePath.value = ""
}

watch(sessionId, applyDefaultMode)
onMounted(async () => {
	try {
		const response = await getCodeExecutors()
		if (response.code === 0) executors.value = response.data || []
	} catch {
		void 0
	}
	applyDefaultMode()
})

const openFile = (file: { path: string; extension: string }) => {
	workspaceMode.value = "editor"
	selectedFile.value = file
}
</script>

<template>
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
      <header class="detail-pane__header flex shrink-0 flex-wrap items-center gap-2 px-3 py-2">
        <div class="flex min-w-0 flex-1 items-center gap-2">
          <CodeProjectIdentity
            v-if="projectName"
            class="shrink-0 text-[11px] text-[var(--n-text-color-3)]"
            :project-id="task?.projectId || session?.projectId || 0"
            :name="projectName"
          />
          <span
            class="min-w-0 truncate text-sm font-medium tracking-[0.01em] text-[var(--n-text-color)]"
            :title="task?.title || session?.title"
          >
            {{ task?.title || session?.title }}
          </span>
          <TaskStatusBadge
            v-if="task"
            :status="task.status"
            class="shrink-0"
          />
          <span
            v-if="executorLabel"
            class="hidden min-w-0 truncate text-[11px] text-[var(--n-text-color-3)] lg:inline"
            :title="executorLabel"
          >
            {{ executorLabel }}
          </span>
        </div>
        <div class="flex min-w-0 flex-wrap items-center gap-2">
          <CodeTaskDeliveryButton
            v-if="sessionId && executorId !== 'terminal'"
            :session-id="sessionId"
            compact
          />
          <WorkspaceModeSwitch
            :value="workspaceMode"
            :show-conversation="structuredTurn || defaultMode === 'conversation'"
            @update:value="workspaceMode = $event"
          />
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
      </header>

      <div
        v-show="workspaceMode === 'conversation'"
        class="min-h-0 flex-1"
      >
        <CodeConversationPanel
          v-if="sessionId"
          :session-id="sessionId"
          :task-id="task?.id || null"
          @task-created="emit('taskCreated', $event)"
        />
      </div>

      <div
        v-show="workspaceMode === 'terminal'"
        class="min-h-0 flex-1 bg-[#1e1e1e]"
      >
        <KeepAlive :max="CODE_TERMINAL_POOL_SIZE">
          <CodeTerminal
            v-if="workspaceMode === 'terminal'"
            :key="terminalIdentity"
            :task-id="task?.id || null"
            :session-id="sessionId"
            auto-take-control
            @task-created="emit('taskCreated', $event)"
          />
        </KeepAlive>
      </div>

      <div
        v-show="workspaceMode === 'editor'"
        class="flex min-h-0 flex-1 overflow-hidden bg-white"
      >
        <div class="min-w-0 flex-1">
          <SessionFileEditor
            v-if="sessionId"
            :session-id="sessionId"
            :path="selectedFile.path"
            :extension="selectedFile.extension"
            @active-path="activeFilePath = $event"
          />
        </div>
        <aside
          v-if="sessionId"
          class="hidden h-full w-72 shrink-0 border-l border-slate-200 lg:block"
        >
          <ProjectStructurePanel
            :key="sessionId"
            :session-id="sessionId"
            :selected-path="activeFilePath"
            @select-file="openFile"
          />
        </aside>
      </div>

      <div
        v-show="workspaceMode === 'changes'"
        class="min-h-0 flex-1 overflow-hidden bg-white"
      >
        <CodeGitReview
          v-if="sessionId"
          :session-id="sessionId"
          :active="workspaceMode === 'changes'"
          @open-file="openFile"
        />
      </div>
    </template>
  </section>
</template>

<style scoped>
.detail-pane {
	background: var(--bg-accent-color, var(--bg-default-color));
}

.detail-pane__header {
	border-bottom: 1px solid color-mix(in srgb, var(--border-color) 70%, transparent);
}

.detail-pane--fullscreen {
	position: fixed;
	z-index: 1000;
	border-radius: 0;
	box-shadow: none;
	inset: 0;
	background: var(--bg-accent-color, var(--bg-default-color));
}
</style>
