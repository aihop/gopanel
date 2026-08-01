<script setup lang="ts">
import {
	decideMobileApproval,
	getMobileNodes,
	getMobileOverview,
	getMobileProjects,
	getMobileSessionState,
	getMobileSessions,
	logoutMobileDevice,
	retryMobileInstruction,
	stopMobileSession,
	type MobileNode,
	type MobileOverview
} from "@/api/modules/mobile"
import type { AIProject, CodeSession, CodeSessionState } from "@/api/interface/code"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import { mobileMessages } from "@/i18n/locales/mobile"
import MobileConsoleHeader from "./components/MobileConsoleHeader.vue"
import MobileConsoleNavigation from "./components/MobileConsoleNavigation.vue"
import MobileResourcePanel from "./components/MobileResourcePanel.vue"
import MobileFileBrowser from "./components/MobileFileBrowser.vue"
import MobileRecentSessions from "./components/MobileRecentSessions.vue"
import MobileSessionCreator from "./components/MobileSessionCreator.vue"
import MobileSessionBrowser from "./components/MobileSessionBrowser.vue"
import MobileSystemUpdate from "./components/MobileSystemUpdate.vue"
import MobileTaskStatusDrawer from "./components/MobileTaskStatusDrawer.vue"
import MobileTerminal from "./components/MobileTerminal.vue"
import { useI18n } from "vue-i18n"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useRouter } from "vue-router"
import MobileNodeSwitcher from "./components/MobileNodeSwitcher.vue"
import MobileNodeOverview from "./components/MobileNodeOverview.vue"
import MobileProjectTerminals from "./components/MobileProjectTerminals.vue"

const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const activeTab = ref<"overview" | "resources" | "code">("overview")
const overview = ref<MobileOverview | null>(null)
const nodes = ref<MobileNode[]>([])
const selectedNodeId = ref(Number(localStorage.getItem("gopanel-mobile-node-id") || 0))
const showNodeSwitcher = ref(false)
const nodesLoading = ref(false)
const nodesLoadError = ref("")
const projects = ref<AIProject[]>([])
const sessions = ref<CodeSession[]>([])
const selectedProjectId = ref<number | null>(Number(localStorage.getItem("gopanel-mobile-project-id")) || null)
const selectedSessionId = ref(0)
const sessionsLoading = ref(false)
const sessionState = ref<CodeSessionState | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const loadError = ref("")
const showSessionCreator = ref(false)
const showFiles = ref(false)
const showTaskStatus = ref(false)
const showProjectTerminals = ref(false)
const projectTerminalSession = ref<HostTerminalSession | null>(null)
const projectTerminalProject = ref<AIProject | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null
let nodeRefreshTicks = 0

async function handleDeliveryQueued() {
	await Promise.all([loadSessions(), loadSessionState(true)])
}

const selectedSession = computed(() => sessions.value.find(item => item.id === selectedSessionId.value) || null)
const selectedNode = computed(
	() => nodes.value.find(item => item.id === selectedNodeId.value) || nodes.value[0] || null
)
const isProjectTerminal = computed(() => activeTab.value === "code" && Boolean(projectTerminalSession.value))
const isTaskDetail = computed(
	() => activeTab.value === "code" && (Boolean(selectedSession.value) || isProjectTerminal.value)
)
const memoryPercent = computed(() =>
	Math.round(
		selectedNode.value?.isLocal
			? overview.value?.system.memoryUsedPercent || 0
			: selectedNode.value?.summary.memPercent || 0
	)
)
const cpuPercent = computed(() =>
	Math.round(
		selectedNode.value?.isLocal
			? overview.value?.system.cpuUsedPercent || 0
			: selectedNode.value?.summary.cpuPercent || 0
	)
)
const load1 = computed(() =>
	selectedNode.value?.isLocal ? overview.value?.system.load1 || 0 : selectedNode.value?.summary.load1 || 0
)
const nodeIsOnline = computed(() => selectedNode.value?.status === "online")

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

function selectNode(node: MobileNode) {
	selectedNodeId.value = node.id
	localStorage.setItem("gopanel-mobile-node-id", String(node.id))
}

function normalizedProjectPath(value?: string) {
	return (value || "").replace(/\\/g, "/").replace(/\/+$/, "")
}

function sessionProject(session: CodeSession) {
	const projectById = projects.value.find(item => item.id === session.projectId)
	if (projectById) return projectById
	const sessionPaths = [session.sourceWorkDir, session.workDir].map(normalizedProjectPath).filter(Boolean)
	return projects.value.find(project => {
		const projectPaths = [project.workDir, ...(project.sourceDirs || [])].map(normalizedProjectPath).filter(Boolean)
		return projectPaths.some(path => sessionPaths.includes(path))
	})
}

function sessionProjectName(session: CodeSession) {
	return sessionProject(session)?.name || t("mobile.unlinkedProject")
}

function sessionTaskTitle(session: CodeSession) {
	return session.currentTaskTitle || session.title
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
	await loadSessionState()
}

async function openSession(session: CodeSession) {
	activeTab.value = "code"
	const projectId = sessionProject(session)?.id || session.projectId
	if (projectId) {
		selectedProjectId.value = projectId
		localStorage.setItem("gopanel-mobile-project-id", String(projectId))
		await loadSessions(true)
	} else {
		sessions.value = [session]
	}
	await selectSession(session)
}

async function selectProject(projectId: number) {
	selectedProjectId.value = projectId
	localStorage.setItem("gopanel-mobile-project-id", String(projectId))
	selectedSessionId.value = 0
	sessionState.value = null
	await loadSessions(true)
}

async function switchToOverview() {
	activeTab.value = "overview"
	await loadOverview()
}

async function switchToCode() {
	activeTab.value = "code"
	selectedSessionId.value = 0
	sessionState.value = null
	projectTerminalSession.value = null
	projectTerminalProject.value = null
	await loadSessions(true)
}

function selectTab(tab: "overview" | "resources" | "code") {
	if (tab === "overview") void switchToOverview()
	else if (tab === "code") void switchToCode()
	else activeTab.value = "resources"
}

async function leaveTaskDetail() {
	if (isProjectTerminal.value) {
		projectTerminalSession.value = null
		projectTerminalProject.value = null
		return
	}
	activeTab.value = "overview"
	await loadOverview()
}

function openProjectTerminal(session: HostTerminalSession, project: AIProject) {
	activeTab.value = "code"
	selectedSessionId.value = 0
	sessionState.value = null
	projectTerminalSession.value = session
	projectTerminalProject.value = project
}

async function handleSessionCreated(session: CodeSession) {
	activeTab.value = "code"
	selectedProjectId.value = session.projectId
	localStorage.setItem("gopanel-mobile-project-id", String(session.projectId))
	selectedSessionId.value = session.id
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

function confirmLogout() {
	dialog.warning({
		title: t("mobile.logout"),
		content: t("mobile.logoutConfirm"),
		positiveText: t("mobile.logout"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			await logoutMobileDevice()
			await router.replace("/mobile/auth")
		}
	})
}

function startRefresh() {
	refreshTimer = setInterval(() => {
		if (activeTab.value === "overview") {
			if (selectedNode.value?.isLocal) void loadOverview(true)
			nodeRefreshTicks++
			if (nodeRefreshTicks >= 5) {
				nodeRefreshTicks = 0
				void loadNodes(true)
				if (!selectedNode.value?.isLocal) void loadOverview(true)
			}
		} else if (activeTab.value === "code" && selectedSessionId.value) void loadSessionState(true)
	}, 2000)
}

onMounted(async () => {
	await Promise.all([loadOverview(), loadNodes(), loadProjects()])
	await loadSessions()
	startRefresh()
})

onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
	<div class="min-h-dvh text-slate-900" :class="isTaskDetail ? 'bg-[#0b1020] pb-0' : 'bg-slate-100 pb-24'">
		<MobileConsoleHeader
			v-if="!isTaskDetail"
			:active-tab="activeTab"
			:node-name="selectedNode?.name || ''"
			:node-online="nodeIsOnline"
			@select-node="showNodeSwitcher = true"
			@logout="confirmLogout"
			@new-session="showSessionCreator = true"
		/>

		<main :class="isTaskDetail ? 'w-full p-0' : 'mx-auto max-w-2xl p-4'">
			<MobileSystemUpdate v-if="activeTab === 'overview' && selectedNode?.isLocal" />
			<n-alert
				v-if="loadError && activeTab !== 'resources'"
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
					<MobileNodeOverview
						:node="selectedNode"
						:online="nodeIsOnline"
						:cpu-percent="cpuPercent"
						:memory-percent="memoryPercent"
						:load="load1"
					/>
					<MobileRecentSessions
						:sessions="overview?.sessions || []"
						:projects="projects"
						:pending-count="overview?.pendingApprovals.length || 0"
						@open="openSession"
						@show-all="switchToCode"
					/>
				</div>

				<MobileResourcePanel v-else-if="activeTab === 'resources'" />

				<div v-else :class="isTaskDetail ? '' : 'space-y-4'">
					<MobileSessionBrowser
						v-if="!isTaskDetail"
						:projects="projects"
						:sessions="sessions"
						:selected-project-id="selectedProjectId"
						:selected-session-id="selectedSessionId"
						:loading="sessionsLoading"
						@update:selected-project-id="selectProject"
						@new-session="showSessionCreator = true"
						@project-terminal="showProjectTerminals = true"
						@select-session="selectSession"
					/>
					<MobileTerminal
						v-if="projectTerminalSession && projectTerminalProject"
						:session-id="projectTerminalSession.id"
						:task-name="t('mobile.projectTerminal')"
						:project-name="projectTerminalProject.name"
						mode="native"
						@back="leaveTaskDetail"
					/>
					<template v-else-if="selectedSession">
						<MobileTerminal
							:session-id="selectedSessionId"
							:task-name="sessionTaskTitle(selectedSession)"
							:project-name="sessionProjectName(selectedSession)"
							@back="leaveTaskDetail"
							@open-files="showFiles = true"
							@open-status="showTaskStatus = true"
							@renamed="loadSessions"
						/>
					</template>
				</div>
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
			@delivery-queued="handleDeliveryQueued"
		/>
	</div>
</template>
