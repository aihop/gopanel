<template>
	<div
		class="ai-workspace-root page page-wrapped page-mobile-full page-without-footer relative flex w-full flex-col overflow-hidden rounded-[24px] border border-slate-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_18px_45px_rgba(15,23,42,0.08)]"
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
										<div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
											{{ t("code.workspace") }}
										</div>
										<div class="truncate text-sm font-semibold text-[var(--n-text-color)]">
											{{ groupInfo?.name || t("code.projectFallback") }}
										</div>
									</div>
								</div>
								<n-button
									type="primary"
									block
									class="!h-10 !rounded-[14px] shadow-[0_12px_28px_rgba(37,99,235,0.18)]"
									@click="createNewTask"
								>
									<template #icon><Icon name="mdi:plus" /></template>
									{{ t("code.newSession") }}
								</n-button>
							</div>
						</div>
					</div>

					<div class="ai-workspace-task-history mt-3 flex min-h-0 flex-1 flex-col overflow-hidden">
						<div class="flex items-center justify-between px-4 pb-2 pt-1">
							<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
								{{ t("code.taskHistory") }}
							</div>
							<div class="text-xs text-slate-400">{{ t("code.taskCount", { count: tasks.length }) }}</div>
						</div>
						<n-scrollbar class="ai-workspace-task-scrollbar min-h-0 flex-1">
							<div class="px-2.5 pb-3 pr-3.5">
								<div
									v-if="tasks.length === 0"
									class="ai-workspace-task-empty flex min-h-[180px] items-center justify-center"
								>
									<n-empty :description="t('code.noProjectHistory')" />
								</div>
								<div v-else class="space-y-1">
									<div
										v-for="task in tasks"
										:key="task.id"
										class="ai-workspace-task-row group/task relative flex cursor-pointer items-start justify-between gap-3 rounded-xl px-3 py-2.5 transition-colors duration-200 hover:bg-slate-200/45"
										:class="
											currentTaskId === task.id
												? 'ai-workspace-task-row--active bg-blue-50/80'
												: ''
										"
										@click="selectTask(task)"
									>
										<div class="min-w-0 flex-1">
											<div
												class="truncate text-sm font-semibold text-slate-800"
												:title="task.title"
											>
												{{ task.title }}
											</div>
											<div class="mt-1.5 flex items-center gap-2">
												<TaskStatusBadge :status="task.status" />
												<span class="text-xs text-slate-400">
													{{ new Date(task.createdAt).toLocaleDateString() }}
												</span>
											</div>
										</div>
										<div
											class="opacity-100 transition-opacity md:opacity-0 md:group-hover/task:opacity-100"
											@click.stop
										>
											<n-dropdown
												trigger="click"
												:options="taskActionOptions"
												@select="key => handleTaskAction(key, task)"
											>
												<n-button
													quaternary
													circle
													size="small"
													class="ai-workspace-task-btn !bg-transparent"
												>
													<template #icon><Icon name="mdi:dots-horizontal" /></template>
												</n-button>
											</n-dropdown>
										</div>
									</div>
								</div>
							</div>
						</n-scrollbar>
					</div>
				</div>
			</n-layout-sider>

			<n-layout-content content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
				<div
					class="ai-workspace-content-panel flex h-full min-h-0 flex-1 flex-col bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.08),transparent_28%)] p-2 md:p-3"
					:class="isWorkspaceFullscreen ? 'fixed inset-0 z-[1000] bg-slate-50' : ''"
				>
					<div
						class="ai-workspace-session-bar mb-2 flex shrink-0 items-center justify-between gap-3 px-2 py-1"
					>
						<div class="min-w-0">
							<div class="truncate text-sm font-semibold text-slate-800">{{ sessionLabel }}</div>
							<div class="truncate text-xs text-slate-400">
								{{ activeFilePath || t("code.selectFileToEdit") }}
							</div>
						</div>
						<div class="flex flex-wrap items-center justify-end gap-2">
							<n-button
								v-if="currentSessionId !== null && workspaceMode === 'editor'"
								size="small"
								quaternary
								class="xl:hidden"
								@click="showProjectStructure = true"
							>
								{{ t("code.projectStructure") }}
							</n-button>
							<CodeApprovalCenter :session-id="currentSessionId" @take-terminal="takeOverTerminal" />
							<SessionApprovalPolicy v-if="currentSessionId !== null" :session-id="currentSessionId" />
							<WorkspaceModeSwitch
								v-if="currentSessionId !== null || currentTaskId !== null"
								:value="workspaceMode"
								@update:value="switchWorkspaceMode"
							/>
							<n-button
								v-if="currentSessionId !== null || currentTaskId !== null"
								size="small"
								secondary
								@click="showHistoryDrawer = true"
							>
								{{ t("code.conversationHistory") }}
							</n-button>
							<n-button
								circle
								secondary
								:aria-label="fullscreenLabel"
								:title="fullscreenLabel"
								@click="toggleWorkspaceFullscreen"
							>
								<template #icon>
									<Icon
										:name="
											isWorkspaceFullscreen
												? 'fluent:full-screen-minimize-24-regular'
												: 'fluent:full-screen-maximize-24-regular'
										"
									/>
								</template>
							</n-button>
						</div>
					</div>
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
						class="ai-workspace-terminal-panel min-h-0 flex-1 overflow-hidden rounded-[20px] border border-slate-700 bg-[#1e1e1e] shadow-lg"
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
			:project-id="currentGroupId"
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
import SessionApprovalPolicy from "./components/SessionApprovalPolicy.vue"
import CodeApprovalCenter from "./components/CodeApprovalCenter.vue"
import SessionHistoryDrawer from "./components/SessionHistoryDrawer.vue"
import ProjectStructurePanel from "./components/ProjectStructurePanel.vue"
import SessionFileEditor from "./components/SessionFileEditor.vue"
import TaskStatusBadge from "./components/TaskStatusBadge.vue"
import CodeGitReview from "./components/CodeGitReview.vue"
import WorkspaceModeSwitch, { type CodeWorkspaceMode } from "./components/WorkspaceModeSwitch.vue"
import { useCodeTaskPolling } from "./useCodeTaskPolling"
import { useCodeWorkspaceFullscreen } from "./useCodeWorkspaceFullscreen"
import { deleteAITask, getAIGroups, updateAITask } from "@/api/modules/code"
import type { AIGroup, AITask, CodeSession } from "@/api/interface/code"
import { codeWorkspaceMessages } from "./codeWorkspaceMessages"

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n({ messages: codeWorkspaceMessages })
useHideLayoutFooter()
const currentGroupId = computed(() => Number(route.params.id))
const groupInfo = ref<AIGroup | null>(null)
const tasks = ref<AITask[]>([])
const currentTaskId = ref<number | null>(null)
const currentSessionId = ref<number | null>(null)
const showNewSessionModal = ref(false)
const showHistoryDrawer = ref(false)
const showProjectStructure = ref(false)
const showRenameModal = ref(false)
const workspaceMode = ref<CodeWorkspaceMode>("terminal")
const terminalMounted = ref(false)
const terminalKey = ref(0)
const terminalTakeoverRequested = ref(false)
const { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen } = useCodeWorkspaceFullscreen(t)
const selectedFile = ref({ path: "", extension: "" })
const activeFilePath = ref("")
const fileEditorRef = ref<{ hasUnsavedChanges: boolean } | null>(null)
const editingTaskId = ref<number | null>(null)
const editingTaskTitle = ref("")
const renaming = ref(false)

const currentTask = computed(() => tasks.value.find(task => task.id === currentTaskId.value) || null)
const sessionLabel = computed(
	() => currentTask.value?.title || (currentSessionId.value ? t("code.newSession") : t("code.selectTaskToStart"))
)
const taskActionOptions = computed(() => [
	{ label: t("code.renameTask"), key: "rename" },
	{ label: t("code.deleteTask"), key: "delete", style: "color: red;" }
])

const fetchGroupInfo = async () => {
	try {
		const response = await getAIGroups({ page: 1, limit: 50 })
		groupInfo.value =
			response.code === 0 ? response.data.items.find(group => group.id === currentGroupId.value) || null : null
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.projectLoadFailed"))
	}
}

const { fetchTasks } = useCodeTaskPolling(currentGroupId, tasks, error => {
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
	terminalTakeoverRequested.value = false
	workspaceMode.value = "terminal"
	terminalMounted.value = true
	terminalKey.value++
	void fetchTasks()
}

const selectTask = (task: AITask) => {
	if (currentTaskId.value === task.id && currentSessionId.value === task.sessionId) return
	confirmDiscardEditorChanges(() => {
		resetSelectedFile()
		currentTaskId.value = task.id
		currentSessionId.value = task.sessionId || null
		terminalTakeoverRequested.value = false
		workspaceMode.value = "terminal"
		terminalMounted.value = true
		terminalKey.value++
	})
}

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

const handleTaskAction = (key: string, task: AITask) => {
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
	showNewSessionModal.value = false
	showHistoryDrawer.value = false
	showProjectStructure.value = false
	workspaceMode.value = "terminal"
	terminalMounted.value = false
	isWorkspaceFullscreen.value = false
	resetSelectedFile()
}

const backToLobby = () => router.push("/code/index")
const confirmLeaveWorkspace = () =>
	!fileEditorRef.value?.hasUnsavedChanges || window.confirm(t("code.switchSessionUnsavedHint"))
onBeforeRouteLeave(confirmLeaveWorkspace)
onBeforeRouteUpdate(confirmLeaveWorkspace)

onMounted(() => {
	void fetchGroupInfo()
	void fetchTasks()
})
watch(
	() => route.params.id,
	newId => {
		if (!newId || route.name !== "Code-Group") return
		resetWorkspace()
		void fetchGroupInfo()
		void fetchTasks()
	}
)
</script>

<style scoped src="./workspace.css"></style>
