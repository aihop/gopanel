<template>
  <div class="mt-4 space-y-8">

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
          </div>
          <div class="rounded-full border border-slate-200 bg-slate-50 px-4 py-2 text-sm text-slate-500">
            当前共管理 {{ siteSummary.total }} 个网站
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
          <HttpConfigFile />
        </div>
      </div>
    </div>

    <Create
      ref="createRef"
      @confirm="postConfirm"
    />

    <DeployHistory
      ref="deployHistoryRef"
      @confirm="postConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import type { DataTableColumns } from "naive-ui"
import { httpDefaultReloadAPI, httpDefaultStatusAPI, httpDefaultStopAPI } from "@/api/modules/http"
import { DeleteWebsiteAPI, ListWebsitesAPI } from "@/api/modules/website"
import { NButton, NSpace, NTag, useDialog, useMessage } from "naive-ui"
import { computed, h, onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"
import HttpConfigFile from "./components/HttpConfigFile.vue"
import Create from "./components/Create.vue"
import DeployHistory from "./components/DeployHistory.vue"
import { formatTime } from "@/utils/date"
import { t } from "@/i18n"

const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const createRef = ref<InstanceType<typeof Create> | null>(null)
const deployHistoryRef = ref<any>(null)

const httpServerStatus = ref(false)

const loading = ref(false)
const tableData = ref<Website.WebsiteDTO[]>([])

const activeTab = ref("list")

const siteSummary = computed(() => {
	const rows = tableData.value || []
	return {
		total: rows.length,
		static: rows.filter((item: Website.WebsiteDTO) => item.type !== "proxy").length,
		proxy: rows.filter((item: Website.WebsiteDTO) => item.type === "proxy").length,
		defaultServer: rows.filter((item: Website.WebsiteDTO) => item.defaultServer).length
	}
})

const featuredPrimaryDomain = computed(() => tableData.value[0]?.primaryDomain || "暂无网站")

function normalizeDomainList(row: Website.WebsiteDTO) {
	if (Array.isArray(row.domains)) {
		return row.domains.map((item: any) => (typeof item === "string" ? item : item?.domain)).filter(Boolean)
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

const columns: DataTableColumns<any> = [
	{
		title: t("website.primaryDomain"),
		key: "primaryDomain",
		render(row) {
			return h("div", { class: "flex flex-col space-y-1" }, [
				h("a", { href: row.protocol === "HTTP" ? `http://${row.primaryDomain}` : `https://${row.primaryDomain}`, target: "_blank", class: "text-base font-semibold fg-base-100" }, row.primaryDomain),
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
								type: row.protocol === "https" ? "success" : "default"
							},
							{ default: () => (row.protocol || "http").toUpperCase() }
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
				const sourceMap: Record<string, string> = {
					'git': '自定义镜像',
					'pipeline': '流水线',
					'app_store': '应用商店',
					'upload': '代码上传'
				}
				const sourceText = sourceMap[row.codeSource] || row.codeSource
				tags.push(
					h(
						NTag,
						{ type: "default", size: "small", bordered: false, style: { marginLeft: '4px' } },
						{ default: () => sourceText }
					)
				)
			}

			return h('div', { class: 'flex items-center' }, tags)
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
		const res = await ListWebsitesAPI()
		tableData.value = res.data || []
	} catch (error: any) {
	} finally {
		loading.value = false
	}
}

async function handleDelete(row: any) {
	dialog.info({
		title: "提示",
		content: `确定要删除 ${row.primaryDomain} 吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await DeleteWebsiteAPI({ id: row.id, deleteApp: false, deleteBackup: false, forceDelete: false })
				message.success("删除成功")
				fetchData()
			} catch (error) {
				console.error(error)
				message.error("删除失败, 请尝试从配置文件删除")
			}
		}
	})
}

function handleAdd() {
	createRef.value?.open()
}

function handleOpenConfig() {
	activeTab.value = "config"
}

function handleUpdate(row: any) {
	createRef.value?.open(row, "update")
}

function handleDeploy(row: any) {
	deployHistoryRef.value?.open(row)
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
	httpDefaultStatusAPI()
		.then(res => {
			httpServerStatus.value = res.data.running || res.data.status
		})
		.catch(err => {
			console.error("获取HTTP服务状态失败", err)
		})
	fetchData()
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
