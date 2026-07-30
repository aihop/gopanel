<script setup lang="ts">
import {
	decideMobileApproval,
	getMobileNodes,
	getMobileOverview,
	getMobileSessionState,
	getMobileSessions,
	logoutMobileDevice,
	retryMobileInstruction,
	sendMobileInstruction,
	stopMobileSession,
	type MobileNode,
	type MobileOverview
} from "@/api/modules/mobile"
import type { CodeSession, CodeSessionState } from "@/api/interface/code"
import { mobileMessages } from "@/i18n/locales/mobile"
import { useI18n } from "vue-i18n"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useRouter } from "vue-router"
import MobileNodeSwitcher from "./components/MobileNodeSwitcher.vue"

const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const isHttp = window.location.protocol === "http:"
const activeTab = ref<"overview" | "code">("overview")
const overview = ref<MobileOverview | null>(null)
const nodes = ref<MobileNode[]>([])
const selectedNodeId = ref(Number(localStorage.getItem("gopanel-mobile-node-id") || 0))
const showNodeSwitcher = ref(false)
const nodesLoading = ref(false)
const nodesLoadError = ref("")
const sessions = ref<CodeSession[]>([])
const selectedSessionId = ref(0)
const sessionState = ref<CodeSessionState | null>(null)
const prompt = ref("")
const loading = ref(false)
const actionLoading = ref(false)
const loadError = ref("")
let refreshTimer: ReturnType<typeof setInterval> | null = null
let nodeRefreshTicks = 0

const selectedSession = computed(() => sessions.value.find(item => item.id === selectedSessionId.value) || null)
const selectedNode = computed(() => nodes.value.find(item => item.id === selectedNodeId.value) || nodes.value[0] || null)
const isRunning = computed(() => sessionState.value?.currentStage === "executing" || sessionState.value?.latestRun?.status === "running")
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
		if (activeTab.value === "overview") {
			if (selectedNode.value?.isLocal) void loadOverview(true)
			nodeRefreshTicks++
			if (nodeRefreshTicks >= 5) {
				nodeRefreshTicks = 0
				void loadNodes(true)
				if (!selectedNode.value?.isLocal) void loadOverview(true)
			}
		}
		else if (selectedSessionId.value) void loadSessionState(true)
	}, 2000)
}

onMounted(async () => {
	await Promise.all([loadOverview(), loadNodes(), loadSessions()])
	startRefresh()
})

onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
	<div class="min-h-dvh bg-slate-100 pb-24 text-slate-900">
		<header class="sticky top-0 z-20 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur">
			<div class="mx-auto flex max-w-2xl items-center justify-between">
				<div class="min-w-0">
					<div class="text-lg font-bold">GoPanel</div>
					<button type="button" class="mt-0.5 flex max-w-[65vw] items-center gap-1 border-0 bg-transparent p-0 text-xs text-slate-500" @click="showNodeSwitcher = true">
						<span class="h-2 w-2 shrink-0 rounded-full" :class="nodeIsOnline ? 'bg-emerald-500' : 'bg-slate-400'"></span>
						<span class="truncate">{{ selectedNode?.name || t("mobile.selectNode") }}</span>
						<span>⌄</span>
					</button>
				</div>
				<n-button size="small" quaternary @click="confirmLogout">{{ t("mobile.logout") }}</n-button>
			</div>
		</header>

		<main class="mx-auto max-w-2xl p-4">
			<n-alert v-if="isHttp" type="warning" :show-icon="false" class="mb-4">{{ t("mobile.httpWarning") }}</n-alert>
			<n-alert v-if="loadError" type="error" class="mb-4" :title="t('mobile.loadFailed')">{{ loadError }}</n-alert>
			<n-alert v-if="nodesLoadError" type="error" class="mb-4" :title="t('mobile.nodesLoadFailed')">
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
							<div><div class="text-xs text-slate-500">{{ t("mobile.containers") }}</div><div class="mt-1 font-semibold">{{ selectedNode.summary.containerRunning }}/{{ selectedNode.summary.containerTotal }}</div></div>
							<div><div class="text-xs text-slate-500">{{ t("mobile.certificates") }}</div><div class="mt-1 font-semibold">{{ selectedNode.summary.certExpiringCount }}/{{ selectedNode.summary.certTotal }} {{ t("mobile.expiring") }}</div></div>
						</div>
						<div v-if="selectedNode.warnings.length" class="mt-3 flex flex-wrap gap-2">
							<n-tag v-for="(warning, index) in selectedNode.warnings" :key="index" size="small" :type="warning.level === 'danger' ? 'error' : 'warning'">{{ t(`mobile.warning_${warning.type}`) }}</n-tag>
						</div>
					</section>
					<section class="rounded-2xl bg-white p-4 shadow-sm">
						<div class="mb-3 flex items-center justify-between">
							<div class="flex items-center gap-2">
								<h2 class="font-semibold">{{ t("mobile.controllerSessions") }}</h2>
								<n-tag v-if="overview?.pendingApprovals.length" size="small" type="warning" :bordered="false">
									{{ t("mobile.pendingCount", { count: overview.pendingApprovals.length }) }}
								</n-tag>
							</div>
							<n-button size="small" text type="primary" @click="activeTab = 'code'">{{ t("mobile.code") }}</n-button>
						</div>
						<n-empty v-if="!overview?.sessions.length" size="small" :description="t('mobile.noSessions')" />
						<button v-for="session in overview?.sessions || []" :key="session.id" class="mb-2 flex w-full items-center justify-between rounded-xl border-0 bg-slate-50 px-3 py-3 text-left" @click="activeTab = 'code'; selectSession(session)">
							<span class="min-w-0 truncate text-sm font-medium">{{ session.title }}</span>
							<n-tag size="small" :bordered="false">{{ session.currentStage }}</n-tag>
						</button>
					</section>
				</div>

				<div v-else class="space-y-4">
					<div class="flex gap-2 overflow-x-auto pb-1">
						<n-button v-for="session in sessions" :key="session.id" size="small" round :type="selectedSessionId === session.id ? 'primary' : 'default'" @click="selectSession(session)">{{ session.title }}</n-button>
					</div>
					<n-empty v-if="sessions.length === 0" :description="t('mobile.noSessions')" class="rounded-2xl bg-white py-16" />
					<template v-else-if="selectedSession">
						<section class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="flex items-center justify-between gap-3">
								<div class="min-w-0">
									<h2 class="truncate font-semibold">{{ selectedSession.title }}</h2>
									<div class="mt-1 truncate text-xs text-slate-500">{{ selectedSession.workDir }}</div>
								</div>
								<n-tag :type="sessionState?.currentStage === 'failed' ? 'error' : isRunning ? 'info' : 'success'">{{ sessionState?.currentStage || selectedSession.currentStage }}</n-tag>
							</div>
							<div class="mt-4 max-h-[42dvh] space-y-3 overflow-y-auto">
								<div v-for="item in sessionState?.recentMessages || []" :key="item.id" class="flex" :class="item.role === 'user' ? 'justify-end' : 'justify-start'">
									<pre class="max-w-[90%] whitespace-pre-wrap break-words rounded-2xl px-3 py-2 font-sans text-sm" :class="item.role === 'user' ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-700'">{{ item.content }}</pre>
								</div>
							</div>
						</section>

						<n-alert v-if="sessionState?.pendingApproval" type="warning" :title="sessionState.pendingApproval.title">
							<div class="whitespace-pre-wrap text-sm">{{ sessionState.pendingApproval.content }}</div>
							<div class="mt-3 flex gap-2">
								<n-button type="warning" :loading="actionLoading" @click="decideApproval(true)">{{ t("mobile.approve") }}</n-button>
								<n-button :disabled="actionLoading" @click="decideApproval(false)">{{ t("mobile.reject") }}</n-button>
							</div>
						</n-alert>

						<n-alert v-if="sessionState?.errorSummary" type="error">{{ sessionState.errorSummary }}</n-alert>
						<section v-if="sessionState?.changedFiles.length" class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="mb-2 text-sm font-semibold">{{ t("mobile.changedFiles") }}</div>
							<div class="flex flex-wrap gap-2"><n-tag v-for="file in sessionState.changedFiles" :key="file" size="small">{{ file }}</n-tag></div>
						</section>
						<section v-if="sessionState?.previews.length" class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="mb-2 text-sm font-semibold">{{ t("mobile.previews") }}</div>
							<a v-for="preview in sessionState.previews" :key="preview.id" :href="preview.url" target="_blank" class="mr-3 text-sm text-blue-600">{{ preview.title }}</a>
						</section>
						<div class="flex gap-2">
							<n-button v-if="isRunning" type="error" secondary :loading="actionLoading" @click="stopExecution">{{ t("mobile.stop") }}</n-button>
							<n-button v-if="['failed', 'cancelled'].includes(sessionState?.latestInstruction?.status || '')" secondary :loading="actionLoading" @click="retryExecution">{{ t("mobile.retryExecution") }}</n-button>
						</div>
						<div class="sticky bottom-20 rounded-2xl border border-slate-200 bg-white p-3 shadow-lg">
							<n-input v-model:value="prompt" type="textarea" :placeholder="t('mobile.instructionPlaceholder')" :autosize="{ minRows: 2, maxRows: 5 }" :disabled="actionLoading" />
							<n-button type="primary" block class="mt-2" :loading="actionLoading" :disabled="!prompt.trim()" @click="sendInstruction">{{ t("mobile.send") }}</n-button>
						</div>
					</template>
				</div>
			</n-spin>
		</main>

		<nav class="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white px-4 pb-[max(12px,env(safe-area-inset-bottom))] pt-2">
			<div class="mx-auto grid max-w-2xl grid-cols-2 gap-2">
				<n-button size="large" :type="activeTab === 'overview' ? 'primary' : 'default'" :secondary="activeTab !== 'overview'" @click="activeTab = 'overview'; loadOverview()">{{ t("mobile.overview") }}</n-button>
				<n-button size="large" :type="activeTab === 'code' ? 'primary' : 'default'" :secondary="activeTab !== 'code'" @click="activeTab = 'code'; loadSessions()">{{ t("mobile.code") }}</n-button>
			</div>
		</nav>
		<MobileNodeSwitcher v-model:show="showNodeSwitcher" :nodes="nodes" :selected-id="selectedNodeId" :loading="nodesLoading" @select="selectNode" />
	</div>
</template>
