import { type Ref, computed, nextTick, onMounted, ref, watch } from "vue"
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from "vue-router"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { useAuthStore } from "@/store/auth"
import { useHideLayoutFooter } from "@/composables/useHideLayoutFooter"
import { useCodeTaskPolling } from "./useCodeTaskPolling"
import { useCodeWorkspaceFullscreen } from "./useCodeWorkspaceFullscreen"
import { useProjectTerminal } from "./useProjectTerminal"
import { deleteAITask, getAIProjects, updateAITask } from "@/api/modules/code"
import type { AIProject, AITask, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import { codeWorkspaceMessages } from "./codeWorkspaceMessages"
import { codeTerminalIdentity } from "./components/codeTerminalSession"
import type { CodeWorkspaceMode } from "./components/WorkspaceModeSwitch.vue"

export interface UseCodeWorkspaceProps {
	projectId?: number
	embedded?: boolean
}

export function useCodeWorkspace(props: UseCodeWorkspaceProps, emit: (event: "close") => void) {
	const route = useRoute()
	const router = useRouter()
	const message = useMessage()
	const dialog = useDialog()
	const authStore = useAuthStore()
	const { t } = useI18n({ messages: codeWorkspaceMessages })
	if (!props.embedded) useHideLayoutFooter()
	const currentProjectId = computed(() => props.projectId ?? Number(route.params.id))
	const projectTerminalAvailable = computed(() => authStore.role === "ADMIN" || authStore.role === "SUPER")
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
	const terminalMounted = ref(false)
	const terminalTakeoverRequested = ref(false)
	const isProjectTerminalActive = ref(false)
	const projectTerminalSessionId = ref<number | null>(null)
	const projectTerminalWorkDir = ref("")
	const currentSessionWorkDir = ref("")
	const { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen } = useCodeWorkspaceFullscreen(t)
	const selectedFile = ref({ path: "", extension: "" })
	const activeFilePath = ref("")
	const fileEditorRef = ref<{ hasUnsavedChanges: boolean } | null>(null)
	const editingTaskId = ref<number | null>(null),
		editingTaskTitle = ref("")
	const renaming = ref(false)
	const aiTasks = computed(() => tasks.value.filter(task => task.agentName !== "terminal"))
	const aiTaskTotal = computed(() => Math.max(0, taskTotal.value - (tasks.value.length - aiTasks.value.length)))
	const currentTask = computed(() => tasks.value.find(task => task.id === currentTaskId.value) || null)
	// 以前这里是个自增计数器，切任务就 +1 强制重建终端；
	// 现在换成稳定身份，切回同一个任务命中 KeepAlive 缓存，连接和滚屏都还在。
	const terminalIdentity = computed(() => codeTerminalIdentity(currentSessionId.value, currentTaskId.value))
	const hasWorkspaceContext = computed(
		() => isProjectTerminalActive.value || currentSessionId.value !== null || currentTaskId.value !== null,
	)
	const isTerminalSession = computed(() => isProjectTerminalActive.value)
	const sessionLabel = computed(() =>
		isProjectTerminalActive.value
			? t("code.projectTerminal")
			: currentTask.value?.title || (currentSessionId.value ? t("code.newSession") : t("code.selectTaskToStart")),
	)
	const sessionSubtitle = computed(() =>
		isProjectTerminalActive.value
			? projectTerminalWorkDir.value || t("code.projectTerminalHint")
			: activeFilePath.value || t("code.selectFileToEdit"),
	)
	const taskActionOptions = computed(() => [
		{ label: t("code.renameTask"), key: "rename" },
		{ label: t("code.deleteTask"), key: "delete", style: "color: red;" },
	])

	const fetchProjectInfo = async () => {
		try {
			const response = await getAIProjects({ page: 1, limit: 50 })
			projectInfo.value =
				response.code === 0
					? response.data.items.find(project => project.id === currentProjectId.value) || null
					: null
		} catch (error) {
			void 0
		}
	}

	const { fetchTasks } = useCodeTaskPolling(
		currentProjectId,
		tasks,
		taskTotal,
		error => {
			message.error(error instanceof Error ? error.message : t("code.taskLoadFailed"))
		},
		// 工作台的 3 秒节奏只在真有任务在跑时才需要；全空闲时降到 15 秒。
		{ idleIntervalMs: 15000 },
	)

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
			onPositiveClick: action,
		})
	}

	const goTaskHome = () =>
		confirmDiscardEditorChanges(() => {
			currentTaskId.value = null
			currentSessionId.value = null
			currentSessionWorkDir.value = ""
			isProjectTerminalActive.value = false
			projectTerminalSessionId.value = null
			projectTerminalWorkDir.value = ""
			terminalTakeoverRequested.value = false
			workspaceMode.value = "terminal"
			terminalMounted.value = false
			resetSelectedFile()
		})

	const createNewTask = () =>
		confirmDiscardEditorChanges(() => {
			terminalTakeoverRequested.value = false
			showNewSessionModal.value = true
		})

	const handleSessionCreated = (session: CodeSession) => {
		resetSelectedFile()
		currentTaskId.value = null
		currentSessionId.value = session.id
		currentSessionWorkDir.value = session.workDir
		isProjectTerminalActive.value = false
		terminalTakeoverRequested.value = false
		workspaceMode.value = "terminal"
		terminalMounted.value = true
		void fetchTasks()
	}

	const activateTask = (task: AITask) => {
		resetSelectedFile()
		currentTaskId.value = task.id
		currentSessionId.value = task.sessionId || null
		currentSessionWorkDir.value = task.workDir || currentSessionWorkDir.value
		isProjectTerminalActive.value = false
		terminalTakeoverRequested.value = false
		workspaceMode.value = "terminal"
		terminalMounted.value = true
	}

	const selectTask = (task: AITask) => {
		if (currentTaskId.value === task.id && currentSessionId.value === task.sessionId) return
		confirmDiscardEditorChanges(() => activateTask(task))
	}

	const activateProjectTerminal = (session: HostTerminalSession) => {
		projectTerminalSessionId.value = session.id
		projectTerminalWorkDir.value = session.workDir
		isProjectTerminalActive.value = true
		workspaceMode.value = "terminal"
		terminalMounted.value = true
	}

	const { opening: projectTerminalOpening, open: openNativeProjectTerminal } = useProjectTerminal(
		currentProjectId,
		currentSessionId,
		activateProjectTerminal,
		value => message.success(value),
		value => message.error(value),
		{
			created: t("code.projectTerminalOpened"),
			failed: t("code.projectTerminalOpenFailed"),
			unavailable: t("code.projectTerminalUnavailable"),
		},
	)

	const openProjectTerminal = () => {
		if (projectTerminalSessionId.value !== null) {
			isProjectTerminalActive.value = true
			workspaceMode.value = "terminal"
			terminalMounted.value = true
			return
		}
		void openNativeProjectTerminal()
	}

	const handleProjectTerminalClosed = () => {
		projectTerminalSessionId.value = null
	}

	const takeOverTerminal = () => {
		if (currentSessionId.value === null && currentTaskId.value === null) return
		isProjectTerminalActive.value = false
		terminalTakeoverRequested.value = true
		workspaceMode.value = "terminal"
		terminalMounted.value = true
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
						workspaceMode.value = "terminal"
						terminalMounted.value = false
						resetSelectedFile()
					}
					await fetchTasks()
				} catch (error) {
					void 0
				}
			},
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
			void 0
		} finally {
			renaming.value = false
		}
	}

	const resetWorkspace = () => {
		currentTaskId.value = null
		currentSessionId.value = null
		isProjectTerminalActive.value = false
		projectTerminalSessionId.value = null
		projectTerminalWorkDir.value = ""
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

	return {
		t,
		currentProjectId,
		projectTerminalAvailable,
		projectInfo,
		tasks,
		taskTotal,
		currentTaskId,
		currentSessionId,
		currentSessionWorkDir,
		showNewSessionModal,
		showHistoryDrawer,
		showProjectStructure,
		showRenameModal,
		workspaceMode,
		terminalMounted,
		terminalIdentity,
		terminalTakeoverRequested,
		isProjectTerminalActive,
		projectTerminalSessionId,
		projectTerminalWorkDir,
		isWorkspaceFullscreen,
		fullscreenLabel,
		toggleWorkspaceFullscreen,
		selectedFile,
		activeFilePath,
		fileEditorRef,
		editingTaskId,
		editingTaskTitle,
		renaming,
		aiTasks,
		aiTaskTotal,
		currentTask,
		hasWorkspaceContext,
		isTerminalSession,
		sessionLabel,
		sessionSubtitle,
		taskActionOptions,
		fetchProjectInfo,
		fetchTasks,
		resetSelectedFile,
		openFile,
		openFileFromDrawer,
		confirmDiscardEditorChanges,
		goTaskHome,
		createNewTask,
		handleSessionCreated,
		activateTask,
		selectTask,
		activateProjectTerminal,
		projectTerminalOpening,
		openProjectTerminal,
		handleProjectTerminalClosed,
		takeOverTerminal,
		switchWorkspaceMode,
		handleTaskCreated,
		handleTaskAction,
		submitRename,
		resetWorkspace,
		backToLobby,
		confirmLeaveWorkspace,
	}
}
