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

        <WebsiteLogPanel
          :show-structured-list="showStructuredList"
          :show-filters="showFilters"
          :status-filter="statusFilter"
          :search-keyword="searchKeyword"
          :log-path="logPath"
          :loading="loading"
          :filtered-entries="filteredEntries"
          :parsed-entries="parsedEntries"
          :selected-log-raw="selectedLogRaw"
          :detail-visible="detailVisible"
          :page="page"
          :total="total"
          :can-toggle-view="canToggleView"
          :is-error-view="isErrorView"
          :empty-text="emptyText"
          :log-panel-title="logPanelTitle"
          :log-content="logContent"
          @update:show-filters="showFilters = $event"
          @update:raw-mode="rawMode = $event"
          @update:status-filter="statusFilter = $event"
          @update:search-keyword="searchKeyword = $event"
          @copy-path="copyPath"
          @open-detail="openDetail"
        />

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

      <WebsiteLogDetailModal
        :show="detailVisible"
        :title="detailTitle"
        :entry="activeEntry"
        @update:show="detailVisible = $event"
      />

      <WebsiteTodayIPStatsModal
        :show="todayStatsVisible"
        :loading="todayStatsLoading"
        :stats="todayStats"
        @update:show="todayStatsVisible = $event"
      />
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import { WebsiteLogAPI, WebsiteTodayIPStatsAPI } from "@/api/modules/website"
import { hasWebsiteRuntimeMeta } from "@/utils/websiteRuntime"
import { copyText } from "@/utils/util"
import { NButton, NDrawer, NDrawerContent, NTag, useMessage } from "naive-ui"
import { computed, ref, watch } from "vue"
import WebsiteLogDetailModal from "./WebsiteLogDetailModal.vue"
import WebsiteLogPanel from "./WebsiteLogPanel.vue"
import WebsiteTodayIPStatsModal from "./WebsiteTodayIPStatsModal.vue"
import { useWebsiteLogBindingMeta } from "./useWebsiteLogBindingMeta"
import {
  getErrorMessage,
  matchSearchKeyword,
  matchStatusFilter,
  parseLogLine,
  type ParsedLogEntry,
  type StatusFilter,
  type WebsiteLogType
} from "./websiteLogHelpers"

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

const message = useMessage()
const { bindingRuntimeText, loadBindingMeta } = useWebsiteLogBindingMeta(website)

const drawerTitle = computed(() => (logType.value === "error" ? "网站错误日志" : "网站访问记录"))
const logLabel = computed(() => (logType.value === "error" ? "错误日志" : "访问日志"))
const emptyText = computed(() =>
  logType.value === "error" ? "暂无错误日志。若网站当前没有报错，这里会暂时为空。" : "暂无访问记录。若网站刚创建或还没有请求进入，这里会暂时为空。"
)
const logPanelTitle = computed(() => (logType.value === "error" ? "Error Log" : "Access Log"))
const simpleHelperText = computed(() =>
  logType.value === "error" ? "这里按条展示网站异常请求，默认只突出最关键的错误信息。" : "这里按条展示网站访问记录，默认只保留最核心的访问信息。"
)
const parsedEntries = computed(() => logLines.value.map(parseLogLine).filter((item): item is ParsedLogEntry => !!item))
const filteredEntries = computed(() =>
  parsedEntries.value.filter(item => matchStatusFilter(item, statusFilter.value)).filter(item => matchSearchKeyword(item, searchKeyword.value))
)
const activeEntry = computed(() => filteredEntries.value.find(item => item.raw === selectedLogRaw.value) || null)
const canToggleView = computed(() => logLines.value.length > 0)
const showStructuredList = computed(() => logLines.value.length > 0 && !rawMode.value)
const isErrorView = computed(() => logType.value === "error")
const detailTitle = computed(() => (!activeEntry.value ? "日志详情" : `${activeEntry.value.method} ${activeEntry.value.path}`))

function closeDetail() {
  detailVisible.value = false
  selectedLogRaw.value = ""
}

watch(activeEntry, entry => {
  if (!entry && detailVisible.value) {
    closeDetail()
  }
})

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
      logType: logType.value
    })
    logContent.value = res.data?.content || ""
    logLines.value = res.data?.lines || (logContent.value ? logContent.value.split("\n").filter(Boolean) : [])
    logPath.value = res.data?.path || getDefaultLogPath()
    total.value = res.data?.total || 1
    end.value = !!res.data?.end
    page.value = latest ? Math.max(total.value, 1) : targetPage
  } catch (error) {
    // 错误提示由请求拦截器统一处理
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
      websiteId: website.value.id
    })
    todayStats.value = res.data || null
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  } finally {
    todayStatsLoading.value = false
  }
}

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
  logPath.value = type === "error" ? "" : row.accessLogPath || ""
  if (!hasWebsiteRuntimeMeta(row)) {
    loadBindingMeta()
  }
  loadLogs(true)
}

function changePage(nextPage: number) {
  closeDetail()
  page.value = nextPage
  loadLogs(false)
}

function openDetail(item: ParsedLogEntry) {
  selectedLogRaw.value = item.raw
  detailVisible.value = true
}

defineExpose({ open })
</script>
