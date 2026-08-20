<template>
  <div
    class="ai-workspace-root page page-wrapped page-mobile-full page-without-footer relative flex w-full flex-col overflow-hidden rounded-[24px] border border-slate-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_18px_45px_rgba(15,23,42,0.08)]"
    :class="{
      'ai-workspace-root--embedded': embedded,
      'ai-workspace-root--immersive': isWorkspaceFullscreen,
    }"
  >
    <n-layout
      has-sider
      class="h-full flex-1 !bg-transparent"
      style="width: 100%"
    >
      <n-layout-sider
        collapse-mode="width"
        :collapsed-width="0"
        :width="280"
        show-trigger="bar"
        class="ai-workspace-sider !bg-[rgba(248,250,252,0.75)] backdrop-blur-sm"
        style="height: 100%"
      >
        <div class="ai-workspace-sider-inner flex h-full flex-col border-r border-slate-200/80">
          <div class="ai-workspace-sider-header border-b border-slate-200/80 p-3">
            <div
              class="ai-workspace-sider-card  bg-white/90 p-3"
            >
              <div class="flex flex-col gap-3">
                <div
                  class="flex cursor-pointer items-center gap-3 rounded-2xl px-1 py-1 transition-opacity hover:opacity-80"
                  @click="backToLobby"
                >
                  <n-button
                    quaternary
                    circle
                    size="small"
                    class="!bg-slate-100"
                  >
                    <template #icon>
                      <Icon name="mdi:arrow-left" />
                    </template>
                  </n-button>
                  <div class="min-w-0 flex-1">
                    <CodeProjectIdentity
                      class="text-lg font-semibold text-[var(--n-text-color)]"
                      :project-id="currentProjectId"
                      :name="projectInfo?.name || t('code.projectFallback')"
                    />
                  </div>
                  <n-tooltip v-if="hasWorkspaceContext">
                    <template #trigger>
                      <n-button
                        quaternary
                        circle
                        size="small"
                        class="!bg-slate-100 shrink-0"
                        @click.stop="goTaskHome"
                      >
                        <template #icon>
                          <Icon name="mdi:view-dashboard-outline" />
                        </template>
                      </n-button>
                    </template>
                    {{ t("code.taskHome") }}
                  </n-tooltip>
                </div>
              </div>
            </div>
          </div>

          <ProjectTaskSidebar
            :project-id="currentProjectId"
            :tasks="aiTasks"
            :task-total="aiTaskTotal"
            :current-task-id="currentTaskId"
            :task-action-options="taskActionOptions"
            @select-task="selectTask"
            @task-action="handleTaskAction"
            @refresh-tasks="fetchTasks()"
            @create-task="createNewTask"
          />
        </div>
      </n-layout-sider>

      <n-layout-content content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
        <div
          class="ai-workspace-content-panel flex h-full min-h-0 flex-1 flex-col bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.08),transparent_28%)] p-2 md:p-3"
        >
          <CodeWorkspaceToolbar
            :session-label="sessionLabel"
            :session-subtitle="sessionSubtitle"
            :session-id="currentSessionId"
            :has-context="hasWorkspaceContext"
            :is-terminal-session="isTerminalSession"
            :show-conversation="structuredTurn"
            :workspace-mode="workspaceMode"
            :embedded="embedded"
            :fullscreen-enabled="isDesktop"
            :fullscreen-label="fullscreenLabel"
            :is-fullscreen="isWorkspaceFullscreen"
            :project-id="currentProjectId"
            @show-structure="showProjectStructure = true"
            @take-terminal="takeOverTerminal"
            @open-history="showHistoryDrawer = true"
            @toggle-fullscreen="toggleWorkspaceFullscreen"
            @open-file="openFile"
            @update-mode="switchWorkspaceMode"
          />
          <ProjectOverviewPanel
            v-if="currentSessionId === null && currentTaskId === null && !isProjectTerminalActive"
            :project="projectInfo"
            :project-id="currentProjectId"
            :tasks="aiTasks"
            :terminal-available="projectTerminalAvailable"
            @create-task="createNewTask"
            @open-terminal="openProjectTerminal"
            @select-task="selectTask"
          />
          <div
            v-if="currentSessionId !== null && !isProjectTerminalActive"
            v-show="workspaceMode === 'changes'"
            class="ai-workspace-editor-shell min-h-0 flex-1 overflow-hidden rounded  border border-slate-200/80 bg-white shadow-[0_24px_50px_rgba(15,23,42,0.14)]"
          >
            <CodeGitReview
              :session-id="currentSessionId"
              :active="workspaceMode === 'changes'"
              @open-file="openFile"
            />
          </div>
          <div
            v-show="workspaceMode === 'editor' && !isProjectTerminalActive"
            class="ai-workspace-editor-shell flex min-h-0 flex-1 overflow-hidden rounded-[20px] border border-slate-200/80 bg-white shadow-[0_24px_50px_rgba(15,23,42,0.14)]"
          >
            <div class="min-w-0 flex-1">
              <SessionFileEditor
                ref="fileEditorRef"
                :session-id="currentSessionId"
                :path="selectedFile.path"
                :extension="selectedFile.extension"
                @active-path="activeFilePath = $event"
              />
            </div>
            <aside
              v-if="currentSessionId !== null"
              class="hidden h-full w-80 shrink-0 border-l border-slate-200 xl:block"
            >
              <ProjectStructurePanel
                :key="currentSessionId"
                :session-id="currentSessionId"
                :selected-path="activeFilePath"
                @select-file="openFile"
              />
            </aside>
          </div>

          <CodeConversationPanel
            v-if="workspaceMode === 'conversation' && currentSessionId !== null && !isProjectTerminalActive"
            class="min-h-0 flex-1 overflow-hidden"
            :session-id="currentSessionId"
            :task-id="currentTaskId"
            @task-created="handleTaskCreated"
          />
          <CodeWorkspaceTerminalPanel
            :active="workspaceMode === 'terminal' && terminalMounted"
            :is-project-terminal-active="isProjectTerminalActive"
            :project-terminal-session-id="projectTerminalSessionId"
            :project-terminal-work-dir="projectTerminalWorkDir"
            :project-terminal-opening="projectTerminalOpening"
            :task-id="currentTaskId"
            :session-id="currentSessionId"
            :task-title="currentTask?.title || ''"
            :task-work-dir="currentTask?.workDir || ''"
            :session-work-dir="currentSessionWorkDir"
            :terminal-takeover-requested="terminalTakeoverRequested"
            :mount-task-terminal="terminalMounted"
            :terminal-identity="terminalIdentity"
            @open-project-terminal="openProjectTerminal"
            @reopen-project-terminal="openProjectTerminal"
            @close-project-terminal="handleProjectTerminalClosed"
            @take-task-terminal="takeOverTerminal"
            @task-created="handleTaskCreated"
            @new-session="createNewTask"
          />
        </div>
      </n-layout-content>
    </n-layout>
    <n-modal
      v-model:show="showRenameModal"
      preset="dialog"
      :title="t('code.renameTask')"
    >
      <n-input
        v-model:value="editingTaskTitle"
        :placeholder="t('code.taskNamePlaceholder')"
        class="mt-4"
        @keyup.enter="submitRename"
      />
      <template #action>
        <n-button @click="showRenameModal = false">
          {{ t("code.cancel") }}
        </n-button>
        <n-button
          type="primary"
          :loading="renaming"
          @click="submitRename"
        >
          {{ t("code.saveChanges") }}
        </n-button>
      </template>
    </n-modal>

    <NewSessionModal
      v-model:show="showNewSessionModal"
      :project-id="currentProjectId"
      @created="handleSessionCreated"
    />
    <SessionHistoryDrawer
      v-model:show="showHistoryDrawer"
      :session-id="currentSessionId"
      :task-id="currentTaskId"
    />
    <n-drawer
      v-model:show="showProjectStructure"
      placement="right"
      style="width: min(420px, 92vw)"
    >
      <n-drawer-content
        :title="t('code.projectStructure')"
        closable
        body-content-style="padding: 0;"
      >
        <ProjectStructurePanel
          v-if="showProjectStructure && currentSessionId !== null"
          :session-id="currentSessionId"
          :selected-path="activeFilePath"
          @select-file="openFileFromDrawer"
        />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import CodeProjectIdentity from "./components/CodeProjectIdentity.vue"
import CodeConversationPanel from "./components/CodeConversationPanel.vue"
import CodeWorkspaceTerminalPanel from "./components/CodeWorkspaceTerminalPanel.vue"
import NewSessionModal from "./components/NewSessionModal.vue"
import CodeWorkspaceToolbar from "./components/CodeWorkspaceToolbar.vue"
import SessionHistoryDrawer from "./components/SessionHistoryDrawer.vue"
import ProjectStructurePanel from "./components/ProjectStructurePanel.vue"
import SessionFileEditor from "./components/SessionFileEditor.vue"
import ProjectTaskSidebar from "./components/ProjectTaskSidebar.vue"
import ProjectOverviewPanel from "./components/ProjectOverviewPanel.vue"
import CodeGitReview from "./components/CodeGitReview.vue"
import { useCodeWorkspace } from "./useCodeWorkspace"
import { useCodeImmersiveMode } from "./useCodeImmersiveMode"
import { computed, nextTick, watch } from "vue"
import { useRoute, useRouter } from "vue-router"

// App.vue 的 keep-alive 按组件名筛选，改名要同步改那边的 persistentViewNames。
defineOptions({ name: "CodeWorkspaceView" })

const props = withDefaults(defineProps<{ projectId?: number; embedded?: boolean }>(), { embedded: false })
const emit = defineEmits<{ close: [] }>()
const {
	isDesktop,
	isImmersive: isWorkspaceFullscreen,
	toggleImmersive: toggleWorkspaceFullscreen,
} = useCodeImmersiveMode(() => !props.embedded)
const {
	activeFilePath,
	aiTaskTotal,
	aiTasks,
	backToLobby,
	confirmLeaveWorkspace,
	createNewTask,
	currentProjectId,
	currentSessionId,
	currentSessionWorkDir,
	currentTask,
	currentTaskId,
	editingTaskTitle,
	fetchTasks,
	fileEditorRef,
	goTaskHome,
	handleProjectTerminalClosed,
	handleSessionCreated,
	loadRouteSession,
	handleTaskAction,
	handleTaskCreated,
	hasWorkspaceContext,
	isProjectTerminalActive,
	isTerminalSession,
	openFile,
	openFileFromDrawer,
	openProjectTerminal,
	projectInfo,
	projectTerminalAvailable,
	projectTerminalOpening,
	projectTerminalSessionId,
	projectTerminalWorkDir,
	renaming,
	selectedFile,
	sessionLabel,
	sessionSubtitle,
	structuredTurn,
	selectTask,
	taskActionOptions,
	showHistoryDrawer,
	showNewSessionModal,
	showProjectStructure,
	showRenameModal,
	submitRename,
	switchWorkspaceMode,
	t,
	takeOverTerminal,
	terminalIdentity,
	terminalMounted,
	terminalTakeoverRequested,
	workspaceMode,
} = useCodeWorkspace(props, emit)

const fullscreenLabel = computed(() =>
	t(isWorkspaceFullscreen.value ? "code.exitWorkspaceFullscreen" : "code.enterWorkspaceFullscreen")
)

// 工作台会被 KeepAlive：离开后从开发面板再次进入同一项目，组件不会重新挂载，
// 所以任务和会话入口都必须持续监听 query，不能只在首次进入时读取一次。
const route = useRoute()
const router = useRouter()
watch(
	[() => route.query.taskId, aiTasks],
	([taskIdValue, list]) => {
		if (props.embedded) return
		const taskId = Number(taskIdValue) || 0
		if (!taskId || taskId === currentTaskId.value || !list.length) return
		const target = list.find(task => task.id === taskId)
		if (target) selectTask(target)
	},
	{ immediate: true },
)

watch(
	[currentProjectId, () => route.query.sessionId],
	([, sessionIdValue]) => {
		const sessionId = Number(sessionIdValue) || 0
		if (props.embedded || route.query.taskId || !sessionId) return
		if (sessionId === currentSessionId.value && currentTaskId.value === null) return
		void loadRouteSession()
	},
	{ immediate: true },
)

watch(
	[currentProjectId, () => route.query.newTask],
	async ([projectId, newTask]) => {
		if (props.embedded || !projectId || newTask !== "1") return
		await nextTick()
		createNewTask()
		const query = { ...route.query }
		delete query.newTask
		await router.replace({ path: route.path, query })
	},
	{ immediate: true },
)

defineExpose({ confirmClose: confirmLeaveWorkspace })
</script>

<style scoped src="./workspace.css"></style>
