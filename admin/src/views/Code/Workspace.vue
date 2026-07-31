<template>
	<div
		class="ai-workspace-root page page-wrapped page-mobile-full page-without-footer relative flex w-full flex-col overflow-hidden rounded-[24px] border border-slate-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_18px_45px_rgba(15,23,42,0.08)]"
		:class="{ 'ai-workspace-root--embedded': embedded }"
	>
		<n-layout has-sider class="h-full flex-1 !bg-transparent" style="width: 100%">
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
							class="ai-workspace-sider-card rounded-[18px] border border-slate-200/80 bg-white/90 p-3 shadow-sm"
						>
							<div class="flex flex-col gap-3">
								<div
									class="flex cursor-pointer items-center gap-3 rounded-2xl px-1 py-1 transition-opacity hover:opacity-80"
									@click="backToLobby"
								>
									<n-button quaternary circle size="small" class="!bg-slate-100">
										<template #icon><Icon name="mdi:arrow-left" /></template>
									</n-button>
									<div class="min-w-0">
										<div class="truncate text-lg font-semibold text-[var(--n-text-color)]">
											{{ projectInfo?.name || t("code.projectFallback") }}
										</div>
									</div>
								</div>
								<div class="grid grid-cols-2 gap-2">
									<n-button
										type="primary"
										class="!h-10 !rounded-[14px] shadow-[0_12px_28px_rgba(37,99,235,0.18)]"
										@click="createNewTask"
									>
										<template #icon><Icon name="mdi:robot-outline" /></template>
										{{ t("code.aiTaskShort") }}
									</n-button>
									<n-button
										secondary
										class="!h-10 !rounded-[14px]"
										:loading="projectTerminalOpening"
										@click="openProjectTerminal"
									>
										<template #icon><Icon name="mdi:console-line" /></template>
										{{ t("code.terminalShort") }}
									</n-button>
								</div>
							</div>
						</div>
					</div>

					<ProjectTaskSidebar
						:project-id="currentProjectId"
						:tasks="tasks"
						:task-total="taskTotal"
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
						:terminal-opening="projectTerminalOpening"
						:workspace-mode="workspaceMode"
						:embedded="embedded"
						:fullscreen-label="fullscreenLabel"
						:is-fullscreen="isWorkspaceFullscreen"
						@open-terminal="openProjectTerminal"
						@show-structure="showProjectStructure = true"
						@take-terminal="takeOverTerminal"
						@open-history="showHistoryDrawer = true"
						@toggle-fullscreen="toggleWorkspaceFullscreen"
						@open-file="openFile"
						@update-mode="switchWorkspaceMode"
					/>
					<ProjectOverviewPanel
						v-if="currentSessionId === null && currentTaskId === null"
						:project="projectInfo"
						:project-id="currentProjectId"
						:tasks="tasks"
						@create-task="createNewTask"
						@open-terminal="openProjectTerminal"
						@select-task="selectTask"
					/>
					<div
						v-if="currentSessionId !== null"
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
						v-show="workspaceMode === 'editor'"
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

					<div
						v-if="terminalMounted && (currentTaskId !== null || currentSessionId !== null)"
						v-show="workspaceMode === 'terminal'"
						class="ai-workspace-terminal-panel min-h-0 flex-1 overflow-hidden rounded border border-slate-700 bg-[#1e1e1e] shadow-lg"
					>
						<CodeTerminal
							:key="terminalKey"
							:task-id="currentTaskId"
							:session-id="currentSessionId"
							:auto-take-control="terminalTakeoverRequested"
							@task-created="handleTaskCreated"
						/>
					</div>
				</div>
			</n-layout-content>
		</n-layout>
		<n-modal v-model:show="showRenameModal" preset="dialog" :title="t('code.renameTask')">
			<n-input
				v-model:value="editingTaskTitle"
				:placeholder="t('code.taskNamePlaceholder')"
				class="mt-4"
				@keyup.enter="submitRename"
			/>
			<template #action>
				<n-button @click="showRenameModal = false">{{ t("code.cancel") }}</n-button>
				<n-button type="primary" :loading="renaming" @click="submitRename">
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
		<n-drawer v-model:show="showProjectStructure" placement="right" style="width: min(420px, 92vw)">
			<n-drawer-content :title="t('code.projectStructure')" closable body-content-style="padding: 0;">
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
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from "vue-router"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { useHideLayoutFooter } from "@/composables/useHideLayoutFooter"
import Icon from "@/components/common/Icon.vue"
import CodeTerminal from "./components/CodeTerminal.vue"
import NewSessionModal from "./components/NewSessionModal.vue"
import CodeWorkspaceToolbar from "./components/CodeWorkspaceToolbar.vue"
import SessionHistoryDrawer from "./components/SessionHistoryDrawer.vue"
import ProjectStructurePanel from "./components/ProjectStructurePanel.vue"
import SessionFileEditor from "./components/SessionFileEditor.vue"
import ProjectTaskSidebar from "./components/ProjectTaskSidebar.vue"
import ProjectOverviewPanel from "./components/ProjectOverviewPanel.vue"
import CodeGitReview from "./components/CodeGitReview.vue"
import type { CodeWorkspaceMode } from "./components/WorkspaceModeSwitch.vue"
import { useCodeTaskPolling } from "./useCodeTaskPolling"
import { useCodeWorkspaceFullscreen } from "./useCodeWorkspaceFullscreen"
import { useProjectTerminal } from "./useProjectTerminal"
import { deleteAITask, getAIProjects, updateAITask } from "@/api/modules/code"
import type { AIProject, AITask, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { codeWorkspaceMessages } from "./codeWorkspaceMessages"

const props = withDefaults(defineProps<{ projectId?: number; embedded?: boolean }>(), { embedded: false })
const emit = defineEmits<{ close: [] }>()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n({ messages: codeWorkspaceMessages })
if (!props.embedded) useHideLayoutFooter()
const currentProjectId = computed(() => props.projectId ?? Number(route.params.id))
const projectInfo = ref<AIProject | null>(null),
	tasks = ref<CodeTaskListItem[]>([])
const taskTotal = ref(0)
const currentTaskId = ref<number | null>(null),
	currentSessionId = ref<number | null>(null)
const showNewSessionModal = ref(false),
	showHistoryDrawer = ref(false)
const showProjectStructure = ref(false),
	showRenameModal = ref(false)
const workspaceMode = ref<CodeWorkspaceMode>("terminal")
const terminalMounted = ref(false),
	terminalKey = ref(0)
const terminalTakeoverRequested = ref(false)
const currentSessionExecutor = ref("")
const { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen } = useCodeWorkspaceFullscreen(t)
const selectedFile = ref({ path: "", extension: "" })
const activeFilePath = ref("")
const fileEditorRef = ref<{ hasUnsavedChanges: boolean } | null>(null)
const editingTaskId = ref<number | null>(null),
	editingTaskTitle = ref("")
const renaming = ref(false)
const currentTask = computed(() => tasks.value.find(task => task.id === currentTaskId.value) || null)
const hasWorkspaceContext = computed(() => currentSessionId.value !== null || currentTaskId.value !== null)
const sessionLabel = computed(
	() =>
		currentTask.value?.title ||
		(isTerminalSession.value
			? t("code.projectTerminal")
			: currentSessionId.value
				? t("code.newSession")
				: t("code.selectTaskToStart"))
)
const sessionSubtitle = computed(
	() =>
		activeFilePath.value ||
		(isTerminalSession.value ? t("code.projectTerminalHint") : t("code.selectFileToEdit"))
)
const taskActionOptions = computed(() => [
	{ label: t("code.renameTask"), key: "rename" },
	{ label: t("code.deleteTask"), key: "delete", style: "color: red;" }
])

const fetchProjectInfo = async () => {
	try {
		const response = await getAIProjects({ page: 1, limit: 50 })
		projectInfo.value =
			response.code === 0 ? response.data.items.find(project => project.id === currentProjectId.value) || null : null
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.projectLoadFailed"))
	}
}

const { fetchTasks } = useCodeTaskPolling(currentProjectId, tasks, taskTotal, error => {
	message.error(error instanceof Error ? error.message : t("code.taskLoadFailed"))
})

const resetSelectedFile = () => {
	selectedFile.value = { path: "", extension: "" }
	activeFilePath.value = ""
}

const openFile = async (file: { path: string; extension: string }) => {
	workspaceMode.value = "editor"
	if (selectedFile.value.path === file.path) {
		selectedFile.value = { path: "", extension: "" }
		await nextTick()
	}
	selectedFile.value = file
}

const openFileFromDrawer = async (file: { path: string; extension: string }) => {
	await openFile(file)
	showProjectStructure.value = false
}

const confirmDiscardEditorChanges = (action: () => void) => {
	if (!fileEditorRef.value?.hasUnsavedChanges) {
		action()
		return
	}
	dialog.warning({
		title: t("code.unsavedChanges"),
		content: t("code.switchSessionUnsavedHint"),
		positiveText: t("code.discardAndContinue"),
		negativeText: t("code.cancel"),
		onPositiveClick: action
	})
}

const createNewTask = () =>
	confirmDiscardEditorChanges(() => {
		terminalTakeoverRequested.value = false
		showNewSessionModal.value = true
	})

const handleSessionCreated = (session: CodeSession) => {
	resetSelectedFile()
	currentTaskId.value = null
	currentSessionId.value = session.id
	currentSessionExecutor.value = session.agentName
	terminalTakeoverRequested.value = false
	workspaceMode.value = "terminal"
	terminalMounted.value = true
	terminalKey.value++
	void fetchTasks()
}

const activateTask = (task: AITask) => {
	resetSelectedFile()
	currentTaskId.value = task.id
	currentSessionId.value = task.sessionId || null
	currentSessionExecutor.value = task.agentName || ""
	terminalTakeoverRequested.value = false
	workspaceMode.value = "terminal"
	terminalMounted.value = true
	terminalKey.value++
}

const selectTask = (task: AITask) => {
	if (currentTaskId.value === task.id && currentSessionId.value === task.sessionId) return
	confirmDiscardEditorChanges(() => activateTask(task))
}

const {
	opening: projectTerminalOpening,
	isTerminalSession,
	open: activateProjectTerminal
} = useProjectTerminal(
	currentProjectId,
	tasks,
	currentSessionExecutor,
	activateTask,
	handleSessionCreated,
	value => message.success(value),
	value => message.error(value),
	{
		title: t("code.projectTerminal"),
		created: t("code.projectTerminalOpened"),
		failed: t("code.projectTerminalOpenFailed")
	}
)

const openProjectTerminal = () => confirmDiscardEditorChanges(() => void activateProjectTerminal())

const takeOverTerminal = () => {
	if (currentSessionId.value === null) return
	terminalTakeoverRequested.value = true
	workspaceMode.value = "terminal"
	terminalMounted.value = true
	terminalKey.value++
}

const switchWorkspaceMode = (mode: CodeWorkspaceMode) => {
	workspaceMode.value = mode
	if (mode === "terminal") terminalMounted.value = true
}

const handleTaskCreated = (taskId: number) => {
	currentTaskId.value = taskId
	void fetchTasks()
}

const handleTaskAction = (key: string, task: CodeTaskListItem) => {
	if (key === "rename") {
		editingTaskId.value = task.id
		editingTaskTitle.value = task.title
		showRenameModal.value = true
		return
	}
	if (key !== "delete") return
	dialog.warning({
		title: t("code.deleteTask"),
		content: t("code.deleteTaskConfirm", { name: task.title }),
		positiveText: t("code.confirmDelete"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			try {
				await deleteAITask(task.id)
				message.success(t("code.taskDeleted"))
				if (currentTaskId.value === task.id) {
					currentTaskId.value = null
					currentSessionId.value = null
					currentSessionExecutor.value = ""
					workspaceMode.value = "terminal"
					terminalMounted.value = false
					resetSelectedFile()
				}
				await fetchTasks()
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("code.taskDeleteFailed"))
			}
		}
	})
}

const submitRename = async () => {
	if (!editingTaskTitle.value.trim() || !editingTaskId.value) return
	renaming.value = true
	try {
		await updateAITask(editingTaskId.value, editingTaskTitle.value.trim())
		message.success(t("code.taskRenamed"))
		showRenameModal.value = false
		await fetchTasks()
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.taskRenameFailed"))
	} finally {
		renaming.value = false
	}
}

const resetWorkspace = () => {
	currentTaskId.value = null
	currentSessionId.value = null
	currentSessionExecutor.value = ""
	tasks.value = []
	taskTotal.value = 0
	showNewSessionModal.value = false
	showHistoryDrawer.value = false
	showProjectStructure.value = false
	workspaceMode.value = "terminal"
	terminalMounted.value = false
	isWorkspaceFullscreen.value = false
	resetSelectedFile()
}

const backToLobby = () => (props.embedded ? emit("close") : router.push("/code/index"))
const confirmLeaveWorkspace = () =>
	!fileEditorRef.value?.hasUnsavedChanges || window.confirm(t("code.switchSessionUnsavedHint"))
defineExpose({ confirmClose: confirmLeaveWorkspace })
onBeforeRouteLeave(() => props.embedded || confirmLeaveWorkspace())
onBeforeRouteUpdate(() => props.embedded || confirmLeaveWorkspace())

onMounted(() => {
	void fetchProjectInfo()
	void fetchTasks()
})
watch(currentProjectId, newId => {
	if (!newId) return
	resetWorkspace()
	void fetchProjectInfo()
	void fetchTasks()
})
</script>

<style scoped src="./workspace.css"></style>
