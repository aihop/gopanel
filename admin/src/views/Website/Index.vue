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
            {{ httpServerStatus ? "HTTP服务 运行中" : "HTTP服务 未启动" }}
            {{ statusStartErrorText }}
          </div>
          <div class="rounded-full border border-slate-200 bg-slate-50 px-4 py-2 text-sm text-slate-500">
            当前共管理 {{ total }} 个网站
          </div>
        </div>
      </div>
      <div class="flex flex-wrap gap-3 lg:justify-end">
        <n-button
          ghost
          type="primary"
          @click="handleReload"
        >{{ $t("website.restart") }}</n-button>
        <n-button
          ghost
          :disabled="!httpServerStatus"
          @click="handleStop"
        >
          {{ $t("commons.button.stop") }}
        </n-button>
      </div>
    </div>

    <n-alert
      v-if="!agentStatus.online"
      type="warning"
      :show-icon="true"
      title="Agent 未初始化"
      class="rounded-[28px] border border-blue-100/80 bg-base-100 p-6 shadow-[0_18px_48px_rgba(15,23,42,0.08)]"
    >
      <div class="text-sm leading-7 text-slate-600">
        <div>gp-agent 未启动或未安装，网站功能暂不可用。</div>
        <div
          v-if="agentStatus.error"
          class="mt-1 text-slate-500"
        >{{ agentStatus.error }}</div>
        <n-space class="mt-4">
          <n-button
            type="primary"
            :loading="ensuringAgent"
            @click="ensureAgent"
          >一键初始化</n-button>
        </n-space>
      </div>
    </n-alert>
    <div class="rounded-[28px] border border-blue-100/80 bg-base-100 shadow-[0_24px_72px_rgba(15,23,42,0.08)]">
      <div class="flex flex-col gap-5 border-b border-slate-100 px-8 py-7 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-2">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Website Workspace</div>
          <div class="text-2xl font-semibold fg-base-100">
            {{ activeTab === "list" ? $t("website.websiteList") : "配置文件" }}
          </div>
          <div class="text-sm leading-7 text-slate-500">
            {{
							activeTab === "list"
								? "集中管理域名、站点类型与更新时间，快速进入日常维护。"
								: "查看内置 HTTP 服务配置文件，便于排查与调整站点设置。"
						}}
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
          <n-button
            v-if="activeTab === 'list'"
            ghost
            @click="fetchData"
          >刷新列表</n-button>
          <n-button
            v-if="activeTab === 'list'"
            type="primary"
            @click="handleAdd"
          >创建网站</n-button>
        </div>
      </div>

      <div class="p-8">
        <div
          v-if="activeTab === 'list'"
          class="space-y-5"
        >
          <n-data-table
            :loading="loading"
            :columns="columns"
            :data="tableData"
            :bordered="false"
            class="rounded-3xl"
          >
            <template #empty>
              <div class="flex flex-col items-center justify-center py-14">
                <n-empty
                  description="暂无网站"
                  class="mb-3"
                />
                <n-button
                  type="primary"
                  ghost
                  class="mt-6"
                  @click="handleAdd"
                >立即创建</n-button>
              </div>
            </template>
          </n-data-table>
        </div>

        <div
          v-else
          class="overflow-hidden rounded-3xl border border-slate-100 bg-slate-50/70"
        >
          <HttpConfigFile scope-summary="这里编辑的是全局 HTTP 服务配置，作用于代理层本身，不对应某一个网站的应用运行时。具体网站绑定的是 Docker/Podman、rootless/rootful 与运行用户，请在网站列表、部署管理或安全/日志入口查看。" />
        </div>
      </div>
    </div>

    <Create
      ref="createRef"
      @confirm="postConfirm"
    />

    <AppDeployHistory
      ref="appDeployHistoryRef"
      @confirm="postConfirm"
    />

    <AccessLogDrawer ref="accessLogDrawerRef" />

    <SecurityDrawer
      ref="securityDrawerRef"
      @confirm="fetchData"
    />

    <OpDialog
      ref="opDialogRef"
      @search="handleEnsureFinished"
    />
  </div>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import type { Pipeline } from "@/api/interface/pipeline"
import type { App } from "@/api/interface/apps"
import type { DataTableColumns } from "naive-ui"
import { httpDefaultReloadAPI, httpDefaultStatusAPI, httpDefaultStopAPI } from "@/api/modules/http"
import { AgentEnsureAPI, AgentStatusAPI } from "@/api/modules/agent"
import { websiteDeleteAPI, websiteListAPI } from "@/api/modules/website"
import { ListAppInstalled } from "@/api/modules/apps"
import { NButton, NSpace, NTag, useDialog, useMessage, NAlert } from "naive-ui"
import { h, onMounted, ref, watch } from "vue"
import HttpConfigFile from "./components/HttpConfigFile.vue"
import Create from "./components/Create.vue"
import AppDeployHistory from "./components/AppDeployHistory.vue"
import AccessLogDrawer from "./components/AccessLogDrawer.vue"
import SecurityDrawer from "./components/SecurityDrawer.vue"
import { formatTime } from "@/utils/date"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"
import OpDialog from "@/components/OpDialog.vue"
import { buildRuntimeDetailText } from "@/utils/runtime"
import { listAllPipelines } from "@/utils/pipeline"
import { getWebsiteSourceLabel, isHttpsWebsiteProtocol, needsWebsiteBindingLookup, normalizeWebsiteProtocol, resolveWebsiteBindingMeta } from "@/utils/websiteRuntime"

type WebsiteTableRow = Website.WebsiteDTO & {
	domains?: Array<string | { domain?: string }>
	status?: string | boolean
}

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
		agentStatus.value = { online: false, error: getErrorMessage(error, "获取 Agent 状态失败") }
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
				title: "初始化 Agent",
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
					statusStartErrorText.value = res.msg || "获取HTTP服务状态失败"
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
const pipelineMap = ref<Record<number, Pipeline.ResPipeline>>({})

const activeTab = ref("list")

function normalizeDomainList(row: WebsiteTableRow) {
	if (Array.isArray(row.domains)) {
		const domains = row.domains as Array<string | { domain?: string }>
		return domains.map((item) => (typeof item === "string" ? item : item?.domain)).filter(Boolean) as string[]
	}
	if (typeof row.otherDomains === "string") {
		return row.otherDomains
			.split(",")
			.map((item: string) => item.trim())
			.filter(Boolean)
	}
	return []
}

function getWebsiteTypeLabel(type: string) {
	if (type === "proxy") return "反向代理"
	if (type === "web_app") return "容器化应用"
	return "静态网站"
}

function getWebsiteTypeTag(type: string) {
	if (type === "proxy") return "info"
	if (type === "web_app") return "warning"
	return "success"
}

function formatWebsiteStatus(status: unknown) {
	if (typeof status === "boolean") {
		return status ? "运行中" : "已停止"
	}
	if (typeof status === "string") {
		const normalized = status.trim().toLowerCase()
		if (["running", "enable", "enabled", "online", "success", "active"].includes(normalized)) {
			return "运行中"
		}
		if (["stopped", "stop", "disabled", "disable", "offline", "inactive"].includes(normalized)) {
			return "已停止"
		}
		return status
	}
	return httpServerStatus.value ? "已接入" : "待检查"
}

function getStatusTagType(status: unknown): "success" | "warning" | "default" {
	const text = formatWebsiteStatus(status)
	if (text === "运行中" || text === "已接入") {
		return "success"
	}
	if (text === "已停止" || text === "待检查") {
		return "warning"
	}
	return "default"
}

function getSecuritySummary(row: WebsiteTableRow) {
	const tags: string[] = []
	if (row.antiCrawler) tags.push("防爬虫")
	if (row.antiLeech) tags.push("防盗链")
	if (row.rateLimitMode === "normal") tags.push("常规限流")
	if (row.rateLimitMode === "strict") tags.push("严格限流")
	if (row.wafEnable) tags.push("轻量 WAF")
	if (row.blockSensitive) tags.push("敏感保护")
	return tags
}

function getWebsiteBindingMeta(row: WebsiteTableRow) {
	return resolveWebsiteBindingMeta(row, {
		appInstallMap: appInstallMap.value,
		pipelineMap: pipelineMap.value
	}, {
		includeSourceInDetail: false,
		kindFallback: "Runtime",
		userFallback: "镜像默认",
		runtimePrefix: "",
		runUserPrefix: "用户:"
	})
}

function getWebsiteDeploySummary(row: WebsiteTableRow) {
	const lines: string[] = []
	if (row.activeRelease?.releaseId) {
		lines.push(`正式版本 v${row.activeRelease.version}`)
	} else {
		lines.push("正式版本 未上线")
	}
	if (row.latestPipelineSync?.pipelineRecordId) {
		lines.push(`${row.latestPipelineSync.isActive ? "构建同步 当前生效" : "构建同步"} #${row.latestPipelineSync.pipelineRecordId}`)
	}
	return lines
}

const columns: DataTableColumns<WebsiteTableRow> = [
	{
		title: t("website.primaryDomain"),
		key: "primaryDomain",
		render(row) {
			const protocol = normalizeWebsiteProtocol(row.protocol) || "HTTP"
			return h("div", { class: "flex flex-col space-y-1" }, [
				h("a", { href: protocol === "HTTP" ? `http://${row.primaryDomain}` : `https://${row.primaryDomain}`, target: "_blank", class: "text-base font-semibold fg-base-100" }, row.primaryDomain),
				h(
					"div",
					{ class: "flex flex-wrap gap-2 pt-1" },
					[
						h(
							NTag,
							{
								size: "small",
								round: true,
								bordered: false,
								type: isHttpsWebsiteProtocol(row) ? "success" : "default"
							},
							{ default: () => protocol }
						),
						row.defaultServer
							? h(
									NTag,
									{
										size: "small",
										round: true,
										bordered: false,
										type: "warning"
									},
									{ default: () => "默认站点" }
								)
							: null
					].filter(Boolean)
				)
			])
		}
	},
	{
		title: "子域名",
		key: "otherDomains",
		render(row) {
			const domains = normalizeDomainList(row)
			return h(
				"div",
				{ class: "flex flex-wrap gap-2" },
				domains.length
					? domains.map((item: string) =>
							h(
								NTag,
								{
									size: "small",
									round: true,
									bordered: false
								},
								{ default: () => item }
							)
						)
					: [h("span", { class: "text-sm text-slate-400" }, "无附加域名")]
			)
		}
	},
	{
		title: "类型",
		key: "type",
		render(row) {
			const tags = [
				h(
					NTag,
					{
						round: true,
						bordered: false,
						type: getWebsiteTypeTag(row.type)
					},
					{
						default: () => getWebsiteTypeLabel(row.type)
					}
				)
			]

			if (row.type === "web_app" && row.codeSource) {
				const sourceText = getWebsiteSourceLabel(row.codeSource)
				tags.push(
					h(
						NTag,
						{ type: "default", size: "small", bordered: false, style: { marginLeft: '4px' } },
						{ default: () => sourceText }
					)
				)
			}

			const bindingMeta = getWebsiteBindingMeta(row)
			const deploySummary = getWebsiteDeploySummary(row)
			return h("div", { class: "flex flex-col gap-2" }, [
				h('div', { class: 'flex items-center flex-wrap gap-1' }, tags),
				bindingMeta
					? h("div", { class: "text-xs text-slate-500" }, `${bindingMeta.source} · ${bindingMeta.detail}`)
					: null,
				h(
					"div",
					{ class: "flex flex-wrap gap-2 text-xs text-slate-400" },
					deploySummary.map((item) => h("span", item))
				)
			])
		}
	},
	{
		title: "状态",
		key: "status",
		render(row) {
			return h(
				NTag,
				{
					round: true,
					bordered: false,
					type: getStatusTagType(row.status)
				},
				{
					default: () => formatWebsiteStatus(row.status)
				}
			)
		}
	},
	{
		title: "安全防护",
		key: "security",
		render(row) {
			const tags = getSecuritySummary(row)
			return h(
				"div",
				{ class: "flex flex-wrap gap-2" },
				tags.length
					? tags.slice(0, 3).map((item: string) =>
							h(
								NTag,
								{
									size: "small",
									round: true,
									bordered: false,
									type: "success",
								},
								{ default: () => item }
							)
						)
					: [h("span", { class: "text-sm text-slate-400" }, "未启用")]
			)
		},
	},
	{
		title: t("commons.table.createdAt"),
		key: "updatedAt",
		render(row) {
			return h("span", { class: "text-sm text-slate-500" }, formatTime(row.updatedAt))
		}
	},
	{
		title: t("commons.table.operate"),
		key: "actions",
		render(row) {
			const buttons = [
				h(
					NButton,
					{
						text: true,
						type: "info",
						onClick: () => handleAccessLog(row)
					},
					{ default: () => "访问记录" }
				),
				h(
					NButton,
					{
						text: true,
						type: "error",
						onClick: () => handleErrorLog(row)
					},
					{ default: () => "错误日志" }
				),
				h(
					NButton,
					{
						text: true,
						type: "warning",
						onClick: () => handleSecurity(row)
					},
					{ default: () => "安全防护" }
				),
				h(
					NButton,
					{
						text: true,
						type: "primary",
						onClick: () => handleUpdate(row)
					},
					{ default: () => t("commons.button.set") }
				)
			]

			if (row.type === "web_app" || row.type === "static") {
				buttons.splice(1, 0, h(
					NButton,
					{
						text: true,
						type: "success",
						onClick: () => handleDeploy(row)
					},
					{ default: () => "部署管理" }
				))
			}

			buttons.push(
				h(
					NButton,
					{
						text: true,
						type: "error",
						onClick: () => handleDelete(row)
					},
					{ default: () => "删除" }
				)
			)

			return h(NSpace, { size: 8 }, { default: () => buttons })
		}
	}
]

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
			pipelineMap.value = {}
		}
	} catch (error) {
	} finally {
		loading.value = false
	}
}

async function fetchBindingMeta() {
	try {
		const [appsRes, pipelines] = await Promise.all([
			ListAppInstalled(),
			listAllPipelines()
		])
		const apps: App.AppInstalledInfo[] = Array.isArray(appsRes.data) ? appsRes.data : []
		const nextAppMap: Record<number, App.AppInstalledInfo> = {}
		for (const item of apps) {
			nextAppMap[item.id] = item
		}
		appInstallMap.value = nextAppMap
		const nextPipelineMap: Record<number, Pipeline.ResPipeline> = {}
		for (const item of pipelines) {
			nextPipelineMap[item.id] = item
		}
		pipelineMap.value = nextPipelineMap
	} catch (error) {
		appInstallMap.value = {}
		pipelineMap.value = {}
	}
}

async function handleDelete(row: WebsiteTableRow) {
	dialog.info({
		title: "提示",
		content: `确定要删除 ${row.primaryDomain} 吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await websiteDeleteAPI({ id: row.id, deleteApp: false, deleteBackup: false, forceDelete: false })
				message.success("删除成功")
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
	createRef.value?.open({
		...row,
		runtimeBindingSummary: bindingMeta ? `${bindingMeta.source} · ${bindingMeta.detail}` : ""
	}, "update")
}

function handleDeploy(row: WebsiteTableRow) {
	appDeployHistoryRef.value?.open(row)
}

function handleSecurity(row: Website.WebsiteDTO) {
	securityDrawerRef.value?.open(row)
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
		message.success("重新加载成功")

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
		message.success("HTTP服务 已停止")
	} catch (error) {
		console.error(error)
		httpDefaultStatusAPI()
			.then(res => {
				httpServerStatus.value = res.data.running || res.data.status
			})
			.catch(err => {
				console.error("获取HTTP服务状态失败", err)
			})
		message.error("停止失败")
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
					statusStartErrorText.value = res.msg || "获取HTTP服务状态失败"
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
