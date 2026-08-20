import { type Ref, computed, nextTick, onMounted, ref, watch } from "vue"
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from "vue-router"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { useAuthStore } from "@/store/auth"
import { useHideLayoutFooter } from "@/composables/useHideLayoutFooter"
import { useCodeTaskPolling } from "./useCodeTaskPolling"
import { useProjectTerminal } from "./useProjectTerminal"
import { deleteAITask, getAIProjects, getCodeExecutors, getCodeSession, updateAITask } from "@/api/modules/code"
import type { AIProject, AITask, CodeExecutor, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import { sortCodeTasksStably } from "./codeDashboardBuckets"
import { defaultCodeWorkspaceView, executorSupportsStructuredTurn, findExecutorById } from "./codeStructuredTurn"
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
	const executors = ref<CodeExecutor[]>([])
	const currentExecutorId = ref("")
	const isProjectTerminalActive = ref(false)
	const projectTerminalSessionId = ref<number | null>(null)
	const projectTerminalWorkDir = ref("")
	const currentSessionWorkDir = ref("")
	const selectedFile = ref({ path: "", extension: "" })
	const activeFilePath = ref("")
	const fileEditorRef = ref<{ hasUnsavedChanges: boolean } | null>(null)
	const editingTaskId = ref<number | null>(null),
		editingTaskTitle = ref("")
	const renaming = ref(false)
	const aiTasks = computed(() => sortCodeTasksStably(tasks.value.filter(task => task.agentName !== "terminal")))
	const aiTaskTotal = computed(() => Math.max(0, taskTotal.value - (tasks.value.length - aiTasks.value.length)))
	const currentTask = computed(() => tasks.value.find(task => task.id === currentTaskId.value) || null)
	// 以前这里是个自增计数器，切任务就 +1 强制重建终端；
	// 现在换成稳定身份，切回同一个任务命中 KeepAlive 缓存，连接和滚屏都还在。
	const terminalIdentity = computed(() => codeTerminalIdentity(currentSessionId.value, currentTaskId.value))
	const hasWorkspaceContext = computed(
		() => isProjectTerminalActive.value || currentSessionId.value !== null || currentTaskId.value !== null,
	)
	const isTerminalSession = computed(() => isProjectTerminalActive.value)
	const structuredTurn = computed(() =>
		executorSupportsStructuredTurn(findExecutorById(executors.value, currentExecutorId.value)),
	)

	const ensureExecutors = async () => {
		if (executors.value.length) return
		try {
			const response = await getCodeExecutors()
			if (response.code === 0) executors.value = response.data || []
		} catch {
			void 0
		}
	}

	const applySessionView = (executorId: string, isProjectTerminal = false) => {
		currentExecutorId.value = executorId
		const view = defaultCodeWorkspaceView({
			isProjectTerminal,
			structuredTurn: executorSupportsStructuredTurn(findExecutorById(executors.value, executorId)),
		})
		workspaceMode.value = view.mode
		terminalMounted.value = view.terminalMounted
		if (view.mode !== "terminal") terminalTakeoverRequested.value = false
	}
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

	const { fetchTasks, fetchTasksFast } = useCodeTaskPolling(
		currentProjectId,
		tasks,
		taskTotal,
		error => {
			message.error(error instanceof Error ? error.message : t("code.taskLoadFailed"))
		},
		// 工作台的 3 秒节奏只在真有任务在跑时才需要；全空闲时降到 15 秒。
		{ idleIntervalMs: 15000, selectedTaskId: currentTaskId },
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
			currentExecutorId.value = ""
			workspaceMode.value = "terminal"
			terminalMounted.value = false
			resetSelectedFile()
			syncTaskQuery(null)
		})

	const createNewTask = () =>
		confirmDiscardEditorChanges(() => {
			terminalTakeoverRequested.value = false
			showNewSessionModal.value = true
		})

	const handleSessionCreated = (session: CodeSession) => {
		void activateCreatedSession(session)
	}

	const activateCreatedSession = async (session: CodeSession) => {
		currentTaskId.value = null
		currentSessionId.value = session.id
		await ensureExecutors()
		resetSelectedFile()
		currentSessionWorkDir.value = session.workDir
		isProjectTerminalActive.value = false
		applySessionView(session.agentName)
		syncSessionQuery(session.id)
		void fetchTasks()
	}

	const loadRouteSession = async () => {
		if (props.embedded) return
		const sessionId = Number(route.query.sessionId)
		if (!sessionId) return
		try {
			const response = await getCodeSession(sessionId)
			if (response.data.session.projectId !== currentProjectId.value) {
				throw new Error(t("code.sessionProjectMismatch"))
			}
			handleSessionCreated(response.data.session)
		} catch (error) {
			message.error(error instanceof Error && error.message ? error.message : t("code.sessionLoadFailed"))
		}
	}

	/**
	 * 把当前任务写进地址栏，刷新后还能回到同一条任务。
	 *
	 * 用 replace 不用 push：切任务是浏览同一个工作台，不该在历史里堆一串条目，
	 * 否则「后退」要点很多次才出得去。
	 * 内嵌模式（快捷浮窗）不写：它和主页面共用路由，写了会篡改主页面的地址。
	 */
	const syncWorkspaceQuery = (taskId: number | null, sessionId: number | null) => {
		if (props.embedded) return
		const currentTask = route.query.taskId ? String(route.query.taskId) : ""
		const currentSession = route.query.sessionId ? String(route.query.sessionId) : ""
		const nextTask = taskId ? String(taskId) : ""
		const nextSession = sessionId ? String(sessionId) : ""
		if (currentTask === nextTask && currentSession === nextSession) return
		const query = { ...route.query }
		if (nextTask) query.taskId = nextTask
		else delete query.taskId
		if (nextSession) query.sessionId = nextSession
		else delete query.sessionId
		void router.replace({ path: route.path, query })
	}
	const syncTaskQuery = (taskId: number | null) => syncWorkspaceQuery(taskId, null)
	const syncSessionQuery = (sessionId: number) => syncWorkspaceQuery(null, sessionId)

	const activateTask = async (task: AITask) => {
		currentTaskId.value = task.id
		currentSessionId.value = task.sessionId || null
		await ensureExecutors()
		resetSelectedFile()
		currentSessionWorkDir.value = task.workDir || currentSessionWorkDir.value
		isProjectTerminalActive.value = false
		applySessionView(task.agentName)
		syncTaskQuery(task.id)
	}

	const selectTask = (task: AITask) => {
		if (currentTaskId.value === task.id && currentSessionId.value === task.sessionId) return
		confirmDiscardEditorChanges(() => void activateTask(task))
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
		if (mode === "terminal") {
			terminalMounted.value = true
			return
		}
		if (mode === "conversation") terminalTakeoverRequested.value = false
	}

	const handleTaskCreated = (taskId: number) => {
		currentTaskId.value = taskId
		syncTaskQuery(taskId)
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
					const response = await deleteAITask(task.id)
					if (response.code !== 0) throw new Error(response.message)
					message.success(t("code.taskDeleted"))
					if (currentTaskId.value === task.id) {
						currentTaskId.value = null
						currentSessionId.value = null
						currentExecutorId.value = ""
						workspaceMode.value = "terminal"
						terminalMounted.value = false
						resetSelectedFile()
						syncTaskQuery(null)
					}
					await fetchTasks()
				} catch (error) {
					message.error(error instanceof Error && error.message ? error.message : t("code.taskDeleteFailed"))
				}
			},
		})
	}

	const submitRename = async () => {
		if (!editingTaskTitle.value.trim() || !editingTaskId.value) return
		renaming.value = true
		try {
			const response = await updateAITask(editingTaskId.value, editingTaskTitle.value.trim())
			if (response.code !== 0) throw new Error(response.message)
			message.success(t("code.taskRenamed"))
			showRenameModal.value = false
			await fetchTasks()
		} catch (error) {
			message.error(error instanceof Error && error.message ? error.message : t("code.taskRenameFailed"))
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
		currentExecutorId.value = ""
		workspaceMode.value = "terminal"
		terminalMounted.value = false
		resetSelectedFile()
	}

	const backToLobby = () => (props.embedded ? emit("close") : router.push("/code/index"))
	const confirmLeaveWorkspace = () =>
		!fileEditorRef.value?.hasUnsavedChanges || window.confirm(t("code.switchSessionUnsavedHint"))
	onBeforeRouteLeave(() => props.embedded || confirmLeaveWorkspace())
	onBeforeRouteUpdate((to, from) => {
		// 只是把当前任务写进 query，不算离开工作台，不该再弹一次未保存提示 ——
		// selectTask 已经走过 confirmDiscardEditorChanges 了，这里再弹就是第二次。
		if (to.path === from.path) return true
		return props.embedded || confirmLeaveWorkspace()
	})

	onMounted(() => {
		void ensureExecutors()
		void fetchProjectInfo()
		void fetchTasksFast()
	})
	watch(currentProjectId, newId => {
		if (!newId) return
		resetWorkspace()
		void fetchProjectInfo()
		void fetchTasksFast()
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
		structuredTurn,
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
		loadRouteSession,
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
