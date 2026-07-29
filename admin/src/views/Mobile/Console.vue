<script setup lang="ts">
import { decideMobileApproval, getMobileOverview, getMobileProjects, getMobileSessionState, getMobileSessions, logoutMobileDevice, retryMobileInstruction, sendMobileInstruction, stopMobileSession, type MobileOverview } from "@/api/modules/mobile"
import type { AIGroup, CodeSession, CodeSessionState } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import Logo from "@/layouts/common/Logo.vue"
import MobileContainerPanel from "./components/MobileContainerPanel.vue"
import MobileFileBrowser from "./components/MobileFileBrowser.vue"
import MobileSessionCreator from "./components/MobileSessionCreator.vue"
import MobileTerminal from "./components/MobileTerminal.vue"
import { useI18n } from "vue-i18n"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useRouter } from "vue-router"

const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const isHttp = window.location.protocol === "http:"
const activeTab = ref<"overview" | "containers" | "code">("overview")
const overview = ref<MobileOverview | null>(null)
const projects = ref<AIGroup[]>([])
const sessions = ref<CodeSession[]>([])
const selectedSessionId = ref(0)
const sessionState = ref<CodeSessionState | null>(null)
const prompt = ref("")
const loading = ref(false)
const actionLoading = ref(false)
const loadError = ref("")
const showSessionCreator = ref(false)
const showFiles = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const selectedSession = computed(() => sessions.value.find(item => item.id === selectedSessionId.value) || null)
const isTaskDetail = computed(() => activeTab.value === "code" && Boolean(selectedSession.value))
const isRunning = computed(() => sessionState.value?.currentStage === "executing" || sessionState.value?.latestRun?.status === "running")
const memoryPercent = computed(() => Math.round(overview.value?.system.memoryUsedPercent || 0))
const cpuPercent = computed(() => Math.round(overview.value?.system.cpuUsedPercent || 0))

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

function sessionProjectName(session: CodeSession) {
	const project = projects.value.find(item => item.id === session.projectId)
	if (project) return project.name
	if (session.projectId) return `${t("mobile.project")} #${session.projectId}`
	const segments = session.workDir.split(/[\\/]/).filter(Boolean)
	return segments.at(-1) || session.workDir
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
		if (!selectedSessionId.value && sessions.value.length) {
			selectedSessionId.value = sessions.value[0].id
		}
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
	await loadSessions()
}

async function leaveTaskDetail() {
	activeTab.value = "overview"
	await loadOverview()
}

async function handleSessionCreated(session: CodeSession) {
	activeTab.value = "code"
	selectedSessionId.value = session.id
	await loadSessions()
	await loadSessionState()
}

async function sendInstruction() {
	const content = prompt.value.trim()
	if (!content || !selectedSessionId.value || actionLoading.value) return
	actionLoading.value = true
	try {
		await sendMobileInstruction(selectedSessionId.value, content)
		prompt.value = ""
		message.success(t("mobile.instructionQueued"))
		await loadSessionState(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
	} finally {
		actionLoading.value = false
	}
}

async function decideApproval(approved: boolean) {
	const approvalId = sessionState.value?.pendingApproval?.id
	if (!approvalId) return
	actionLoading.value = true
	try {
		await decideMobileApproval(approvalId, approved)
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
		if (activeTab.value === "overview") void loadOverview(true)
		else if (activeTab.value === "code" && selectedSessionId.value) void loadSessionState(true)
	}, 2000)
}

onMounted(async () => {
	await Promise.all([loadOverview(), loadProjects(), loadSessions()])
	startRefresh()
})

onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
	<div class="min-h-dvh bg-slate-100 pb-24 text-slate-900">
		<header class="sticky top-0 z-20 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur">
			<div v-if="isTaskDetail" class="mx-auto grid max-w-2xl grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
				<n-button size="small" quaternary @click="leaveTaskDetail">
					<template #icon><Icon name="mdi:arrow-left" /></template>
					{{ t("commons.button.back") }}
				</n-button>
				<div class="truncate text-center text-sm font-semibold">{{ selectedSession?.title }}</div>
				<n-button size="small" type="primary" secondary @click="showFiles = true">
					{{ t("mobile.files") }}
				</n-button>
			</div>
			<div v-else class="mx-auto flex max-w-2xl items-center justify-between">
				<Logo :dark="false" class="shrink-0" />
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

		<main class="mx-auto max-w-2xl" :class="isTaskDetail ? 'p-2' : 'p-4'">
			<n-alert v-if="isHttp && !isTaskDetail" type="warning" :show-icon="false" class="mb-4">
				{{ t("mobile.httpWarning") }}
			</n-alert>
			<n-alert v-if="loadError && activeTab !== 'containers'" type="error" class="mb-4" :title="t('mobile.loadFailed')">
				{{ loadError }}
			</n-alert>

			<n-spin :show="loading">
				<div v-if="activeTab === 'overview'" class="space-y-4">
					<div class="grid grid-cols-2 gap-3">
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.cpu") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ cpuPercent }}%</div>
							<n-progress type="line" :percentage="cpuPercent" :show-indicator="false" class="mt-3" />
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.memory") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ memoryPercent }}%</div>
							<n-progress type="line" :percentage="memoryPercent" :show-indicator="false" class="mt-3" />
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.load") }}</div>
							<div class="mt-2 text-2xl font-bold">
								{{ overview?.system.load1?.toFixed(2) || "0.00" }}
							</div>
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.pending") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ overview?.pendingApprovals.length || 0 }}</div>
						</div>
					</div>
					<section>
						<div class="mb-3 flex items-center justify-between">
							<h2 class="font-semibold">{{ t("mobile.sessions") }}</h2>
							<n-button size="small" text type="primary" @click="activeTab = 'code'">
								{{ t("mobile.code") }}
							</n-button>
						</div>
						<n-empty v-if="!overview?.sessions.length" size="small" :description="t('mobile.noSessions')" />
						<div v-else class="space-y-3">
							<button v-for="session in overview?.sessions || []" :key="session.id" class="w-full rounded-2xl border border-slate-200 bg-white p-4 text-left shadow-sm transition active:scale-[0.99]" @click="openSession(session)">
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0 flex-1">
										<div class="truncate font-semibold text-slate-900">{{ session.title }}</div>
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

				<div v-else :class="isTaskDetail ? 'space-y-2' : 'space-y-4'">
					<div v-if="!isTaskDetail" class="flex items-center gap-2 overflow-x-auto pb-1">
						<n-button size="small" round type="primary" secondary class="shrink-0" @click="showSessionCreator = true">+ {{ t("mobile.newSession") }}</n-button>
						<n-button v-for="session in sessions" :key="session.id" size="small" round :type="selectedSessionId === session.id ? 'primary' : 'default'" @click="selectSession(session)">
							{{ session.title }}
						</n-button>
					</div>
					<n-empty v-if="sessions.length === 0" :description="t('mobile.noSessions')" class="rounded-2xl bg-white py-16">
						<template #extra>
							<n-button type="primary" @click="showSessionCreator = true">
								{{ t("mobile.newSession") }}
							</n-button>
						</template>
					</n-empty>
					<template v-else-if="selectedSession">
						<MobileTerminal :session-id="selectedSessionId" />
						<n-alert v-if="sessionState?.pendingApproval" type="warning" :title="sessionState.pendingApproval.title">
							<div class="whitespace-pre-wrap text-sm">{{ sessionState.pendingApproval.content }}</div>
							<div class="mt-3 flex gap-2">
								<n-button type="warning" :loading="actionLoading" @click="decideApproval(true)">
									{{ t("mobile.approve") }}
								</n-button>
								<n-button :disabled="actionLoading" @click="decideApproval(false)">
									{{ t("mobile.reject") }}
								</n-button>
							</div>
						</n-alert>

						<n-alert v-if="sessionState?.errorSummary" type="error">
							{{ sessionState.errorSummary }}
						</n-alert>
						<div class="flex gap-2">
							<n-button v-if="isRunning" type="error" secondary :loading="actionLoading" @click="stopExecution">
								{{ t("mobile.stop") }}
							</n-button>
							<n-button v-if="['failed', 'cancelled'].includes(sessionState?.latestInstruction?.status || '')" secondary :loading="actionLoading" @click="retryExecution">
								{{ t("mobile.retryExecution") }}
							</n-button>
						</div>
						<div class="fixed inset-x-0 bottom-0 z-30 px-3 pb-[max(12px,env(safe-area-inset-bottom))]">
							<div class="mx-auto flex max-w-2xl items-end gap-2 rounded-2xl border border-slate-200 bg-white p-2 shadow-xl">
								<n-input v-model:value="prompt" class="min-w-0 flex-1" type="textarea" :placeholder="t('mobile.instructionPlaceholder')" :autosize="{ minRows: 1, maxRows: 4 }" :disabled="actionLoading" />
								<n-button type="primary" :loading="actionLoading" :disabled="!prompt.trim()" @click="sendInstruction">
									{{ t("mobile.send") }}
								</n-button>
							</div>
						</div>
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

		<MobileSessionCreator v-model:show="showSessionCreator" @created="handleSessionCreated" />
		<MobileFileBrowser v-if="selectedSessionId" v-model:show="showFiles" :session-id="selectedSessionId" />
	</div>
</template>
