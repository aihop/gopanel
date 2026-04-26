<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <n-drawer
    v-model:show="visible"
    :width="880"
    placement="right"
  >
    <n-drawer-content closable>
      <template #header>
        <div class="flex items-center gap-3">
          <div class="text-base font-semibold">{{ drawerTitle }}</div>
          <n-tag
            v-if="website?.primaryDomain"
            round
            :bordered="false"
            type="primary"
          >
            {{ website.primaryDomain }}
          </n-tag>
        </div>
      </template>

      <div class="space-y-5">
        <div class="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-slate-700">{{ logLabel }}</div>
              <div class="mt-1 text-xs text-slate-500">
                {{ simpleHelperText }}
              </div>
              <div
                v-if="bindingRuntimeText"
                class="mt-2 text-xs text-slate-500"
              >
                {{ bindingRuntimeText }}
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              <n-button
                v-if="!isErrorView"
                size="small"
                ghost
                @click="openTodayIPStats"
              >
                最近一天 IP
              </n-button>
              <n-button
                size="small"
                @click="loadLogs(false)"
              >
                刷新
              </n-button>
              <n-button
                size="small"
                type="primary"
                ghost
                @click="loadLogs(true)"
              >
                定位最新
              </n-button>
            </div>
          </div>
        </div>

        <div
          v-if="showStructuredList"
          class="rounded-2xl border border-slate-200 bg-white"
        >
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
            <div class="flex items-center gap-2">
              <div class="text-xs font-medium uppercase tracking-[0.16em] text-slate-500">{{ logPanelTitle }}</div>
            </div>
            <div class="flex items-center gap-2">
              <n-button
                size="small"
                quaternary
                @click="showFilters = !showFilters"
              >
                {{ showFilters ? "收起筛选" : "筛选" }}
              </n-button>
              <n-button
                v-if="canToggleView"
                size="small"
                quaternary
                @click="rawMode = true"
              >
                原始日志
              </n-button>
              <div class="text-xs text-slate-400">
                {{ loading ? "加载中..." : `第 ${page} / ${Math.max(total, 1)} 页` }}
              </div>
            </div>
          </div>

          <div
            v-if="showFilters"
            class="border-b border-slate-100 bg-slate-50/70 px-4 py-3"
          >
            <div class="flex flex-wrap gap-2">
              <n-button
                v-for="item in statusFilters"
                :key="item.value"
                size="small"
                :type="statusFilter === item.value ? 'primary' : 'default'"
                :ghost="statusFilter !== item.value"
                @click="statusFilter = item.value"
              >
                {{ item.label }}
              </n-button>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <n-input
                v-model:value="searchKeyword"
                clearable
                size="small"
                placeholder="搜索 IP、页面、状态码"
                class="w-full max-w-xs"
              />
              <n-button
                size="small"
                ghost
                @click="copyPath"
              >
                复制日志路径
              </n-button>
            </div>
            <div class="mt-3 text-xs text-slate-400 break-all">
              {{ logPath || "日志路径将在首次读取后显示" }}
            </div>
          </div>

          <div class="max-h-[65vh] overflow-auto">
            <div
              v-if="loading"
              class="px-4 py-6 text-sm text-slate-500"
            >
              正在读取访问日志...
            </div>
            <div
              v-else-if="filteredEntries.length"
              class="divide-y divide-slate-100"
            >
              <div
                v-for="(item, index) in filteredEntries"
                :key="`${index}-${item.raw}`"
                class="cursor-pointer px-4 py-3 transition-colors hover:bg-slate-50"
                :class="selectedLogRaw === item.raw && detailVisible ? 'bg-slate-50' : ''"
                @click="openDetail(item)"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="min-w-0 flex-1">
                    <div class="flex items-start gap-3">
                      <div class="shrink-0 rounded-lg bg-slate-100 px-2.5 py-1 text-sm font-semibold tabular-nums text-slate-700">
                        {{ item.ip }}
                      </div>
                      <div class="min-w-0 flex-1">
                        <div
                          class="truncate text-sm font-semibold text-slate-900"
                          :title="item.path"
                        >
                          {{ item.path }}
                        </div>
                        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                          <span class="tabular-nums">{{ item.timeText }}</span>
                          <span>{{ item.method }}</span>
                          <span>{{ item.statusText }}</span>
                          <span>{{ item.durationText }}</span>
                          <template v-if="isErrorView">
                            <span v-if="item.host">{{ item.host }}</span>
                            <span v-if="item.userAgent">{{ item.userAgent }}</span>
                          </template>
                        </div>
                      </div>
                    </div>
                  </div>
                  <n-tag
                    round
                    size="small"
                    :bordered="false"
                    :type="getStatusTagType(item.status)"
                  >
                    {{ item.statusText }}
                  </n-tag>
                </div>
              </div>
            </div>
            <div
              v-else-if="parsedEntries.length"
              class="px-4 py-6 text-sm text-slate-500"
            >
              当前筛选条件下暂无匹配记录
            </div>
            <div
              v-else
              class="px-4 py-6 text-sm text-slate-500"
            >
              {{ emptyText }}
            </div>
          </div>
        </div>

        <div
          v-else
          class="rounded-2xl border border-slate-200 bg-black"
        >
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
            <div class="text-xs font-medium uppercase tracking-[0.16em] text-emerald-400">{{ logPanelTitle }}</div>
            <div class="flex items-center gap-2">
              <n-button
                v-if="canToggleView"
                size="small"
                quaternary
                @click="rawMode = false"
              >
                简约视图
              </n-button>
              <div class="text-xs text-slate-400">
                {{ loading ? "加载中..." : `第 ${page} / ${Math.max(total, 1)} 页` }}
              </div>
            </div>
          </div>
          <div class="max-h-[65vh] overflow-auto px-4 py-4 font-mono text-xs leading-6 text-emerald-300 whitespace-pre-wrap">
            <div v-if="loading">正在读取访问日志...</div>
            <div v-else-if="logContent">{{ logContent }}</div>
            <div
              v-else
              class="text-slate-500"
            >
              {{ emptyText }}
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between gap-3">
          <div class="text-xs text-slate-500">
            {{ end ? "已经到达日志末尾" : "可继续翻页查看更早记录" }}
          </div>
          <div class="flex gap-2">
            <n-button
              size="small"
              :disabled="page <= 1 || loading"
              @click="changePage(page - 1)"
            >
              上一页
            </n-button>
            <n-button
              size="small"
              :disabled="page >= total || loading"
              @click="changePage(page + 1)"
            >
              下一页
            </n-button>
          </div>
        </div>
      </div>

      <n-modal
        v-model:show="detailVisible"
        preset="card"
        :style="{ width: 'min(820px, calc(100vw - 32px))' }"
        :title="detailTitle"
        size="small"
      >
        <template v-if="activeEntry">
          <div class="space-y-4">
            <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
              <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
                关键信息
              </div>
              <div class="grid gap-3 text-sm text-slate-600 md:grid-cols-2">
                <div class="rounded-lg bg-white px-3 py-3">
                  <div class="text-xs text-slate-400">来源 IP</div>
                  <div class="mt-1 font-medium">{{ activeEntry.ip }}</div>
                </div>
                <div class="rounded-lg bg-white px-3 py-3">
                  <div class="text-xs text-slate-400">状态码</div>
                  <div class="mt-1 font-medium">{{ activeEntry.statusText }}</div>
                </div>
                <div class="rounded-lg bg-white px-3 py-3">
                  <div class="text-xs text-slate-400">请求方法</div>
                  <div class="mt-1 font-medium">{{ activeEntry.method }}</div>
                </div>
                <div class="rounded-lg bg-white px-3 py-3">
                  <div class="text-xs text-slate-400">请求时间</div>
                  <div class="mt-1 font-medium">{{ activeEntry.timeText }}</div>
                </div>
                <div class="rounded-lg bg-white px-3 py-3">
                  <div class="text-xs text-slate-400">耗时</div>
                  <div class="mt-1 font-medium">{{ activeEntry.durationText }}</div>
                </div>
                <div
                  v-if="activeEntry.sizeText"
                  class="rounded-lg bg-white px-3 py-3"
                >
                  <div class="text-xs text-slate-400">响应大小</div>
                  <div class="mt-1 font-medium">{{ activeEntry.sizeText }}</div>
                </div>
                <div class="rounded-lg bg-white px-3 py-3 md:col-span-2">
                  <div class="text-xs text-slate-400">访问页面</div>
                  <div class="mt-1 font-medium break-all">{{ activeEntry.path }}</div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-slate-200 bg-white px-4 py-4">
              <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
                补充信息
              </div>
              <div class="grid gap-3 text-sm text-slate-600 md:grid-cols-2">
                <div
                  v-if="activeEntry.host"
                  class="rounded-lg bg-slate-50 px-3 py-3"
                >
                  <div class="text-xs text-slate-400">域名</div>
                  <div class="mt-1 font-medium">{{ activeEntry.host }}</div>
                </div>
                <div class="rounded-lg bg-slate-50 px-3 py-3">
                  <div class="text-xs text-slate-400">解析状态</div>
                  <div class="mt-1 font-medium">{{ activeEntry.parsed ? "已结构化" : "原始日志行" }}</div>
                </div>
                <div
                  v-if="activeEntry.referer"
                  class="rounded-lg bg-slate-50 px-3 py-3 md:col-span-2"
                >
                  <div class="text-xs text-slate-400">Referer</div>
                  <div class="mt-1 font-medium break-all">{{ activeEntry.referer }}</div>
                </div>
                <div
                  v-if="activeEntry.userAgentFull"
                  class="rounded-lg bg-slate-50 px-3 py-3 md:col-span-2"
                >
                  <div class="text-xs text-slate-400">User-Agent</div>
                  <div class="mt-1 font-medium break-all">{{ activeEntry.userAgentFull }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="mt-4 rounded-xl border border-slate-200 bg-slate-900 px-4 py-4">
            <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-400">
              格式化数据
            </div>
            <div class="font-mono text-[11px] leading-5 text-slate-200 break-all whitespace-pre-wrap">
              {{ activeEntry.formattedRaw }}
            </div>
          </div>
        </template>
      </n-modal>

      <n-modal
        v-model:show="todayStatsVisible"
        preset="card"
        :style="{ width: 'min(720px, calc(100vw - 32px))' }"
        title="最近一天 IP 统计"
        size="small"
      >
        <div
          v-if="todayStatsLoading"
          class="py-10 text-center text-sm text-slate-500"
        >
          正在统计最近一天的访问 IP...
        </div>
        <template v-else-if="todayStats">
          <div class="space-y-4">
            <div class="grid gap-3 md:grid-cols-3">
              <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
                <div class="text-xs text-slate-400">统计日期</div>
                <div class="mt-2 text-lg font-semibold text-slate-800">{{ todayStats.date }}</div>
              </div>
              <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
                <div class="text-xs text-slate-400">独立 IP</div>
                <div class="mt-2 text-lg font-semibold text-slate-800">{{ todayStats.uniqueIpCount }}</div>
              </div>
              <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
                <div class="text-xs text-slate-400">请求数</div>
                <div class="mt-2 text-lg font-semibold text-slate-800">{{ todayStats.requestCount }}</div>
              </div>
            </div>

            <div class="rounded-xl border border-slate-200 bg-white px-4 py-4">
              <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
                Top IP
              </div>
              <div
                v-if="todayStats.topIps.length"
                class="space-y-2"
              >
                <div
                  v-for="item in todayStats.topIps"
                  :key="item.ip"
                  class="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-3 text-sm text-slate-700"
                >
                  <div class="font-medium tabular-nums">{{ item.ip }}</div>
                  <div class="text-slate-500">{{ item.count }} 次</div>
                </div>
              </div>
              <div
                v-else
                class="text-sm text-slate-500"
              >
                最近一天暂无访问记录
              </div>
            </div>

            <div class="text-xs text-slate-400 break-all">
              {{ todayStats.path }}
            </div>
          </div>
        </template>
      </n-modal>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import { WebsiteLogAPI, WebsiteTodayIPStatsAPI } from "@/api/modules/website"
import { ListAppInstalled } from "@/api/modules/apps"
import { getPipelinePage } from "@/api/modules/pipeline"
import { copyText } from "@/utils/util"
import { NButton, NDrawer, NDrawerContent, NInput, NModal, NTag, useMessage } from "naive-ui"
import { computed, ref } from "vue"

type WebsiteLogType = "access" | "error"
type StatusFilter = "all" | "2xx" | "3xx" | "4xx" | "5xx"

const visible = ref(false)
const loading = ref(false)
const website = ref<Website.WebsiteDTO | null>(null)
const logType = ref<WebsiteLogType>("access")
const rawMode = ref(false)
const searchKeyword = ref("")
const statusFilter = ref<StatusFilter>("all")
const detailVisible = ref(false)
const todayStatsVisible = ref(false)
const todayStatsLoading = ref(false)
const showFilters = ref(false)
const selectedLogRaw = ref("")
const page = ref(1)
const limit = ref(200)
const total = ref(1)
const end = ref(true)
const logContent = ref("")
const logLines = ref<string[]>([])
const logPath = ref("")
const todayStats = ref<Website.WebSiteTodayIPStats | null>(null)
const appInstallMap = ref<Record<number, any>>({})
const pipelineMap = ref<Record<number, any>>({})

const message = useMessage()

type ParsedLogEntry = {
	raw: string;
	parsed: boolean;
	formattedRaw: string;
	timeText: string;
	method: string;
	path: string;
	status?: number;
	statusText: string;
	ip: string;
	host: string;
	durationText: string;
	sizeText: string;
	userAgent: string;
	userAgentFull: string;
	referer: string;
}

type ExtractedLogPayload = {
	timestamp: string;
	jsonText: string;
}

async function loadLogs(latest = false) {
	if (!website.value) return
	loading.value = true
	try {
		const targetPage = latest ? 1 : page.value
		const res = await WebsiteLogAPI({
			websiteId: website.value.id,
			page: targetPage,
			limit: limit.value,
			latest,
			logType: logType.value,
		})
		logContent.value = res.data?.content || ""
		logLines.value = res.data?.lines || (logContent.value ? logContent.value.split("\n").filter(Boolean) : [])
		logPath.value = res.data?.path || getDefaultLogPath()
		total.value = res.data?.total || 1
		end.value = !!res.data?.end
		page.value = latest ? Math.max(total.value, 1) : targetPage
	} catch (error: any) {
		message.error(error?.message || `读取${drawerTitle.value}失败`)
	} finally {
		loading.value = false
	}
}

function getDefaultLogPath() {
	if (!website.value) return ""
	return logType.value === "error" ? website.value.errorLogPath : website.value.accessLogPath
}

async function copyPath() {
	const path = logPath.value || getDefaultLogPath()
	if (!path) {
		message.warning("暂无日志路径")
		return
	}
	await copyText(path)
}

async function openTodayIPStats() {
	if (!website.value) return
	todayStatsVisible.value = true
	todayStatsLoading.value = true
	try {
		const res = await WebsiteTodayIPStatsAPI({
			websiteId: website.value.id,
		})
		todayStats.value = res.data || null
	} catch (error: any) {
		message.error(error?.message || "读取今日 IP 统计失败")
	} finally {
		todayStatsLoading.value = false
	}
}

const drawerTitle = computed(() => (logType.value === "error" ? "网站错误日志" : "网站访问记录"))
const logLabel = computed(() => (logType.value === "error" ? "错误日志" : "访问日志"))
const emptyText = computed(() => (logType.value === "error" ? "暂无错误日志。若网站当前没有报错，这里会暂时为空。" : "暂无访问记录。若网站刚创建或还没有请求进入，这里会暂时为空。"))
const logPanelTitle = computed(() => (logType.value === "error" ? "Error Log" : "Access Log"))
const simpleHelperText = computed(() => (logType.value === "error"
	? "这里按条展示网站异常请求，默认只突出最关键的错误信息。"
	: "这里按条展示网站访问记录，默认只保留最核心的访问信息。"))
const parsedEntries = computed(() => logLines.value.map(parseLogLine).filter((item): item is ParsedLogEntry => !!item))
const filteredEntries = computed(() => parsedEntries.value.filter(matchStatusFilter).filter(matchSearchKeyword))
const activeEntry = computed(() => filteredEntries.value.find(item => item.raw === selectedLogRaw.value) || null)
const canToggleView = computed(() => logLines.value.length > 0)
const showStructuredList = computed(() => logLines.value.length > 0 && !rawMode.value)
const isErrorView = computed(() => logType.value === "error")
const bindingRuntimeText = computed(() => {
	if (!website.value) return ""
	if (website.value.codeSource === "app_store" && website.value.appInstallId) {
		const item = appInstallMap.value[website.value.appInstallId]
		if (!item) return `绑定目标：应用商店应用 #${website.value.appInstallId}`
		return buildBindingText("应用商店", item.name, item)
	}
	if (website.value.codeSource === "pipeline" && website.value.pipelineId) {
		const item = pipelineMap.value[website.value.pipelineId]
		if (!item) return `绑定目标：流水线 #${website.value.pipelineId}`
		return buildBindingText("流水线", item.name, item)
	}
	return ""
})
const detailTitle = computed(() => {
	if (!activeEntry.value) return "日志详情"
	return `${activeEntry.value.method} ${activeEntry.value.path}`
})
const statusFilters: Array<{ label: string; value: StatusFilter }> = [
	{ label: "全部", value: "all" },
	{ label: "2xx", value: "2xx" },
	{ label: "3xx", value: "3xx" },
	{ label: "4xx", value: "4xx" },
	{ label: "5xx", value: "5xx" },
]

function open(row: Website.WebsiteDTO, type: WebsiteLogType = "access") {
	website.value = row
	logType.value = type
	rawMode.value = false
	searchKeyword.value = ""
	statusFilter.value = "all"
	showFilters.value = false
	selectedLogRaw.value = ""
	detailVisible.value = false
	todayStatsVisible.value = false
	todayStats.value = null
	visible.value = true
	page.value = 1
	total.value = 1
	end.value = true
	logContent.value = ""
	logLines.value = []
	logPath.value = type === "error" ? "" : (row.accessLogPath || "")
	loadBindingMeta()
	loadLogs(true)
}

async function loadBindingMeta() {
	if (!website.value) return
	try {
		if (website.value.appInstallId) {
			const res = await ListAppInstalled()
			const list = Array.isArray(res.data) ? res.data : []
			const nextMap: Record<number, any> = {}
			for (const item of list) nextMap[item.id] = item
			appInstallMap.value = nextMap
		}
		if (website.value.pipelineId) {
			const res = await getPipelinePage({ page: 1, limit: 200 })
			const list = Array.isArray(res.data?.items) ? res.data.items : []
			const nextMap: Record<number, any> = {}
			for (const item of list) nextMap[item.id] = item
			pipelineMap.value = nextMap
		}
	} catch (error) {
		appInstallMap.value = {}
		pipelineMap.value = {}
	}
}

function getRuntimeKindLabel(item: any) {
	const kind = String(item?.runtimeKind || "").toLowerCase()
	if (kind === "podman") return "Podman"
	if (kind === "docker") return "Docker"
	return "Runtime"
}

function getRuntimeModeLabel(item: any) {
	switch (String(item?.runtimeMode || "").toLowerCase()) {
		case "rootless":
			return "rootless"
		case "rootful":
			return "rootful"
		default:
			return "default"
	}
}

function getRunUserLabel(item: any) {
	return item?.runUser || "镜像默认"
}

function buildBindingText(source: string, name: string, item: any) {
	return `绑定目标：${source} · ${name} · ${getRuntimeKindLabel(item)}/${getRuntimeModeLabel(item)} · 运行用户：${getRunUserLabel(item)}`
}

function changePage(nextPage: number) {
	page.value = nextPage
	loadLogs(false)
}

defineExpose({ open })

function openDetail(item: ParsedLogEntry) {
	selectedLogRaw.value = item.raw
	detailVisible.value = true
}

function parseLogLine(line: string): ParsedLogEntry | null {
	const raw = line.trim()
	if (!raw) return null
	const extracted = extractLogPayload(raw)
	try {
		const payload = JSON.parse(extracted?.jsonText || raw)
		const request = payload?.request || {}
		const headers = request?.headers || {}
		const status = readNumber(payload?.status)
		return {
			raw,
			parsed: true,
			formattedRaw: formatStructuredRaw(payload),
			timeText: formatLogTime(payload?.ts || extracted?.timestamp),
			method: readString(request?.method) || "-",
			path: readString(request?.uri) || readString(request?.path) || "/",
			status,
			statusText: status ? String(status) : "--",
			ip: readString(request?.remote_ip) || readString(request?.client_ip) || "-",
			host: readString(request?.host) || getHeaderValue(headers, "Host"),
			durationText: formatDuration(payload?.duration),
			sizeText: formatBytes(payload?.size),
			userAgent: simplifyUserAgent(getHeaderValue(headers, "User-Agent")),
			userAgentFull: getHeaderValue(headers, "User-Agent"),
			referer: getHeaderValue(headers, "Referer"),
		}
	} catch {
		return {
			raw,
			parsed: false,
			formattedRaw: raw,
			timeText: formatLogTime(extracted?.timestamp),
			method: "LOG",
			path: raw,
			status: undefined,
			statusText: "--",
			ip: "-",
			host: "",
			durationText: "--",
			sizeText: "",
			userAgent: "",
			userAgentFull: "",
			referer: "",
		}
	}
}

function extractLogPayload(raw: string): ExtractedLogPayload | null {
	const clean = stripAnsi(raw)
	const jsonStart = clean.indexOf("{")
	if (jsonStart < 0) return null
	const prefix = clean.slice(0, jsonStart).trim()
	const jsonText = clean.slice(jsonStart).trim()
	return {
		timestamp: extractTimestamp(prefix),
		jsonText,
	}
}

function stripAnsi(value: string) {
	return value.replace(/\u001b\[[0-9;]*m/g, "")
}

function extractTimestamp(prefix: string) {
	const match = prefix.match(/\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}(?:\.\d+)?/)
	return match?.[0] || ""
}

function readString(value: unknown) {
	return typeof value === "string" ? value : ""
}

function readNumber(value: unknown) {
	return typeof value === "number" && Number.isFinite(value) ? value : undefined
}

function getHeaderValue(headers: Record<string, unknown>, name: string) {
	const target = headers?.[name]
	if (Array.isArray(target)) {
		return typeof target[0] === "string" ? target[0] : ""
	}
	return typeof target === "string" ? target : ""
}

function formatLogTime(value: unknown) {
	if (typeof value === "number" && Number.isFinite(value)) {
		return new Date(value * 1000).toLocaleTimeString("zh-CN", {
			hour12: false,
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		})
	}
	if (typeof value === "string" && value) {
		const normalized = value.includes("/") ? value.replace(/\//g, "-") : value
		const date = new Date(normalized)
		if (!Number.isNaN(date.getTime())) {
			return date.toLocaleTimeString("zh-CN", {
				hour12: false,
				hour: "2-digit",
				minute: "2-digit",
				second: "2-digit",
			})
		}
	}
	return "--:--:--"
}

function formatDuration(value: unknown) {
	if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
		return "--"
	}
	if (value >= 1) {
		return `${value.toFixed(value >= 10 ? 1 : 2)}s`
	}
	const ms = value * 1000
	if (ms >= 1) {
		return `${Math.round(ms)}ms`
	}
	return `${Math.max(1, Math.round(ms * 1000))}us`
}

function formatBytes(value: unknown) {
	if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
		return ""
	}
	if (value < 1024) {
		return `${Math.round(value)}B`
	}
	if (value < 1024 * 1024) {
		return `${(value / 1024).toFixed(1)}KB`
	}
	return `${(value / 1024 / 1024).toFixed(1)}MB`
}

function simplifyUserAgent(ua: string) {
	if (!ua) return ""
	const browser = detectBrowser(ua)
	const os = detectOS(ua)
	if (browser && os) return `${browser} / ${os}`
	if (browser) return browser
	return ua.length > 36 ? `${ua.slice(0, 36)}...` : ua
}

function detectBrowser(ua: string) {
	if (/curl\//i.test(ua)) return "curl"
	if (/PostmanRuntime/i.test(ua)) return "Postman"
	if (/Go-http-client/i.test(ua)) return "Go HTTP"
	if (/Edg\//i.test(ua)) return "Edge"
	if (/Chrome\//i.test(ua) && !/Edg\//i.test(ua)) return "Chrome"
	if (/Firefox\//i.test(ua)) return "Firefox"
	if (/Safari\//i.test(ua) && /Version\//i.test(ua) && !/Chrome\//i.test(ua)) return "Safari"
	return ""
}

function detectOS(ua: string) {
	if (/iPhone/i.test(ua)) return "iPhone"
	if (/iPad/i.test(ua)) return "iPad"
	if (/Android/i.test(ua)) return "Android"
	if (/Mac OS X|Macintosh/i.test(ua)) return "macOS"
	if (/Windows/i.test(ua)) return "Windows"
	if (/Linux/i.test(ua)) return "Linux"
	return ""
}

function getStatusTagType(status?: number): "default" | "info" | "success" | "warning" | "error" {
	if (!status) return "default"
	if (status >= 500) return "error"
	if (status >= 400) return "warning"
	if (status >= 300) return "info"
	if (status >= 200) return "success"
	return "default"
}

function matchStatusFilter(item: ParsedLogEntry) {
	if (statusFilter.value === "all") return true
	if (!item.status) return false
	return String(item.status).startsWith(statusFilter.value[0] || "")
}

function formatStructuredRaw(payload: unknown) {
	try {
		return JSON.stringify(payload, null, 2)
	} catch {
		return String(payload ?? "")
	}
}

function matchSearchKeyword(item: ParsedLogEntry) {
	const keyword = searchKeyword.value.trim().toLowerCase()
	if (!keyword) return true
	return [
		item.raw,
		item.path,
		item.ip,
		item.host,
		item.userAgent,
		item.userAgentFull,
		item.referer,
		item.statusText,
	].some(value => value.toLowerCase().includes(keyword))
}
</script>
