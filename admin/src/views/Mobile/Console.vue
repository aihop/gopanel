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
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import Logo from "@/layouts/common/Logo.vue"
import MobileContainerPanel from "./components/MobileContainerPanel.vue"
import MobileFileBrowser from "./components/MobileFileBrowser.vue"
import MobileSessionCreator from "./components/MobileSessionCreator.vue"
import MobileSystemUpdate from "./components/MobileSystemUpdate.vue"
import MobileTaskStatusDrawer from "./components/MobileTaskStatusDrawer.vue"
import MobileTerminal from "./components/MobileTerminal.vue"
import { useI18n } from "vue-i18n"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useRouter } from "vue-router"
import MobileNodeSwitcher from "./components/MobileNodeSwitcher.vue"
import MobileProjectTerminals from "./components/MobileProjectTerminals.vue"

const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const activeTab = ref<"overview" | "containers" | "code">("overview")
const overview = ref<MobileOverview | null>(null)
const nodes = ref<MobileNode[]>([])
const selectedNodeId = ref(Number(localStorage.getItem("gopanel-mobile-node-id") || 0))
const showNodeSwitcher = ref(false)
const nodesLoading = ref(false)
const nodesLoadError = ref("")
const projects = ref<AIProject[]>([])
const sessions = ref<CodeSession[]>([])
const selectedSessionId = ref(0)
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

const selectedSession = computed(() => sessions.value.find(item => item.id === selectedSessionId.value) || null)
const selectedNode = computed(() => nodes.value.find(item => item.id === selectedNodeId.value) || nodes.value[0] || null)
const isProjectTerminal = computed(() => activeTab.value === "code" && Boolean(projectTerminalSession.value))
const isTaskDetail = computed(
	() => activeTab.value === "code" && (Boolean(selectedSession.value) || isProjectTerminal.value)
)
const memoryPercent = computed(() => Math.round(selectedNode.value?.isLocal ? overview.value?.system.memoryUsedPercent || 0 : selectedNode.value?.summary.memPercent || 0))
const cpuPercent = computed(() => Math.round(selectedNode.value?.isLocal ? overview.value?.system.cpuUsedPercent || 0 : selectedNode.value?.summary.cpuPercent || 0))
const load1 = computed(() => selectedNode.value?.isLocal ? overview.value?.system.load1 || 0 : selectedNode.value?.summary.load1 || 0)
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

function sessionStageLabel(session: CodeSession) {
	const knownStages = ["idle", "interactive", "task_ready", "instruction_queued", "awaiting_approval", "executing", "completed", "preview_ready", "failed", "cancelled", "approval_rejected"]
	const stage = knownStages.includes(session.currentStage) ? session.currentStage : "unknown"
	return t(`mobile.stage_${stage}`)
}

function sessionStageType(session: CodeSession) {
	if (session.currentStage === "failed") return "error"
	if (["awaiting_approval", "approval_rejected", "cancelled"].includes(session.currentStage)) return "warning"
	if (["completed", "preview_ready"].includes(session.currentStage)) return "success"
	if (["executing", "instruction_queued", "interactive"].includes(session.currentStage)) return "info"
	return "default"
}

function sessionApprovalLabel(session: CodeSession) {
	const labels = {
		manual: "mobile.approvalManual",
		safe_auto: "mobile.approvalSafe",
		full_auto: "mobile.approvalFull"
	} as const
	return t(labels[session.approvalPolicy])
}

function formatSessionTime(value: string) {
	return new Date(value).toLocaleString(undefined, {
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit"
	})
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
	return session.title
}

async function loadProjects() {
	try {
		const result = await getMobileProjects()
		projects.value = result.items
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

async function loadSessions() {
	try {
		const result = await getMobileSessions()
		sessions.value = result.items || []
		if (selectedSessionId.value) await loadSessionState(true)
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
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
	await selectSession(session)
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
	await loadSessions()
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
	selectedSessionId.value = session.id
	await loadSessions()
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
	await Promise.all([loadOverview(), loadNodes(), loadProjects(), loadSessions()])
	startRefresh()
})

onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
	<div class="min-h-dvh text-slate-900" :class="isTaskDetail ? 'bg-[#0b1020] pb-0' : 'bg-slate-100 pb-24'">
		<header v-if="!isTaskDetail" class="sticky top-0 z-20 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur">
			<div class="mx-auto flex max-w-2xl items-center justify-between">
				<div class="flex min-w-0 items-center gap-3">
					<Logo :dark="false" class="shrink-0" />
					<button v-if="activeTab === 'overview'" type="button" class="mt-0.5 flex max-w-[65vw] items-center gap-1 border-0 bg-transparent p-0 text-xs text-slate-500" @click="showNodeSwitcher = true">
						<span class="h-2 w-2 shrink-0 rounded-full" :class="nodeIsOnline ? 'bg-emerald-500' : 'bg-slate-400'"></span>
						<span class="truncate">{{ selectedNode?.name || t("mobile.selectNode") }}</span>
						<span>⌄</span>
					</button>
				</div>
				<div class="flex shrink-0 items-center gap-1">
					<n-button size="small" quaternary circle :title="t('mobile.logout')" :aria-label="t('mobile.logout')" @click="confirmLogout">
						<template #icon><Icon name="mdi:logout" /></template>
					</n-button>
					<n-button v-if="activeTab !== 'containers'" size="small" type="primary" secondary @click="showSessionCreator = true">
						{{ t("mobile.newSession") }}
					</n-button>
				</div>
			</div>
		</header>

		<main :class="isTaskDetail ? 'w-full p-0' : 'mx-auto max-w-2xl p-4'">
			<MobileSystemUpdate v-if="activeTab === 'overview' && selectedNode?.isLocal" />
			<n-alert v-if="loadError && activeTab !== 'containers'" type="error" class="mb-4" :title="t('mobile.loadFailed')">
				{{ loadError }}
			</n-alert>
			<n-alert v-if="nodesLoadError && activeTab === 'overview'" type="error" class="mb-4" :title="t('mobile.nodesLoadFailed')">
				<div class="flex items-center justify-between gap-3">
					<span>{{ nodesLoadError }}</span>
					<n-button size="small" text type="primary" @click="loadNodes()">{{ t("mobile.retry") }}</n-button>
				</div>
			</n-alert>

			<n-spin :show="loading">
				<div v-if="activeTab === 'overview'" class="space-y-4">
					<n-alert v-if="selectedNode && !nodeIsOnline" type="warning" :title="t(`mobile.nodeStatus_${selectedNode.status}`)">
						<div>{{ t("mobile.nodeSnapshotUnavailable") }}</div>
						<div v-if="selectedNode.lastSeenAt" class="mt-1 text-xs">{{ t("mobile.lastSeen", { time: new Date(selectedNode.lastSeenAt).toLocaleString() }) }}</div>
					</n-alert>
					<div class="grid grid-cols-2 gap-3">
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.cpu") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ nodeIsOnline ? `${cpuPercent}%` : '—' }}</div>
							<n-progress v-if="nodeIsOnline" type="line" :percentage="cpuPercent" :show-indicator="false" class="mt-3" />
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.memory") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ nodeIsOnline ? `${memoryPercent}%` : '—' }}</div>
							<n-progress v-if="nodeIsOnline" type="line" :percentage="memoryPercent" :show-indicator="false" class="mt-3" />
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.load") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ nodeIsOnline ? load1.toFixed(2) : '—' }}</div>
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.disk") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ nodeIsOnline ? `${Math.round(selectedNode?.summary.diskMaxPercent || 0)}%` : '—' }}</div>
						</div>
					</div>
					<section v-if="selectedNode && nodeIsOnline" class="rounded-2xl bg-white p-4 shadow-sm">
						<div class="grid grid-cols-2 gap-4 text-sm">
							<div><div class="text-xs text-slate-500">{{ t("mobile.runningContainers") }}</div><div class="mt-1 font-semibold">{{ selectedNode.summary.containerRunning }}/{{ selectedNode.summary.containerTotal }}</div></div>
							<div><div class="text-xs text-slate-500">{{ t("mobile.certificates") }}</div><div class="mt-1 font-semibold">{{ selectedNode.summary.certExpiringCount }}/{{ selectedNode.summary.certTotal }} {{ t("mobile.expiring") }}</div></div>
						</div>
						<div v-if="selectedNode.warnings.length" class="mt-3 flex flex-wrap gap-2">
							<n-tag v-for="(warning, index) in selectedNode.warnings" :key="index" size="small" :type="warning.level === 'danger' ? 'error' : 'warning'">{{ t(`mobile.warning_${warning.type}`) }}</n-tag>
						</div>
					</section>
					<section>
						<div class="flex items-center justify-between my-6">
							<div class="flex items-center gap-2">
								<h2 class="text-xl">{{ t("mobile.controllerSessions") }}</h2>
								<n-tag v-if="overview?.pendingApprovals.length" size="small" type="warning" :bordered="false">
									{{ t("mobile.pendingCount", { count: overview.pendingApprovals.length }) }}
								</n-tag>
							</div>
							<n-button size="small" text type="primary" @click="activeTab = 'code'">{{ t("mobile.code") }}</n-button>
						</div>
						<n-empty v-if="!overview?.sessions.length" size="small" :description="t('mobile.noSessions')" />
						<div v-else class="space-y-3">
							<button v-for="session in overview?.sessions || []" :key="session.id" class="w-full rounded-2xl border border-slate-200 bg-white p-4 text-left shadow-sm transition active:scale-[0.99]" @click="openSession(session)">
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0 flex-1">
										<div class="truncate font-semibold text-slate-900">{{ sessionTaskTitle(session) }}</div>
										<div class="mt-1.5 flex items-center gap-1.5 text-sm font-medium text-blue-600">
											<Icon name="mdi:folder-outline" :size="16" />
											<span class="truncate">{{ sessionProjectName(session) }}</span>
										</div>
										<div class="mt-1 flex items-center gap-1.5 text-xs text-slate-500">
											<Icon name="mdi:robot-outline" :size="14" />
											<span>{{ session.agentName }}</span>
											<span v-if="session.providerModel" class="truncate">· {{ session.providerModel }}</span>
										</div>
									</div>
									<n-tag size="small" :type="sessionStageType(session)" :bordered="false" round>
										{{ sessionStageLabel(session) }}
									</n-tag>
								</div>
								<div class="mt-3 flex items-center gap-2 text-xs text-slate-500">
									<Icon name="mdi:source-branch" :size="15" class="shrink-0" />
									<span class="min-w-0 flex-1 truncate">{{ session.workDir }}</span>
								</div>
								<div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-slate-100 pt-3 text-xs text-slate-400">
									<span>{{ sessionApprovalLabel(session) }}</span>
									<span v-if="session.worktreeBranch" class="max-w-full truncate">
										{{ session.worktreeBranch }}
									</span>
									<span class="ml-auto">{{ formatSessionTime(session.createdAt) }}</span>
								</div>
							</button>
						</div>
					</section>
				</div>

				<MobileContainerPanel v-else-if="activeTab === 'containers'" />

				<div v-else :class="isTaskDetail ? '' : 'space-y-4'">
					<div v-if="!isTaskDetail" class="flex items-center gap-2 overflow-x-auto pb-1">
						<n-button size="small" round type="primary" secondary class="shrink-0" @click="showSessionCreator = true">+ {{ t("mobile.newSession") }}</n-button>
						<n-button size="small" round secondary class="shrink-0" @click="showProjectTerminals = true">
							<template #icon><Icon name="mdi:console-line" /></template>
							{{ t("mobile.projectTerminal") }}
						</n-button>
						<n-button v-for="session in sessions" :key="session.id" size="small" round :type="selectedSessionId === session.id ? 'primary' : 'default'" @click="selectSession(session)">
							{{ sessionTaskTitle(session) }}
						</n-button>
					</div>
					<n-empty v-if="!isTaskDetail && sessions.length === 0" :description="t('mobile.noSessions')" class="rounded-2xl bg-white py-16">
						<template #extra>
							<n-button type="primary" @click="showSessionCreator = true">
								{{ t("mobile.newSession") }}
							</n-button>
						</template>
					</n-empty>
					<MobileTerminal
						v-if="projectTerminalSession && projectTerminalProject"
						:session-id="projectTerminalSession.id"
						:task-name="t('mobile.projectTerminal')"
						:project-name="projectTerminalProject.name"
						mode="native"
						@back="leaveTaskDetail"
					/>
					<template v-else-if="selectedSession">
						<MobileTerminal :session-id="selectedSessionId" :task-name="sessionTaskTitle(selectedSession)" :project-name="sessionProjectName(selectedSession)" @back="leaveTaskDetail" @open-files="showFiles = true" @open-status="showTaskStatus = true" @renamed="loadSessions" />
					</template>
				</div>
			</n-spin>
		</main>

		<nav v-if="!isTaskDetail" class="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white/95 pb-[max(8px,env(safe-area-inset-bottom))] pt-1 backdrop-blur">
			<div class="mx-auto grid max-w-2xl grid-cols-3">
				<button class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors" :class="activeTab === 'overview' ? 'text-blue-600' : 'text-slate-400'" @click="switchToOverview">
					<Icon :name="activeTab === 'overview' ? 'mdi:view-dashboard' : 'mdi:view-dashboard-outline'" :size="23" />
					<span>{{ t("mobile.overview") }}</span>
				</button>
				<button class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors" :class="activeTab === 'containers' ? 'text-blue-600' : 'text-slate-400'" @click="activeTab = 'containers'">
					<Icon :name="activeTab === 'containers' ? 'mdi:cube' : 'mdi:cube-outline'" :size="23" />
					<span>{{ t("mobile.containers") }}</span>
				</button>
				<button class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors" :class="activeTab === 'code' ? 'text-blue-600' : 'text-slate-400'" @click="switchToCode">
					<Icon name="mdi:console-line" :size="23" />
					<span>{{ t("mobile.code") }}</span>
				</button>
			</div>
		</nav>
		<MobileNodeSwitcher v-model:show="showNodeSwitcher" :nodes="nodes" :selected-id="selectedNodeId" :loading="nodesLoading" @select="selectNode" />
		<MobileProjectTerminals v-model:show="showProjectTerminals" :projects="projects" @opened="openProjectTerminal" />
		<MobileSessionCreator v-model:show="showSessionCreator" @created="handleSessionCreated" />
		<MobileFileBrowser v-if="selectedSessionId" v-model:show="showFiles" :session-id="selectedSessionId" />
		<MobileTaskStatusDrawer v-model:show="showTaskStatus" :state="sessionState" :loading="actionLoading" @approve="reason => decideApproval(true, reason)" @reject="reason => decideApproval(false, reason)" @stop="stopExecution" @retry="retryExecution" />
	</div>
</template>
