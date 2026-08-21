<template>
	<div class="website-index-root mt-4 space-y-8">
		<div class="flex flex-col gap-8 lg:flex-row lg:items-start lg:justify-between">
			<div class="max-w-3xl space-y-4">
				<div class="flex flex-wrap items-center gap-3">
					<div
						class="inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium"
						:class="
							httpServerStatus
								? 'border-emerald-200 bg-emerald-50 text-emerald-600'
								: 'border-rose-200 bg-rose-50 text-rose-500'
						"
					>
						<span
							class="h-2.5 w-2.5 rounded-full"
							:class="httpServerStatus ? 'bg-emerald-500' : 'bg-rose-500'"
						></span>
						{{ httpServerStatus ? $t("website.httpServiceRunning") : $t("website.httpServiceStopped") }}
						{{ statusStartErrorText }}
					</div>
					<div class="rounded-full border border-slate-200 bg-slate-50 px-4 py-2 text-sm text-slate-500">
						{{ $t("website.managedWebsitesCount", { count: total }) }}
					</div>
				</div>
			</div>
			<div class="flex flex-wrap gap-3 lg:justify-end">
				<n-button ghost type="primary" @click="handleReload">{{ $t("website.restart") }}</n-button>
				<n-button ghost :disabled="!httpServerStatus" @click="handleStop">
					{{ $t("commons.button.stop") }}
				</n-button>
			</div>
		</div>

		<n-alert
			v-if="!agentStatus.online"
			type="warning"
			:show-icon="true"
			:title="$t('website.agentNotInitialized')"
			class="bg-base-100 rounded-[28px] border border-blue-100/80 p-6 shadow-[0_18px_48px_rgba(15,23,42,0.08)]"
		>
			<div class="text-sm leading-7 text-slate-600">
				<div>{{ $t("website.agentNotRunning") }}</div>
				<div v-if="agentStatus.error" class="mt-1 text-slate-500">{{ agentStatus.error }}</div>
				<n-space class="mt-4">
					<n-button type="primary" :loading="ensuringAgent" @click="ensureAgent">
						{{ $t("website.oneClickInit") }}
					</n-button>
				</n-space>
			</div>
		</n-alert>
		<div class="bg-base-100 rounded-[28px] border border-blue-100/80 shadow-[0_24px_72px_rgba(15,23,42,0.08)]">
			<div
				class="flex flex-col gap-5 border-b border-slate-100 px-8 py-7 lg:flex-row lg:items-center lg:justify-between"
			>
				<div class="space-y-2">
					<div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Website Workspace</div>
					<div class="fg-base-100 text-2xl font-semibold">
						{{ activeTab === "list" ? $t("website.websiteList") : $t("website.config") }}
					</div>
					<div class="text-sm leading-7 text-slate-500">
						{{ activeTab === "list" ? $t("website.websiteListHelper") : $t("website.configHelper") }}
					</div>
				</div>
				<div class="flex flex-wrap gap-3">
					<div class="rounded-full bg-slate-100 p-1">
						<button
							class="rounded-full px-5 py-2 text-sm font-medium transition"
							:class="
								activeTab === 'list'
									? 'bg-base-100 text-blue-600 shadow-sm'
									: 'text-slate-500 hover:text-slate-100'
							"
							@click="activeTab = 'list'"
						>
							{{ $t("website.websiteList") }}
						</button>
						<button
							class="rounded-full px-5 py-2 text-sm font-medium transition"
							:class="
								activeTab === 'config'
									? 'bg-base-100 text-blue-600 shadow-sm'
									: 'text-slate-500 hover:text-slate-100'
							"
							@click="activeTab = 'config'"
						>
							{{ $t("website.config") }}
						</button>
					</div>
					<n-button v-if="activeTab === 'list'" ghost @click="fetchData">
						{{ $t("website.refreshList") }}
					</n-button>
					<n-button v-if="activeTab === 'list'" type="primary" @click="handleAdd">
						{{ $t("website.createWebsite") }}
					</n-button>
				</div>
			</div>

			<div class="p-8">
				<div v-if="activeTab === 'list'" class="space-y-5">
					<n-data-table
						:loading="loading"
						:columns="columns"
						:data="tableData"
						:bordered="false"
						class="rounded-3xl"
					>
						<template #empty>
							<div class="flex flex-col items-center justify-center py-14">
								<n-empty :description="$t('website.noWebsite')" class="mb-3" />
								<n-button type="primary" ghost class="mt-6" @click="handleAdd">
									{{ $t("website.createNow") }}
								</n-button>
							</div>
						</template>
					</n-data-table>
				</div>

				<div v-else class="overflow-hidden rounded-3xl border border-slate-100 bg-slate-50/70">
					<HttpConfigFile :scope-summary="$t('website.httpConfigScopeSummary')" />
				</div>
			</div>
		</div>

		<Create ref="createRef" @confirm="postConfirm" />

		<AppDeployHistory ref="appDeployHistoryRef" @confirm="postConfirm" />

		<AccessLogDrawer ref="accessLogDrawerRef" />

		<SecurityDrawer ref="securityDrawerRef" @confirm="fetchData" />

		<WebsiteDiagnosticDrawer ref="diagnosticDrawerRef" @confirm="fetchData" />

		<OpDialog ref="opDialogRef" @search="handleEnsureFinished" />
	</div>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import type { App } from "@/api/interface/apps"
import { httpDefaultReloadAPI, httpDefaultStatusAPI, httpDefaultStopAPI } from "@/api/modules/http"
import { AgentEnsureAPI, AgentStatusAPI } from "@/api/modules/agent"
import { websiteDeleteAPI, websiteListAPI } from "@/api/modules/website"
import { ListAppInstalled } from "@/api/modules/apps"
import { useDialog, useMessage } from "naive-ui"
import { onMounted, ref, watch } from "vue"
import HttpConfigFile from "./components/HttpConfigFile.vue"
import Create from "./components/Create.vue"
import AppDeployHistory from "./components/AppDeployHistory.vue"
import AccessLogDrawer from "./components/AccessLogDrawer.vue"
import SecurityDrawer from "./components/SecurityDrawer.vue"
import WebsiteDiagnosticDrawer from "./components/WebsiteDiagnosticDrawer.vue"
import { useAuthStore } from "@/store/auth"
import OpDialog from "@/components/OpDialog.vue"
import { needsWebsiteBindingLookup } from "@/utils/websiteRuntime"
import { createWebsiteTableColumns, resolveWebsiteRowBindingMeta, type WebsiteTableRow } from "./websiteTableColumns"
import { useI18n } from "vue-i18n"
import { websiteDiagnosticMessages } from "./websiteDiagnosticMessages"

function getErrorMessage(error: unknown, fallback: string) {
	if (error && typeof error === "object") {
		const maybe = error as { message?: string }
		if (typeof maybe.message === "string" && maybe.message.trim()) {
			return maybe.message
		}
	}
	return fallback
}

const message = useMessage()
const dialog = useDialog()
const createRef = ref<InstanceType<typeof Create> | null>(null)
const appDeployHistoryRef = ref<InstanceType<typeof AppDeployHistory> | null>(null)
const accessLogDrawerRef = ref<InstanceType<typeof AccessLogDrawer> | null>(null)
const securityDrawerRef = ref<InstanceType<typeof SecurityDrawer> | null>(null)
const diagnosticDrawerRef = ref<InstanceType<typeof WebsiteDiagnosticDrawer> | null>(null)
const { t: diagnosticT } = useI18n({ messages: websiteDiagnosticMessages })
const { t } = useI18n()

const httpServerStatus = ref(false)
const statusStartErrorText = ref("")
const authStore = useAuthStore()
const opDialogRef = ref<InstanceType<typeof OpDialog> | null>(null)
const ensuringAgent = ref(false)
const agentStatus = ref<{ online: boolean; error?: string }>({ online: true })

const fetchAgentStatus = async () => {
	try {
		const res = await AgentStatusAPI()
		agentStatus.value = {
			online: !!res?.data?.online,
			error: res?.data?.error
		}
	} catch (error) {
		agentStatus.value = { online: false, error: getErrorMessage(error, t("website.getAgentStatusFailed")) }
	}
}

const ensureAgent = async () => {
	if (ensuringAgent.value) return
	ensuringAgent.value = true
	try {
		const res = await AgentEnsureAPI()
		const log = res?.data?.log
		const token = authStore.getAuth() || authStore.auth || ""
		if (log) {
			opDialogRef.value?.acceptParams({
				title: t("website.initAgent"),
				sseUrl: `/api/agent/ensure/logs?log=${encodeURIComponent(log)}&token=${encodeURIComponent(token)}`
			})
		}
	} catch (e) {
	} finally {
		ensuringAgent.value = false
	}
}

const handleEnsureFinished = () => {
	fetchAgentStatus()
	if (agentStatus.value.online) {
		httpDefaultStatusAPI()
			.then(res => {
				httpServerStatus.value = res.data.running || res.data.status
				if (res.code !== 0) {
					statusStartErrorText.value = res.msg || t("website.getHttpStatusFailed")
				}
			})
			.catch(err => {
				console.error("获取HTTP服务状态失败", err)
			})
		fetchData()
	}
}

const loading = ref(false)
const tableData = ref<WebsiteTableRow[]>([])
const total = ref(0)
const appInstallMap = ref<Record<number, App.AppInstalledInfo>>({})

const activeTab = ref("list")

function getWebsiteBindingMeta(row: WebsiteTableRow) {
	return resolveWebsiteRowBindingMeta(row, appInstallMap.value, (key, params) => diagnosticT(key, params))
}

const columns = createWebsiteTableColumns({
	appInstallMap: () => appInstallMap.value,
	httpServerRunning: () => httpServerStatus.value,
	diagnosticText: (key, params) => diagnosticT(key, params),
	onAccessLog: row => handleAccessLog(row),
	onErrorLog: row => handleErrorLog(row),
	onSecurity: row => handleSecurity(row),
	onDiagnostic: row => handleDiagnostic(row),
	onUpdate: row => handleUpdate(row),
	onDeploy: row => handleDeploy(row),
	onDelete: row => handleDelete(row)
})

async function fetchData() {
	loading.value = true
	try {
		const res = await websiteListAPI()
		tableData.value = res.data.items || []
		total.value = res.data.total || 0
		if (tableData.value.some(row => needsWebsiteBindingLookup(row))) {
			await fetchBindingMeta()
		} else {
			appInstallMap.value = {}
		}
	} catch (error) {
	} finally {
		loading.value = false
	}
}

async function fetchBindingMeta() {
	try {
		const appsRes = await ListAppInstalled()
		const apps: App.AppInstalledInfo[] = Array.isArray(appsRes.data) ? appsRes.data : []
		const nextAppMap: Record<number, App.AppInstalledInfo> = {}
		for (const item of apps) {
			nextAppMap[item.id] = item
		}
		appInstallMap.value = nextAppMap
	} catch (error) {
		appInstallMap.value = {}
	}
}

async function handleDelete(row: WebsiteTableRow) {
	dialog.info({
		title: t("commons.msg.infoTitle"),
		content: t("website.deleteWebsiteConfirm", { domain: row.primaryDomain }),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			try {
				await websiteDeleteAPI({ id: row.id, deleteApp: false, deleteBackup: false, forceDelete: false })
				message.success(t("commons.msg.deleteSuccess"))
				fetchData()
			} catch (error) {
				console.error(error)
			}
		}
	})
}

function handleAdd() {
	createRef.value?.open()
}

function handleUpdate(row: WebsiteTableRow) {
	const bindingMeta = getWebsiteBindingMeta(row)
	createRef.value?.open(
		{
			...row,
			runtimeBindingSummary: bindingMeta ? `${bindingMeta.source} · ${bindingMeta.detail}` : ""
		},
		"update"
	)
}

function handleDeploy(row: WebsiteTableRow) {
	appDeployHistoryRef.value?.open(row)
}

function handleSecurity(row: Website.WebsiteDTO) {
	securityDrawerRef.value?.open(row)
}

function handleDiagnostic(row: Website.WebsiteDTO) {
	diagnosticDrawerRef.value?.open(row)
}

function handleAccessLog(row: Website.WebsiteDTO) {
	accessLogDrawerRef.value?.open(row, "access")
}

function handleErrorLog(row: Website.WebsiteDTO) {
	accessLogDrawerRef.value?.open(row, "error")
}

function postConfirm() {
	createRef.value?.close()
	fetchData()
}

async function handleReload() {
	try {
		await httpDefaultReloadAPI()
		message.success(t("website.reloadSuccess"))

		let statusData = await httpDefaultStatusAPI()
		console.log(statusData)
		httpServerStatus.value = statusData.data.running || statusData.data.status || false
	} catch (error) {
		console.error(error)
		// message.error("重新加载失败")
	}
}

async function handleStop() {
	try {
		await httpDefaultStopAPI()
		httpServerStatus.value = false
		message.success(t("website.httpServiceStoppedMsg"))
	} catch (error) {
		console.error(error)
		httpDefaultStatusAPI()
			.then(res => {
				httpServerStatus.value = res.data.running || res.data.status
			})
			.catch(err => {
				console.error("获取HTTP服务状态失败", err)
			})
	}
}

watch(activeTab, newVal => {
	if (newVal === "list") {
		loading.value = true
		fetchData()
	}
})

onMounted(() => {
	fetchAgentStatus().then(() => {
		if (!agentStatus.value.online) return
		httpDefaultStatusAPI()
			.then(res => {
				httpServerStatus.value = res.data.running || res.data.status
				if (res.code !== 0) {
					statusStartErrorText.value = res.msg || t("website.getHttpStatusFailed")
				}
			})
			.catch(err => {
				console.error("获取HTTP服务状态失败", err)
			})
		fetchData()
	})
})
</script>

<style scoped>
.proxy-container {
	padding: 20px;
}

.header {
	margin-bottom: 20px;
	display: flex;
	gap: 10px;
}
</style>

<style>
.theme-dark .website-index-root .border-slate-100 {
	border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .website-index-root .bg-slate-50\/70 {
	background-color: color-mix(in srgb, var(--bg-default-color) 70%, transparent) !important;
}
</style>
