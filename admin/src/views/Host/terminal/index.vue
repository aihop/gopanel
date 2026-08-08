<script setup lang="ts">
defineOptions({ name: "HostTerminalView" })

import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useDialog, useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import { createHostTerminalSession, deleteHostTerminalSession, getHostTerminalAuditEvents, getHostTerminalCapabilities, listHostTerminalSessions, reconnectHostTerminalSession, stopHostTerminalSession } from "@/api/modules/hostTerminal"
import type { HostTerminalAuditEvent, HostTerminalSession } from "@/api/interface/hostTerminal"
import HostTerminalPanel from "./HostTerminalPanel.vue"

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const sessions = ref<HostTerminalSession[]>([])
const selectedId = ref<number | null>(null)
const connectedSessionId = ref<number | null>(null)
const loading = ref(false)
const creating = ref(false)
const loadError = ref("")
const shell = ref("default")
const workDir = ref("")
const availableShells = ref<string[]>([])
const auditVisible = ref(false)
const auditLoading = ref(false)
const stoppingId = ref<number | null>(null)
const reconnectingId = ref<number | null>(null)
const deletingId = ref<number | null>(null)
const auditEvents = ref<HostTerminalAuditEvent[]>([])
const selectedSession = computed(() => sessions.value.find(item => item.id === selectedId.value) || null)
const shellOptions = computed(() => [
	{ label: t("terminal.defaultShell"), value: "default" },
	...availableShells.value.map(value => ({ label: value, value }))
])

function statusType(status: HostTerminalSession["status"]): "success" | "error" | "warning" | "default" {
	if (status === "running" || status === "starting") return "success"
	if (status === "failed" || status === "interrupted") return "error"
	if (status === "stopped") return "warning"
	return "default"
}

function sessionActionBusy(sessionId: number) {
	return stoppingId.value === sessionId || reconnectingId.value === sessionId || deletingId.value === sessionId
}

async function loadSessions(silent = false) {
	if (!silent) loading.value = true
	try {
		const [response, capabilitiesResponse] = await Promise.all([listHostTerminalSessions(), getHostTerminalCapabilities()])
		sessions.value = response.data.items || []
		availableShells.value = capabilitiesResponse.data.shells || []
		loadError.value = ""
		if (!selectedId.value) {
			selectedId.value = sessions.value.find(item => item.status === "running")?.id || sessions.value[0]?.id || null
		}
		if (connectedSessionId.value && !sessions.value.some(item => item.id === connectedSessionId.value && item.status === "running")) {
			connectedSessionId.value = null
		}
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("terminal.loadFailed")
	} finally {
		loading.value = false
	}
}

async function createSession() {
	creating.value = true
	try {
		const response = await createHostTerminalSession({ shell: shell.value, workDir: workDir.value.trim(), cols: 120, rows: 32 })
		sessions.value.unshift(response.data)
		selectedId.value = response.data.id
		connectedSessionId.value = response.data.id
		message.success(t("terminal.createSuccess"))
	} catch (error) {
	} finally {
		creating.value = false
	}
}

async function endSession(session: HostTerminalSession) {
	if (session.status !== "running") return
	stoppingId.value = session.id
	try {
		await stopHostTerminalSession(session.id)
		if (connectedSessionId.value === session.id) connectedSessionId.value = null
		message.success(t("terminal.endSuccess"))
		await loadSessions(true)
	} catch (error) {
	} finally {
		stoppingId.value = null
	}
}

function confirmEndSession(session: HostTerminalSession) {
	dialog.warning({
		title: t("terminal.endSession"),
		content: t("terminal.endConfirm"),
		positiveText: t("terminal.endSession"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: () => endSession(session)
	})
}

function disconnectSession(session: HostTerminalSession) {
	if (connectedSessionId.value !== session.id) return
	connectedSessionId.value = null
	selectedId.value = session.id
	message.success(t("terminal.disconnectSuccess"))
}

async function reconnectSession(session: HostTerminalSession) {
	reconnectingId.value = session.id
	try {
		const response = await reconnectHostTerminalSession(session.id)
		await loadSessions(true)
		selectedId.value = response.data.id
		connectedSessionId.value = response.data.id
		message.success(t("terminal.reconnectSuccess"))
	} catch (error) {
	} finally {
		reconnectingId.value = null
	}
}

function confirmDeleteSession(session: HostTerminalSession) {
	const sessionId = session.id
	dialog.warning({
		title: t("terminal.deleteSession"),
		content: t("terminal.deleteConfirm"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			deletingId.value = sessionId
			try {
				await deleteHostTerminalSession(sessionId)
				if (selectedId.value === sessionId) selectedId.value = null
				await loadSessions(true)
				message.success(t("terminal.deleteSuccess"))
			} catch (error) {
			} finally {
				deletingId.value = null
			}
		}
	})
}

async function openAudit(session: HostTerminalSession) {
	selectedId.value = session.id
	auditVisible.value = true
	auditLoading.value = true
	try {
		const response = await getHostTerminalAuditEvents(session.id)
		auditEvents.value = response.data || []
	} catch (error) {
		auditEvents.value = []
	} finally {
		auditLoading.value = false
	}
}

function selectSession(session: HostTerminalSession) {
	if (connectedSessionId.value !== session.id) connectedSessionId.value = null
	selectedId.value = session.id
}

function handleClosed() {
	connectedSessionId.value = null
	void loadSessions(true)
}

onMounted(async () => {
	await loadSessions()
	if (sessions.value.some(item => item.id === selectedId.value && item.status === "running")) {
		connectedSessionId.value = selectedId.value
	}
})
</script>

<template>
	<div class="flex h-[calc(100vh-104px)] min-h-[560px] flex-col gap-4 p-4">
		<div class="flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-white p-3 shadow-sm">
			<div class="flex items-center gap-2 text-sm font-semibold text-slate-700">
				<Icon name="mdi:console" :size="20" />
				{{ t("terminal.hostTerminal") }}
			</div>
			<n-select v-model:value="shell" style="width: 150px" size="small" :options="shellOptions" />
			<n-input v-model:value="workDir" style="width: 280px" size="small" clearable :placeholder="t('terminal.workDirPlaceholder')" @keyup.enter="createSession" />
			<n-button size="small" type="primary" :loading="creating" @click="createSession">{{ t("terminal.createSession") }}</n-button>
			<n-button size="small" :loading="loading" @click="loadSessions()">{{ t("commons.button.refresh") }}</n-button>
			<span class="ml-auto text-xs text-slate-400">{{ t("terminal.securityHint") }}</span>
		</div>

		<div class="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[320px_minmax(0,1fr)]">
			<aside class="flex max-h-64 min-h-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm lg:max-h-none">
				<div class="border-b border-slate-200 px-4 py-3 text-sm font-semibold text-slate-700">{{ t("terminal.sessions") }}</div>
				<n-spin :show="loading" class="min-h-0 flex-1">
					<n-alert v-if="loadError" type="error" class="m-3">{{ loadError }}</n-alert>
					<n-empty v-else-if="!sessions.length" class="mt-20" :description="t('terminal.emptyTerminal')" />
					<n-scrollbar v-else class="h-full">
						<div v-for="session in sessions" :key="session.id" class="group flex items-center border-b border-slate-100 pr-2 transition-colors hover:bg-slate-50" :class="selectedId === session.id ? 'bg-blue-50 hover:bg-blue-50' : ''">
							<button type="button" class="min-w-0 flex-1 px-4 py-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500" @click="selectSession(session)">
								<div class="flex items-center justify-between gap-2">
									<span class="truncate text-sm font-medium text-slate-700">{{ session.shell }} #{{ session.id }}</span>
									<n-tag size="small" :type="statusType(session.status)" :bordered="false">{{ t(`terminal.status_${session.status}`) }}</n-tag>
								</div>
								<div class="mt-1 truncate text-xs text-slate-400" :title="session.workDir">{{ session.workDir }}</div>
								<div class="mt-1 text-[11px] text-slate-400">{{ new Date(session.startedAt).toLocaleString() }}</div>
							</button>
							<div class="flex shrink-0 flex-col gap-0.5 rounded-lg border border-slate-200/70 bg-white/80 p-0.5 opacity-100 shadow-sm transition-opacity" :class="selectedId === session.id ? 'sm:opacity-100' : 'sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100'">
								<n-tooltip placement="right" trigger="hover">
									<template #trigger>
										<n-button quaternary circle size="tiny" :disabled="sessionActionBusy(session.id)" :aria-label="t('terminal.audit')" @click="openAudit(session)">
											<template #icon><Icon name="mdi:history" :size="15" /></template>
										</n-button>
									</template>
									{{ t("terminal.audit") }}
								</n-tooltip>
								<n-tooltip v-if="session.status === 'running' && connectedSessionId === session.id" placement="right" trigger="hover">
									<template #trigger>
										<n-button quaternary circle size="tiny" :disabled="sessionActionBusy(session.id)" :aria-label="t('terminal.disconnectSession')" @click="disconnectSession(session)">
											<template #icon><Icon name="mdi:lan-disconnect" :size="15" /></template>
										</n-button>
									</template>
									{{ t("terminal.disconnectSession") }}
								</n-tooltip>
								<n-tooltip v-if="session.status === 'running' && connectedSessionId !== session.id" placement="right" trigger="hover">
									<template #trigger>
										<n-button quaternary circle size="tiny" type="primary" :loading="reconnectingId === session.id" :disabled="sessionActionBusy(session.id)" :aria-label="t('terminal.reconnectSession')" @click="reconnectSession(session)">
											<template #icon><Icon name="mdi:connection" :size="15" /></template>
										</n-button>
									</template>
									{{ t("terminal.reconnectSession") }}
								</n-tooltip>
								<n-tooltip v-if="session.status === 'running'" placement="right" trigger="hover">
									<template #trigger>
										<n-button quaternary circle size="tiny" type="error" :loading="stoppingId === session.id" :disabled="sessionActionBusy(session.id)" :aria-label="t('terminal.endSession')" @click="confirmEndSession(session)">
											<template #icon><Icon name="mdi:stop-circle-outline" :size="15" /></template>
										</n-button>
									</template>
									{{ t("terminal.endSession") }}
								</n-tooltip>
								<template v-if="session.status !== 'running' && session.status !== 'starting'">
									<n-tooltip placement="right" trigger="hover">
										<template #trigger>
											<n-button quaternary circle size="tiny" type="error" :loading="deletingId === session.id" :disabled="sessionActionBusy(session.id)" :aria-label="t('terminal.deleteSession')" @click="confirmDeleteSession(session)">
												<template #icon><Icon name="mdi:delete-outline" :size="15" /></template>
											</n-button>
										</template>
										{{ t("terminal.deleteSession") }}
									</n-tooltip>
								</template>
							</div>
						</div>
					</n-scrollbar>
				</n-spin>
			</aside>

			<main class="min-h-0 min-w-0">
				<HostTerminalPanel v-if="selectedSession?.status === 'running' && connectedSessionId === selectedSession.id" :key="selectedSession.id" :session-id="selectedSession.id" @closed="handleClosed" @status="handleClosed" />
				<div v-else-if="selectedSession?.status === 'running'" class="flex h-full items-center justify-center rounded-xl border border-slate-200 bg-white">
					<n-result status="info" :title="t('terminal.disconnected')" :description="t('terminal.disconnectedHint')" />
				</div>
				<div v-else-if="selectedSession" class="flex h-full items-center justify-center rounded-xl border border-slate-200 bg-white">
					<n-result :status="selectedSession.status === 'failed' || selectedSession.status === 'interrupted' ? 'error' : 'info'" :title="t(`terminal.status_${selectedSession.status}`)" :description="selectedSession.errorMessage || t('terminal.sessionEnded')" />
				</div>
				<div v-else class="flex h-full items-center justify-center rounded-xl border border-slate-200 bg-white">
					<n-empty :description="t('terminal.createFirstSession')" />
				</div>
			</main>
		</div>

		<n-drawer v-model:show="auditVisible" style="width: 480px">
			<n-drawer-content :title="t('terminal.audit')">
				<n-spin :show="auditLoading">
					<n-empty v-if="!auditLoading && !auditEvents.length" :description="t('terminal.auditEmpty')" />
					<n-timeline v-else>
						<n-timeline-item v-for="event in auditEvents" :key="event.id" :type="event.status === 'failed' || event.status === 'denied' ? 'error' : 'info'" :title="t(`terminal.audit_${event.action}`)" :content="event.detail || event.ip" :time="new Date(event.createdAt).toLocaleString()" />
					</n-timeline>
				</n-spin>
			</n-drawer-content>
		</n-drawer>
	</div>
</template>
