<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import { createHostTerminalSession, getHostTerminalAuditEvents, getHostTerminalCapabilities, listHostTerminalSessions, stopHostTerminalSession } from "@/api/modules/hostTerminal"
import type { HostTerminalAuditEvent, HostTerminalSession } from "@/api/interface/hostTerminal"
import HostTerminalPanel from "./HostTerminalPanel.vue"

const { t } = useI18n()
const message = useMessage()
const sessions = ref<HostTerminalSession[]>([])
const selectedId = ref<number | null>(null)
const loading = ref(false)
const creating = ref(false)
const loadError = ref("")
const shell = ref("default")
const workDir = ref("")
const availableShells = ref<string[]>([])
const auditVisible = ref(false)
const auditLoading = ref(false)
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
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("terminal.loadFailed")
		message.error(loadError.value)
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
		message.success(t("terminal.createSuccess"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("terminal.createFailed"))
	} finally {
		creating.value = false
	}
}

async function stopSession() {
	if (!selectedSession.value || selectedSession.value.status !== "running") return
	try {
		await stopHostTerminalSession(selectedSession.value.id)
		message.success(t("terminal.stopSuccess"))
		await loadSessions(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("terminal.stopFailed"))
	}
}

async function openAudit() {
	if (!selectedSession.value) return
	auditVisible.value = true
	auditLoading.value = true
	try {
		const response = await getHostTerminalAuditEvents(selectedSession.value.id)
		auditEvents.value = response.data || []
	} catch (error) {
		auditEvents.value = []
		message.error(error instanceof Error ? error.message : t("terminal.auditLoadFailed"))
	} finally {
		auditLoading.value = false
	}
}

function handleClosed() {
	void loadSessions(true)
}

onMounted(() => loadSessions())
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
			<n-button v-if="selectedSession" size="small" secondary @click="openAudit">{{ t("terminal.audit") }}</n-button>
			<n-button v-if="selectedSession?.status === 'running'" size="small" type="error" secondary @click="stopSession">{{ t("terminal.stopSession") }}</n-button>
			<span class="ml-auto text-xs text-slate-400">{{ t("terminal.securityHint") }}</span>
		</div>

		<div class="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
			<aside class="flex max-h-64 min-h-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm lg:max-h-none">
				<div class="border-b border-slate-200 px-4 py-3 text-sm font-semibold text-slate-700">{{ t("terminal.sessions") }}</div>
				<n-spin :show="loading" class="min-h-0 flex-1">
					<n-alert v-if="loadError" type="error" class="m-3">{{ loadError }}</n-alert>
					<n-empty v-else-if="!sessions.length" class="mt-20" :description="t('terminal.emptyTerminal')" />
					<n-scrollbar v-else class="h-full">
						<button v-for="session in sessions" :key="session.id" type="button" class="w-full border-b border-slate-100 px-4 py-3 text-left transition-colors hover:bg-slate-50" :class="selectedId === session.id ? 'bg-blue-50' : ''" @click="selectedId = session.id">
							<div class="flex items-center justify-between gap-2">
								<span class="text-sm font-medium text-slate-700">{{ session.shell }} #{{ session.id }}</span>
								<n-tag size="small" :type="statusType(session.status)" :bordered="false">{{ t(`terminal.status_${session.status}`) }}</n-tag>
							</div>
							<div class="mt-1 truncate text-xs text-slate-400" :title="session.workDir">{{ session.workDir }}</div>
							<div class="mt-1 text-[11px] text-slate-400">{{ new Date(session.startedAt).toLocaleString() }}</div>
						</button>
					</n-scrollbar>
				</n-spin>
			</aside>

			<main class="min-h-0 min-w-0">
				<HostTerminalPanel v-if="selectedSession?.status === 'running'" :key="selectedSession.id" :session-id="selectedSession.id" @closed="handleClosed" @status="handleClosed" />
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
