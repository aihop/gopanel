<template>
  <div
    class="ai-workspace-root page page-wrapped page-mobile-full page-without-footer relative flex w-full flex-col overflow-hidden rounded-[24px] border border-slate-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_18px_45px_rgba(15,23,42,0.08)]"
    :class="{ 'ai-workspace-root--embedded': embedded }"
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
                    <div class="truncate text-lg font-semibold text-[var(--n-text-color)]">
                      {{ projectInfo?.name || t("code.projectFallback") }}
                    </div>
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
                <div
                  class="grid gap-2"
                  :class="projectTerminalAvailable ? 'grid-cols-2' : 'grid-cols-1'"
                >
                  <n-button
                    v-if="projectTerminalAvailable"
                    type="primary"
                    class="!h-10 !rounded-[14px] shadow-[0_12px_28px_rgba(37,99,235,0.18)]"
                    @click="createNewTask"
                  >
                    <template #icon>
                      <Icon name="mdi:robot-outline" />
                    </template>
                    {{ t("code.aiTaskShort") }}
                  </n-button>
                  <n-button
                    secondary
                    class="!h-10 !rounded-[14px]"
                    :loading="projectTerminalOpening"
                    @click="openProjectTerminal"
                  >
                    <template #icon>
                      <Icon name="mdi:console-line" />
                    </template>
                    {{ t("code.terminalShort") }}
                  </n-button>
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
          />
        </div>
      </n-layout-sider>

      <n-layout-content content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
        <div
          class="ai-workspace-content-panel flex h-full min-h-0 flex-1 flex-col bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.08),transparent_28%)] p-2 md:p-3"
          :class="isWorkspaceFullscreen ? 'fixed inset-0 z-[1000] bg-slate-50' : ''"
        >
          <CodeWorkspaceToolbar
            :session-label="sessionLabel"
            :session-subtitle="sessionSubtitle"
            :session-id="currentSessionId"
            :has-context="hasWorkspaceContext"
            :is-terminal-session="isTerminalSession"
            :workspace-mode="workspaceMode"
            :embedded="embedded"
            :fullscreen-label="fullscreenLabel"
            :is-fullscreen="isWorkspaceFullscreen"
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
            class="ai-workspace-editor-shell min-h-0 flex-1 overflow-hidden rounded-[20px] border border-slate-200/80 bg-white shadow-[0_24px_50px_rgba(15,23,42,0.14)]"
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
import { watch } from "vue"
import { useRoute } from "vue-router"

const props = withDefaults(defineProps<{ projectId?: number; embedded?: boolean }>(), { embedded: false })
const emit = defineEmits<{ close: [] }>()
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
	fullscreenLabel,
	goTaskHome,
	handleProjectTerminalClosed,
	handleSessionCreated,
	handleTaskAction,
	handleTaskCreated,
	hasWorkspaceContext,
	isProjectTerminalActive,
	isTerminalSession,
	isWorkspaceFullscreen,
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
	toggleWorkspaceFullscreen,
	workspaceMode,
} = useCodeWorkspace(props, emit)

// 从开发面板点任务进来时带 ?taskId=，任务列表回来后自动定位过去。
// 只认第一批任务：之后用户在侧栏切走了，不该被地址栏里的旧参数拽回来。
// 快捷面板和主页面共用这个路由，所以内嵌模式不参与。
const route = useRoute()
let pendingTaskId = props.embedded ? 0 : Number(route.query.taskId) || 0
watch(
	aiTasks,
	list => {
		if (!pendingTaskId || !list.length) return
		const target = list.find(task => task.id === pendingTaskId)
		pendingTaskId = 0
		if (target) selectTask(target)
	},
	{ immediate: true },
)

defineExpose({ confirmClose: confirmLeaveWorkspace })
</script>

<style scoped src="./workspace.css"></style>
