<template>
  <div class="mon-root mt-2 space-y-8">
    <div class="bg-white/86 rounded-[28px] border border-blue-100/80 p-8 shadow-[0_28px_80px_rgba(15,23,42,0.08)] backdrop-blur-xl sm:p-10">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-4">
          <div class="inline-flex rounded-full border border-blue-200 bg-blue-50 px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
            Host Monitor
          </div>
          <div class="text-4xl font-semibold leading-[1.08] fg-base-100 sm:text-5xl">主机实时概览</div>
          <div class="text-base leading-8 text-slate-500 sm:text-lg">查看关注主机的实时概览、健康分析</div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="item in summaryCards"
            :key="item.label"
            class="rounded-[24px] border border-slate-200 bg-slate-50/80 p-5"
          >
            <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
              {{ item.label }}
            </div>
            <div class="mt-3 text-xl font-semibold fg-base-100">{{ item.value }}</div>
            <div class="mt-2 text-sm leading-6 text-slate-500">{{ item.desc }}</div>
          </div>
        </div>
      </div>

      <div class="mt-8 grid gap-4 xl:grid-cols-[1.3fr_1fr]">
        <div class="rounded-[24px] border border-slate-200 bg-slate-50/90 p-5">
          <div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">Quick Range</div>
          <div class="mt-4 flex flex-wrap gap-3">
            <n-button
              v-for="item in quickRanges"
              :key="item.key"
              :type="activePreset === item.key ? 'primary' : 'default'"
              :ghost="activePreset !== item.key"
              class="!rounded-[18px] px-5"
              @click="applyQuickRange(item.key)"
            >
              {{ item.label }}
            </n-button>
          </div>
          <div class="mt-5 rounded-2xl border border-blue-100 bg-blue-50/70 p-2">
            <div class="flex flex-wrap justify-between">
              <n-button
                type="warning"
                ghost
                class="!rounded-[18px] px-5"
                @click="handleCleanMonitor"
              >
                清空历史监控
              </n-button>

              <n-date-picker
                :formatted-value="range"
                value-format="yyyy-MM-dd HH:mm:ss"
                type="datetimerange"
                clearable
                @update:formatted-value="handleRangeChange"
              />
            </div>
          </div>
        </div>

        <div class="rounded-[24px] border border-slate-200 bg-slate-50/90 p-5">
          <div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">Monitor Control</div>
          <div class="mt-4 grid gap-3 sm:grid-cols-2">
            <n-select
              :value="selectedIO"
              :options="ioSelectOptions"
              class="control-select"
              placeholder="选择磁盘"
              @update:value="handleIOChange"
            />
            <n-select
              :value="selectedNet"
              :options="netSelectOptions"
              class="control-select"
              placeholder="选择网卡"
              @update:value="handleNetChange"
            />
            <n-select
              :value="autoRefresh"
              :options="refreshOptions"
              class="control-select"
              placeholder="自动刷新"
              @update:value="handleRefreshChange"
            />
            <n-button
              ghost
              type="primary"
              class="!h-[48px] !rounded-[18px]"
              @click="refreshAll"
            >立即刷新</n-button>
          </div>
        </div>
      </div>
    </div>

    <MonitorLoad
      :rangeDate="range"
      :data="data.load"
      @search="getSearch"
    />
    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
      <MonitorCpu
        :rangeDate="range"
        :data="data.cpu"
        @search="getSearch"
      />
      <MonitorMemory
        :rangeDate="range"
        :data="data.memory"
        @search="getSearch"
      />
      <MonitorIo
        :rangeDate="range"
        :data="data.io"
        @search="getSearch"
      />
      <MonitorNetwork
        :rangeDate="range"
        :data="data.network"
        @search="getSearch"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { loadBaseInfo } from "@/api/modules/dashboard"
import { cleanMonitors, getIOOptions, getNetworkOptions, hostMonitorListAPI } from "@/api/modules/host"
import { useDialog, useMessage } from "naive-ui"
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue"
import MonitorLoad from "./components/MonitorLoad.vue"
import MonitorCpu from "./components/MonitorCpu.vue"
import MonitorMemory from "./components/MonitorMemory.vue"
import MonitorIo from "./components/MonitorIo.vue"
import MonitorNetwork from "./components/MonitorNetwork.vue"

type RangeValue = [string, string]

const message = useMessage()
const dialog = useDialog()

const data = reactive<Record<string, any>>({
	load: undefined,
	cpu: undefined,
	memory: undefined,
	io: undefined,
	network: undefined
})

const baseInfo = ref<any>(null)
 
const range = ref<RangeValue>(buildRange("today"))
const activePreset = ref("today")
const selectedIO = ref("all")
const selectedNet = ref("all")
const autoRefresh = ref<number>(5)
const ioOptions = ref<string[]>([])
const netOptions = ref<string[]>([])
let refreshTimer: ReturnType<typeof setInterval> | null = null

const quickRanges = [
	{ key: "today", label: "今天" },
	{ key: "1h", label: "最近 1 小时" },
	{ key: "6h", label: "最近 6 小时" },
	{ key: "24h", label: "最近 24 小时" },
	{ key: "7d", label: "最近 7 天" }
]

const refreshOptions = [
	{ label: "手动刷新", value: 0 },
	{ label: "5 秒", value: 5 },
	{ label: "10 秒", value: 10 },
	{ label: "30 秒", value: 30 },
	{ label: "60 秒", value: 60 },
	{ label: "120 秒", value: 120 }
]

const ioSelectOptions = computed(() => ioOptions.value.map(item => ({ label: item, value: item })))
const netSelectOptions = computed(() => netOptions.value.map(item => ({ label: item, value: item })))

const summaryCards = computed(() => [
	{
		label: "Host",
		value: baseInfo.value?.hostname || "--",
		desc: baseInfo.value?.os || "当前主机系统信息"
	},
	{
		label: "Network",
		value: selectedNet.value || "--",
		desc: "当前网络监控选中的网卡"
	},
	{
		label: "I/O Device",
		value: selectedIO.value || "--",
		desc: "当前 I/O 监控选中的磁盘"
	},
	{
		label: "Refresh",
		value: autoRefresh.value === 0 ? "手动" : `${autoRefresh.value}s`,
		desc: "当前自动刷新频率"
	}
])

function formatDateTime(date: Date) {
	const year = date.getFullYear()
	const month = String(date.getMonth() + 1).padStart(2, "0")
	const day = String(date.getDate()).padStart(2, "0")
	const hour = String(date.getHours()).padStart(2, "0")
	const minute = String(date.getMinutes()).padStart(2, "0")
	const second = String(date.getSeconds()).padStart(2, "0")
	return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}

function buildRange(type: string): RangeValue {
	const now = new Date()
	const start = new Date(now)
	if (type === "today") {
		start.setHours(0, 0, 0, 0)
	} else if (type === "1h") {
		start.setHours(start.getHours() - 1)
	} else if (type === "6h") {
		start.setHours(start.getHours() - 6)
	} else if (type === "24h") {
		start.setHours(start.getHours() - 24)
	} else if (type === "7d") {
		start.setDate(start.getDate() - 7)
	}
	return [formatDateTime(start), formatDateTime(now)]
}

function formatPercent(value?: number) {
	return `${Number(value || 0).toFixed(2)}%`
}

function formatBytes(value?: number) {
	const bytes = Number(value || 0)
	if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
	if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`
	if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`
	return `${bytes.toFixed(2)} B`
}

async function loadOverview() {
	const res = await loadBaseInfo(selectedIO.value || "all", selectedNet.value || "all")
	baseInfo.value = res.data
}

async function getSearch(params: any) {
	const nextParams = { ...params }
	if (nextParams.param === "network") {
		nextParams.info = selectedNet.value
	}
	if (nextParams.param === "io") {
		nextParams.info = selectedIO.value
	}
	const res = await hostMonitorListAPI(nextParams)
	const list = res.data || []
	if (nextParams.param === "all") {
		data.load = list.find((item: any) => item.param === "base")
		data.cpu = data.load
		data.memory = data.load
		data.io = list.find((item: any) => item.param === "io")
		data.network = list.find((item: any) => item.param === "network")
		return
	}
	data[nextParams.param as "load" | "cpu" | "memory" | "io" | "network"] = list[0]
}
const startTime = new Date(range.value[0]).toISOString()
const endTime = new Date(range.value[1]).toISOString()

async function loadAllTrends() {
	await Promise.all([
		getSearch({ startTime, endTime, param: "load" }),
		getSearch({ startTime, endTime, param: "cpu" }),
		getSearch({ startTime, endTime, param: "memory" }),
		getSearch({ startTime, endTime, param: "io" }),
		getSearch({ startTime, endTime, param: "network" })
	])
}

async function refreshAll() {
	try {
		await Promise.all([loadOverview(), loadAllTrends()])
	} catch {
		return
	}
}

function setupRefreshTimer() {
	if (refreshTimer) {
		clearInterval(refreshTimer)
		refreshTimer = null
	}
	if (autoRefresh.value > 0) {
		refreshTimer = setInterval(() => {
			refreshAll()
		}, autoRefresh.value * 1000)
	}
}

function applyQuickRange(type: string) {
	activePreset.value = type
	range.value = buildRange(type)
	refreshAll()
}

function handleRangeChange(value: string[] | null) {
	if (!value || value.length !== 2) return
	range.value = [value[0], value[1]]
	activePreset.value = "custom"
	refreshAll()
}

function handleIOChange(value: string) {
	selectedIO.value = value
	refreshAll()
}

function handleNetChange(value: string) {
	selectedNet.value = value
	refreshAll()
}

function handleRefreshChange(value: number) {
	autoRefresh.value = value
	setupRefreshTimer()
}

function handleCleanMonitor() {
	dialog.warning({
		title: "确认清空历史监控吗？",
		content: "清空后将删除 CPU、内存、I/O、网络等历史监控数据，仅保留后续新采集的数据。",
		positiveText: "确认清空",
		negativeText: "取消",
		onPositiveClick: async () => {
			await cleanMonitors()
			message.success("监控历史已清空")
			await refreshAll()
		}
	})
}

async function loadOptions() {
	const [ioRes, netRes] = await Promise.all([getIOOptions(), getNetworkOptions()])
	ioOptions.value = ioRes.data || []
	netOptions.value = netRes.data || []

	// const defaultIO = ioOptions.value.find(item => item !== "all") || ioOptions.value[0] || "all"
	// const defaultNet = netOptions.value.find(item => item !== "all") || netOptions.value[0] || "all"

	// selectedIO.value = defaultIO
	// selectedNet.value = defaultNet
}

onMounted(async () => {
	await loadOptions()
	await refreshAll()
	setupRefreshTimer()
})

onBeforeUnmount(() => {
	if (refreshTimer) {
		clearInterval(refreshTimer)
	}
})
</script>

<style scoped>
.control-select :deep(.n-input),
.control-select :deep(.n-base-selection) {
	--n-height: 48px;
	--n-border-radius: 18px;
}
</style>

<style>
.theme-dark .mon-root .text-slate-500,
.theme-dark .mon-root .text-slate-400 {
  color: var(--fg-secondary-color) !important;
}
.theme-dark .mon-root .text-slate-600,
.theme-dark .mon-root .text-slate-700,
.theme-dark .mon-root .text-slate-800,
.theme-dark .mon-root .text-slate-900 {
  color: var(--fg-default-color) !important;
}
.theme-dark .mon-root .border-slate-100,
.theme-dark .mon-root .border-slate-200 {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .mon-root .bg-slate-50\/80,
.theme-dark .mon-root .bg-slate-50\/90 {
  background-color: color-mix(in srgb, var(--bg-default-color) 95%, transparent) !important;
}
.theme-dark .mon-root .bg-white\/86 {
  background-color: color-mix(in srgb, var(--bg-default-color) 86%, transparent) !important;
}
.theme-dark .mon-root .border-blue-100\/80 {
  border-color: color-mix(in srgb, var(--primary-color) 20%, transparent) !important;
}
.theme-dark .mon-root .border-blue-200 {
  border-color: color-mix(in srgb, var(--primary-color) 30%, transparent) !important;
}
.theme-dark .mon-root .bg-blue-50 {
  background-color: color-mix(in srgb, var(--primary-color) 10%, var(--bg-default-color)) !important;
}
</style>
