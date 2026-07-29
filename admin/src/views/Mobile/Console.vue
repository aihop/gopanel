<script setup lang="ts">
import {
	decideMobileApproval,
	getMobileOverview,
	getMobileSessionState,
	getMobileSessions,
	logoutMobileDevice,
	retryMobileInstruction,
	sendMobileInstruction,
	stopMobileSession,
	type MobileOverview
} from "@/api/modules/mobile"
import type { CodeSession, CodeSessionState } from "@/api/interface/code"
import { mobileMessages } from "@/i18n/locales/mobile"
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
const activeTab = ref<"overview" | "code">("overview")
const overview = ref<MobileOverview | null>(null)
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
const isRunning = computed(() => sessionState.value?.currentStage === "executing" || sessionState.value?.latestRun?.status === "running")
const memoryPercent = computed(() => Math.round(overview.value?.system.memoryUsedPercent || 0))
const cpuPercent = computed(() => Math.round(overview.value?.system.cpuUsedPercent || 0))

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
		else if (selectedSessionId.value) void loadSessionState(true)
	}, 2000)
}

onMounted(async () => {
	await Promise.all([loadOverview(), loadSessions()])
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
				<div>
					<div class="text-lg font-bold">GoPanel</div>
					<div class="text-xs text-slate-500">{{ t("mobile.title") }}</div>
				</div>
				<div class="flex items-center gap-1">
					<n-button size="small" type="primary" secondary @click="showSessionCreator = true">{{ t("mobile.newSession") }}</n-button>
					<n-button size="small" quaternary @click="confirmLogout">{{ t("mobile.logout") }}</n-button>
				</div>
			</div>
		</header>

		<main class="mx-auto max-w-2xl p-4">
			<n-alert v-if="isHttp" type="warning" :show-icon="false" class="mb-4">{{ t("mobile.httpWarning") }}</n-alert>
			<n-alert v-if="loadError" type="error" class="mb-4" :title="t('mobile.loadFailed')">{{ loadError }}</n-alert>

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
							<div class="mt-2 text-2xl font-bold">{{ overview?.system.load1?.toFixed(2) || '0.00' }}</div>
						</div>
						<div class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="text-xs text-slate-500">{{ t("mobile.pending") }}</div>
							<div class="mt-2 text-2xl font-bold">{{ overview?.pendingApprovals.length || 0 }}</div>
						</div>
					</div>
					<section class="rounded-2xl bg-white p-4 shadow-sm">
						<div class="mb-3 flex items-center justify-between">
							<h2 class="font-semibold">{{ t("mobile.sessions") }}</h2>
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
					<div class="flex items-center gap-2 overflow-x-auto pb-1">
						<n-button size="small" round type="primary" secondary class="shrink-0" @click="showSessionCreator = true">+ {{ t("mobile.newSession") }}</n-button>
						<n-button v-for="session in sessions" :key="session.id" size="small" round :type="selectedSessionId === session.id ? 'primary' : 'default'" @click="selectSession(session)">{{ session.title }}</n-button>
					</div>
					<n-empty v-if="sessions.length === 0" :description="t('mobile.noSessions')" class="rounded-2xl bg-white py-16">
						<template #extra><n-button type="primary" @click="showSessionCreator = true">{{ t("mobile.newSession") }}</n-button></template>
					</n-empty>
					<template v-else-if="selectedSession">
						<section class="rounded-2xl bg-white p-4 shadow-sm">
							<div class="flex flex-wrap items-center justify-between gap-3">
								<div class="min-w-0">
									<h2 class="truncate font-semibold">{{ selectedSession.title }}</h2>
									<div class="mt-1 truncate text-xs text-slate-500">{{ selectedSession.workDir }}</div>
								</div>
								<div class="flex shrink-0 items-center gap-2">
									<n-button size="small" secondary @click="showFiles = true">{{ t("mobile.files") }}</n-button>
									<n-tag :type="sessionState?.currentStage === 'failed' ? 'error' : isRunning ? 'info' : 'success'">{{ sessionState?.currentStage || selectedSession.currentStage }}</n-tag>
								</div>
							</div>
						</section>
						<MobileTerminal :session-id="selectedSessionId" />
						<div v-if="sessionState?.recentMessages.length" class="max-h-[42dvh] space-y-3 overflow-y-auto px-1">
							<div v-for="item in sessionState.recentMessages" :key="item.id" class="flex" :class="item.role === 'user' ? 'justify-end' : 'justify-start'">
								<pre class="max-w-[90%] whitespace-pre-wrap break-words rounded-2xl px-3 py-2 font-sans text-sm" :class="item.role === 'user' ? 'bg-blue-600 text-white' : 'bg-white text-slate-700'">{{ item.content }}</pre>
							</div>
						</div>
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

			<MobileSessionCreator v-model:show="showSessionCreator" @created="handleSessionCreated" />
			<MobileFileBrowser v-if="selectedSessionId" v-model:show="showFiles" :session-id="selectedSessionId" />
		</div>
</template>
