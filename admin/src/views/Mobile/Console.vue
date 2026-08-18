<script setup lang="ts">
import {
	decideMobileApproval,
	getMobileNodes,
	getMobileOverview,
	getMobileProjects,
	getMobileSessionState,
	getMobileSessions,
	retryMobileInstruction,
	stopMobileSession,
	type MobileNode,
	type MobileOverview
} from "@/api/modules/mobile"
import type { AIProject, CodeSession, CodeSessionState } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import { mobileAlignmentMessages } from "@/i18n/locales/mobileAlignment"
import { findMobileSessionProject, useMobileSessionDisplay } from "./mobileConsoleSession"
import { useMobileConsoleRoute, type MobileConsoleTab } from "./useMobileConsoleRoute"
import { useMobileNodeMetrics } from "./useMobileNodeMetrics"
import { useMobileLogout } from "./useMobileLogout"
import { useMobileConsoleRefresh } from "./useMobileConsoleRefresh"
import MobileConsoleHeader from "./components/MobileConsoleHeader.vue"
import MobileAttentionPanel from "./components/MobileAttentionPanel.vue"
import MobileConsoleNavigation from "./components/MobileConsoleNavigation.vue"
import MobileResourcePanel from "./components/MobileResourcePanel.vue"
import MobileFileBrowser from "./components/MobileFileBrowser.vue"
import MobileRecentSessions from "./components/MobileRecentSessions.vue"
import MobileSessionCreator from "./components/MobileSessionCreator.vue"
import MobileCodePanel from "./components/MobileCodePanel.vue"
import MobileSystemUpdate from "./components/MobileSystemUpdate.vue"
import MobileSettingsPanel from "./components/MobileSettingsPanel.vue"
import MobileTaskStatusDrawer from "./components/MobileTaskStatusDrawer.vue"
import { useI18n } from "vue-i18n"
import { computed, onMounted, ref } from "vue"
import { useMessage } from "naive-ui"
import MobileNodeSwitcher from "./components/MobileNodeSwitcher.vue"
import MobileNodeOverview from "./components/MobileNodeOverview.vue"
import MobileProjectTerminals from "./components/MobileProjectTerminals.vue"
const { t } = useI18n({ messages: mobileAlignmentMessages })
const message = useMessage()
const overview = ref<MobileOverview | null>(null)
const nodes = ref<MobileNode[]>([])
const showNodeSwitcher = ref(false)
const nodesLoading = ref(false)
const nodesLoadError = ref("")
const projects = ref<AIProject[]>([])
const sessions = ref<CodeSession[]>([])
const sessionsLoading = ref(false)
const sessionState = ref<CodeSessionState | null>(null)
const {
	activeTab,
	selectedNodeId,
	selectedProjectId,
	selectedSessionId,
	selectedTaskId,
	showSessionDetail,
	syncRoute,
	router
} = useMobileConsoleRoute({
	isLocalNode: nodeId => nodeId === 0 || Boolean(nodes.value.find(item => item.id === nodeId)?.isLocal),
	sessions,
	sessionState,
	loadSessions,
	loadSessionState
})
const confirmLogout = useMobileLogout(t, router)
const loading = ref(false)
const actionLoading = ref(false)
const loadError = ref("")
const showSessionCreator = ref(false)
const showFiles = ref(false)
const showTaskStatus = ref(false)
const showProjectTerminals = ref(false)
const projectTerminalSession = ref<HostTerminalSession | null>(null)
const projectTerminalProject = ref<AIProject | null>(null)

async function handleDeliveryUpdated() {
	await Promise.all([loadSessions(), loadSessionState(true)])
}

const { selectedSession, selectedTaskName, selectedProjectName } = useMobileSessionDisplay(
	sessions,
	selectedSessionId,
	projects,
	() => t("mobile.unlinkedProject")
)
const { selectedNode, memoryPercent, cpuPercent, load1, nodeIsOnline, nodeCanOperate } = useMobileNodeMetrics(
	nodes,
	selectedNodeId,
	overview
)
const isProjectTerminal = computed(() => activeTab.value === "code" && Boolean(projectTerminalSession.value))
const isTaskDetail = computed(
	() =>
		activeTab.value === "code" &&
		(Boolean(showSessionDetail.value && selectedSession.value) || isProjectTerminal.value)
)
const startRefresh = useMobileConsoleRefresh({
	activeTab,
	selectedNode,
	selectedSessionId,
	loadOverview,
	loadNodes,
	loadSessionState
})

async function loadNodes(silent = false) {
	if (!silent) nodesLoading.value = true
	try {
		nodes.value = await getMobileNodes()
		if (!nodes.value.some(item => item.id === selectedNodeId.value)) {
			selectedNodeId.value = 0
			localStorage.setItem("gopanel-mobile-node-id", "0")
		}
		nodesLoadError.value = ""
	} catch (error) {
		nodesLoadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
		if (nodesLoadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
	} finally {
		if (!silent) nodesLoading.value = false
	}
}

async function selectNode(node: MobileNode) {
	selectedNodeId.value = node.id
	localStorage.setItem("gopanel-mobile-node-id", String(node.id))
	if (!node.isLocal) {
		sessions.value = []
		selectedSessionId.value = 0
		selectedTaskId.value = 0
		sessionState.value = null
		showSessionDetail.value = false
		projectTerminalSession.value = null
		projectTerminalProject.value = null
	}
	syncRoute(true)
	if (node.isLocal && activeTab.value === "code") await loadSessions(true)
}

async function loadProjects() {
	try {
		const result = await getMobileProjects()
		projects.value = result.items
		if (!projects.value.some(project => project.id === selectedProjectId.value)) {
			selectedProjectId.value = projects.value[0]?.id || null
			if (selectedProjectId.value)
				localStorage.setItem("gopanel-mobile-project-id", String(selectedProjectId.value))
		}
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
	}
}

async function loadOverview(silent = false) {
	if (!silent) loading.value = true
	try {
		overview.value = await getMobileOverview()
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
	} finally {
		if (!silent) loading.value = false
	}
}

async function loadSessions(clearCurrent = false) {
	if (!selectedProjectId.value) {
		sessions.value = []
		return
	}
	sessionsLoading.value = true
	if (clearCurrent) sessions.value = []
	try {
		const result = await getMobileSessions(selectedProjectId.value)
		sessions.value = result.items || []
		if (selectedSessionId.value) await loadSessionState(true)
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
	} finally {
		sessionsLoading.value = false
	}
}

async function loadSessionState(silent = false) {
	if (!selectedSessionId.value) return
	if (!silent) loading.value = true
	try {
		sessionState.value = await getMobileSessionState(selectedSessionId.value)
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
	} finally {
		if (!silent) loading.value = false
	}
}

async function selectSession(session: CodeSession) {
	projectTerminalSession.value = null
	projectTerminalProject.value = null
	selectedSessionId.value = session.id
	selectedTaskId.value = 0
	showSessionDetail.value = true
	await loadSessionState()
	syncRoute(true)
}

async function openSession(session: CodeSession) {
	activeTab.value = "code"
	const projectId = findMobileSessionProject(session, projects.value)?.id || session.projectId
	if (projectId) {
		selectedProjectId.value = projectId
		localStorage.setItem("gopanel-mobile-project-id", String(projectId))
		await loadSessions(true)
	} else {
		sessions.value = [session]
	}
	await selectSession(session)
}

async function openTask(task: CodeTaskListItem) {
	activeTab.value = "code"
	selectedProjectId.value = task.projectId || null
	selectedSessionId.value = task.sessionId
	selectedTaskId.value = task.id
	showSessionDetail.value = true
	if (task.projectId) localStorage.setItem("gopanel-mobile-project-id", String(task.projectId))
	if (task.projectId) await loadSessions(true)
	if (!sessions.value.some(session => session.id === task.sessionId)) {
		try {
			const state = await getMobileSessionState(task.sessionId)
			sessionState.value = state
			sessions.value = [state.session, ...sessions.value]
		} catch (error) {
			message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
			return
		}
	}
	if (!sessionState.value || sessionState.value.session.id !== task.sessionId) await loadSessionState()
	syncRoute(true)
}

async function openAttentionSession(sessionId: number) {
	let session =
		overview.value?.sessions.find(item => item.id === sessionId) ||
		sessions.value.find(item => item.id === sessionId)
	if (!session) {
		try {
			session = (await getMobileSessionState(sessionId)).session
		} catch (error) {
			return
		}
	}
	if (session) await openSession(session)
}

async function selectProject(projectId: number) {
	selectedProjectId.value = projectId
	localStorage.setItem("gopanel-mobile-project-id", String(projectId))
	selectedSessionId.value = 0
	selectedTaskId.value = 0
	showSessionDetail.value = false
	sessionState.value = null
	syncRoute()
	await loadSessions(true)
}

async function switchToOverview() {
	activeTab.value = "overview"
	syncRoute(true)
	await loadOverview()
}

async function switchToCode() {
	activeTab.value = "code"
	showSessionDetail.value = false
	projectTerminalSession.value = null
	projectTerminalProject.value = null
	syncRoute(true)
	if (selectedNode.value?.isLocal) await loadSessions(true)
}

function selectTab(tab: MobileConsoleTab) {
	if (tab === "overview") void switchToOverview()
	else if (tab === "code") void switchToCode()
	else {
		activeTab.value = tab
		syncRoute(true)
	}
}

async function leaveTaskDetail() {
	if (isProjectTerminal.value) {
		projectTerminalSession.value = null
		projectTerminalProject.value = null
		syncRoute()
		return
	}
	showSessionDetail.value = false
	syncRoute()
}

function openProjectTerminal(session: HostTerminalSession, project: AIProject) {
	activeTab.value = "code"
	selectedSessionId.value = 0
	selectedTaskId.value = 0
	showSessionDetail.value = false
	sessionState.value = null
	projectTerminalSession.value = session
	projectTerminalProject.value = project
}

async function handleSessionCreated(session: CodeSession) {
	activeTab.value = "code"
	selectedProjectId.value = session.projectId
	localStorage.setItem("gopanel-mobile-project-id", String(session.projectId))
	selectedSessionId.value = session.id
	selectedTaskId.value = 0
	showSessionDetail.value = true
	await loadSessions(true)
	await loadSessionState()
}

async function decideApproval(approved: boolean, reason = "") {
	const approvalId = sessionState.value?.pendingApproval?.id
	if (!approvalId) return
	actionLoading.value = true
	try {
		await decideMobileApproval(approvalId, approved, reason)
		await loadSessionState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
	} finally {
		actionLoading.value = false
	}
}

async function stopExecution() {
	actionLoading.value = true
	try {
		await stopMobileSession(selectedSessionId.value)
		await loadSessionState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
	} finally {
		actionLoading.value = false
	}
}

async function retryExecution() {
	const instructionId = sessionState.value?.latestInstruction?.id
	if (!instructionId) return
	actionLoading.value = true
	try {
		await retryMobileInstruction(instructionId)
		await loadSessionState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
	} finally {
		actionLoading.value = false
	}
}

onMounted(async () => {
	await Promise.all([loadOverview(), loadNodes(), loadProjects()])
	if (selectedNode.value?.isLocal) await loadSessions()
	syncRoute()
	startRefresh()
})
</script>

<template>
	<div class="min-h-dvh text-slate-900" :class="isTaskDetail ? 'bg-[#0b1020] pb-0' : 'bg-slate-100 pb-24'">
		<MobileConsoleHeader
			v-if="!isTaskDetail"
			:active-tab="activeTab"
			:node-name="selectedNode?.name || ''"
			:node-online="nodeIsOnline"
			:can-create-session="Boolean(selectedNode?.isLocal)"
			@select-node="showNodeSwitcher = true"
			@new-session="showSessionCreator = true"
		/>

		<main :class="isTaskDetail ? 'w-full p-0' : 'mx-auto max-w-2xl p-4'">
			<MobileSystemUpdate v-if="activeTab === 'overview' && selectedNode?.isLocal" />
			<n-alert
				v-if="loadError && (activeTab === 'overview' || activeTab === 'code')"
				type="error"
				class="mb-4"
				:title="t('mobile.loadFailed')"
			>
				{{ loadError }}
			</n-alert>
			<n-alert
				v-if="nodesLoadError && activeTab === 'overview'"
				type="error"
				class="mb-4"
				:title="t('mobile.nodesLoadFailed')"
			>
				<div class="flex items-center justify-between gap-3">
					<span>{{ nodesLoadError }}</span>
					<n-button size="small" text type="primary" @click="loadNodes()">{{ t("mobile.retry") }}</n-button>
				</div>
			</n-alert>

			<n-spin :show="loading">
				<div v-if="activeTab === 'overview'" class="space-y-4">
					<MobileAttentionPanel v-if="selectedNode?.isLocal" @open-session="openAttentionSession" />
					<MobileNodeOverview
						:node="selectedNode"
						:online="nodeIsOnline"
						:cpu-percent="cpuPercent"
						:memory-percent="memoryPercent"
						:load="load1"
					/>
					<MobileRecentSessions
						v-if="selectedNode?.isLocal"
						:sessions="overview?.sessions || []"
						:projects="projects"
						:pending-count="overview?.pendingApprovals.length || 0"
						@open="openSession"
						@show-all="switchToCode"
					/>
				</div>

				<MobileResourcePanel
					v-else-if="activeTab === 'resources'"
					:node-id="selectedNodeId"
					:node-available="nodeIsOnline && nodeCanOperate"
				/>

				<MobileSettingsPanel
					v-else-if="activeTab === 'settings'"
					:node="selectedNode"
					:node-online="nodeIsOnline"
					@select-node="showNodeSwitcher = true"
					@logout="confirmLogout"
				/>

				<MobileCodePanel
					v-else
					:remote-node="!selectedNode?.isLocal"
					:task-detail="isTaskDetail"
					:projects="projects"
					:sessions="sessions"
					:selected-project-id="selectedProjectId"
					:selected-session-id="selectedSessionId"
					:selected-task-id="selectedTaskId"
					:sessions-loading="sessionsLoading"
					:project-terminal-session="projectTerminalSession"
					:project-terminal-project="projectTerminalProject"
					:selected-session="selectedSession"
					:selected-task-name="selectedTaskName"
					:selected-project-name="selectedProjectName"
					@select-project="selectProject"
					@select-session="selectSession"
					@open-task="openTask"
					@new-session="showSessionCreator = true"
					@project-terminal="showProjectTerminals = true"
					@back="leaveTaskDetail"
					@open-files="showFiles = true"
					@open-status="showTaskStatus = true"
					@renamed="loadSessions"
				/>
			</n-spin>
		</main>

		<MobileConsoleNavigation v-if="!isTaskDetail" :active-tab="activeTab" @select="selectTab" />
		<MobileNodeSwitcher
			v-model:show="showNodeSwitcher"
			:nodes="nodes"
			:selected-id="selectedNodeId"
			:loading="nodesLoading"
			@select="selectNode"
		/>
		<MobileProjectTerminals
			v-model:show="showProjectTerminals"
			:projects="projects"
			@opened="openProjectTerminal"
		/>
		<MobileSessionCreator v-model:show="showSessionCreator" @created="handleSessionCreated" />
		<MobileFileBrowser v-if="selectedSessionId" v-model:show="showFiles" :session-id="selectedSessionId" />
		<MobileTaskStatusDrawer
			v-model:show="showTaskStatus"
			:session="selectedSession"
			:state="sessionState"
			:loading="actionLoading"
			@approve="reason => decideApproval(true, reason)"
			@reject="reason => decideApproval(false, reason)"
			@stop="stopExecution"
			@retry="retryExecution"
			@delivery-updated="handleDeliveryUpdated"
		/>
	</div>
</template>
