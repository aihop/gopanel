<template>
  <div class="space-y-8">
    <div class="info-strip bg-base-accent border-base-accent">
      <div class="info-strip__main">
        <div class="info-strip__eyebrow">{{ t("home.baseInfo") }}</div>
        <div class="info-strip__title">{{ baseInfo.hostname || "--" }}</div>
        <div class="info-strip__desc">
          <div>{{
						[baseInfo.os, baseInfo.platform, baseInfo.platformVersion, baseInfo.kernelArch]
							.filter(Boolean)
							.join(" · ") || "--"
					}}</div>
          <div class="mt-3 flex items-center gap-3">
            <n-tag
              size="small"
              :bordered="false"
              :type="lowPowerMode ? 'warning' : 'info'"
              round
            >
              {{ lowPowerMode ? "省电模式" : "标准模式" }}
            </n-tag>
            <n-switch
              size="small"
              :value="lowPowerMode"
              @update:value="setLowPowerMode"
            />
          </div>
        </div>
      </div>
      <div class="info-strip__items">
        <div class="info-chip">
          <span class="info-chip__label">{{ $t("home.systemInfo") }}</span>
          <span class="info-chip__value">{{ baseInfo.os || "--" }}</span>
        </div>
        <div class="info-chip">
          <span class="info-chip__label">{{ $t("home.uptime") }}</span>
          <span class="info-chip__value">{{ currentInfo.timeSinceUptime || "--" }}</span>
        </div>
        <div class="info-chip">
          <span class="info-chip__label">{{ $t("home.runningTime") }}</span>
          <span class="info-chip__value">{{ formatUptime(currentInfo.uptime) }}</span>
        </div>
        <div class="info-chip">
          <span class="info-chip__label">{{ $t("menu.process") }}</span>
          <span class="info-chip__value">{{ currentInfo.procs }}</span>
        </div>
        <div class="info-chip">
          <span class="info-chip__label">IPv4</span>
          <span class="info-chip__value">{{ baseInfo.ipv4Addr || "--" }}</span>
        </div>
        <div class="info-chip">
          <span class="info-chip__label">{{ $t("home.kernelVersion") }}</span>
          <div class="info-chip__value flex items-center justify-between gap-2">
            <span class="min-w-0 truncate">{{ shortText(baseInfo.kernelVersion || "--", 20) }}</span>
            <div class="flex items-center gap-2">
              <n-button
                size="tiny"
                quaternary
                type="warning"
                :loading="memoryCleaning"
                :disabled="memoryCleaning"
                @click="handleMemoryClean"
              >
                清理内存
              </n-button>
              <n-button
                size="tiny"
                quaternary
                type="primary"
                :loading="cpuRelieving"
                :disabled="cpuRelieving"
                @click="handleCpuRelieve"
              >
                释放 CPU
              </n-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
      <n-popover
        placement="bottom"
        :width="loadWidth()"
        trigger="hover"
      >
        <template #trigger>
          <div class="status-card bg-base-100 border-base-accent">
            <div class="status-card__header">
              <div>
                <div class="status-card__label">CPU</div>
                <div class="status-card__value">{{ formatNumber(currentInfo.cpuUsedPercent) }}%</div>
              </div>
              <ProgressRing
                title=""
                :value="formatNumber(currentInfo.cpuUsedPercent)"
                :size="100"
                :stroke-width="7"
              />
            </div>
            <div class="status-card__meta">
              <span>{{ t("home.core") }} {{ baseInfo.cpuCores }}</span>
              <span>{{ t("home.logicCore") }} {{ baseInfo.cpuLogicalCores }}</span>
            </div>
            <div class="status-card__desc">{{ shortText(baseInfo.cpuModelName, 46) }}</div>
          </div>
        </template>
        <div class="space-y-3">
          <n-tag v-if="baseInfo.cpuModelName">{{ baseInfo.cpuModelName }}</n-tag>
          <div class="grid grid-cols-2 gap-1">
            <div
              v-for="(item, index) of currentInfo.cpuPercent"
              :key="index"
            >
              <n-tag
                v-if="cpuShowAll || index < 24"
                class="tagCPUClass"
              >
                CPU-{{ index }}: {{ formatNumber(item) }}%
              </n-tag>
            </div>
          </div>
          <div v-if="currentInfo.cpuPercent.length > 24">
            <n-button
              v-if="!cpuShowAll"
              text
              type="primary"
              size="small"
              @click="cpuShowAll = true"
            >
              {{ t("commons.button.showAll") }}
            </n-button>
            <n-button
              v-else
              text
              type="primary"
              size="small"
              @click="cpuShowAll = false"
            >
              {{ t("commons.button.hideSome") }}
            </n-button>
          </div>
        </div>
      </n-popover>

      <n-popover
        placement="bottom"
        :width="360"
        trigger="hover"
      >
        <template #trigger>
          <div class="status-card bg-base-100 border-base-accent">
            <div class="status-card__header">
              <div>
                <div class="status-card__label">{{ t("monitor.memory") }}</div>
                <div class="status-card__value">{{ computeSize(currentInfo.memoryUsed) }}</div>
              </div>
              <ProgressRing
                title=""
                :value="formatNumber(currentInfo.memoryUsedPercent)"
                :size="100"
                :stroke-width="7"
              />
            </div>
            <div class="status-card__meta">
              <span>{{ t("home.total") }} {{ computeSize(currentInfo.memoryTotal) }}</span>
              <span>{{ t("home.free") }} {{ computeSize(currentInfo.memoryAvailable) }}</span>
            </div>
            <div class="status-card__desc">
              {{ t("home.percent") }} {{ formatNumber(currentInfo.memoryUsedPercent) }}%
            </div>
          </div>
        </template>
        <div class="grid grid-cols-1 gap-2">
          <n-tag>{{ t("home.total") }}: {{ computeSize(currentInfo.memoryTotal) }}</n-tag>
          <n-tag>{{ t("home.used") }}: {{ computeSize(currentInfo.memoryUsed) }}</n-tag>
          <n-tag>{{ t("home.free") }}: {{ computeSize(currentInfo.memoryAvailable) }}</n-tag>
          <n-tag v-if="currentInfo.swapMemoryTotal">
            Swap: {{ computeSize(currentInfo.swapMemoryUsed) }} /
            {{ computeSize(currentInfo.swapMemoryTotal) }}
          </n-tag>
          <n-popconfirm
            :show-icon="false"
            @positive-click="handleMemoryClean"
          >
            <template #trigger>
              <n-button
                size="small"
                type="warning"
                ghost
                :loading="memoryCleaning"
                :disabled="memoryCleaning"
              >
                清理缓存
              </n-button>
            </template>
            该操作会尝试清理 Linux 内核缓存以回收内存，不会中断 HTTP 服务，但可能短暂增加磁盘 IO。继续？
          </n-popconfirm>
        </div>
      </n-popover>

      <n-popover
        placement="bottom"
        :width="320"
        trigger="hover"
      >
        <template #trigger>
          <div class="status-card bg-base-100 border-base-accent">
            <div class="status-card__header">
              <div>
                <div class="status-card__label">{{ t("home.load") }}</div>
                <div class="status-card__value">{{ formatNumber(currentInfo.loadUsagePercent) }}%</div>
              </div>
              <ProgressRing
                title=""
                :value="formatNumber(currentInfo.loadUsagePercent)"
                :size="100"
                :stroke-width="7"
              />
            </div>
            <div class="status-card__meta">
              <span>1m {{ formatNumber(currentInfo.load1) }}</span>
              <span>5m {{ formatNumber(currentInfo.load5) }}</span>
              <span>15m {{ formatNumber(currentInfo.load15) }}</span>
            </div>
            <div class="status-card__desc">{{ loadStatus(currentInfo.loadUsagePercent) }}</div>
          </div>
        </template>
        <div class="grid grid-cols-1 gap-2">
          <n-tag>{{ t("home.loadAverage", 1) }}: {{ formatNumber(currentInfo.load1) }}</n-tag>
          <n-tag>{{ t("home.loadAverage", 5) }}: {{ formatNumber(currentInfo.load5) }}</n-tag>
          <n-tag>{{ t("home.loadAverage", 15) }}: {{ formatNumber(currentInfo.load15) }}</n-tag>
        </div>
      </n-popover>
    </div>

    <div class="rounded-2xl border border-slate-200 bg-base-100 p-6 shadow-sm">
      <div class="mb-5 flex items-center justify-between gap-4">
        <div>
          <div class="text-sm font-medium text-slate-500">{{ $t("monitor.disk") }}</div>
          <div class="mt-1 text-xl font-semibold fg-base-100">
            {{ currentInfo.diskData.length }} {{ $t("home.blockDevice") }}
          </div>
        </div>
        <n-button
          v-if="hasMoreDisks"
          text
          type="primary"
          @click="toggleDiskExpanded"
        >
          {{ diskExpanded ? $t("tabs.hide") : $t("tabs.more") }}
        </n-button>
      </div>
      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2 2xl:grid-cols-3">
        <n-popover
          v-for="(item, index) in visibleDiskData"
          :key="`disk-${index}`"
          placement="bottom"
          :width="420"
          trigger="hover"
        >
          <template #trigger>
            <div class="disk-card">
              <div class="flex min-w-0 items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold fg-base-100">{{ item.path }}</div>
                  <div class="mt-1 flex flex-wrap gap-2 text-xs text-slate-500">
                    <span>{{ item.type }}</span>
                    <span>{{ shortText(item.device, 20) }}</span>
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-base font-semibold fg-base-100">
                    {{ formatNumber(item.usedPercent) }}%
                  </div>
                  <div class="text-xs text-slate-500">
                    {{ computeSize(item.used) }} / {{ computeSize(item.total) }}
                  </div>
                </div>
              </div>
              <n-progress
                class="mt-3"
                type="line"
                :show-indicator="false"
                :height="8"
                :processing="false"
                :color="progressColor(item.usedPercent)"
                :rail-color="'#e2e8f0'"
                :percentage="formatNumber(item.usedPercent)"
              />
            </div>
          </template>
          <div class="space-y-2">
            <n-tag>{{ $t("home.mount") }}: {{ item.path }}</n-tag>
            <n-tag>{{ $t("commons.table.type") }}: {{ item.type }}</n-tag>
            <n-tag>{{ $t("home.fileSystem") }}: {{ item.device }}</n-tag>
            <n-tag>Inode: {{ item.inodesUsed }} / {{ item.inodesTotal }}</n-tag>
            <n-tag>{{ $t("home.free") }}: {{ computeSize(item.free) }}</n-tag>
          </div>
        </n-popover>
      </div>
    </div>

    <div
      v-if="acceleratorList.length"
      class="rounded-2xl border border-slate-200 bg-base-100 p-6 shadow-sm"
    >
      <div class="mb-5 flex items-center justify-between gap-4">
        <div>
          <div class="text-sm font-medium text-slate-500">GPU / XPU</div>
          <div class="mt-1 text-xl font-semibold fg-base-100">
            {{ acceleratorList.length }} {{ $t("home.blockDevice") }}
          </div>
        </div>
        <n-button
          v-if="hasMoreAccelerators"
          text
          type="primary"
          @click="toggleAcceleratorExpanded"
        >
          {{ acceleratorExpanded ? t("tabs.hide") : t("tabs.more") }}
        </n-button>
      </div>
      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <n-popover
          v-for="item in visibleAccelerators"
          :key="item.key"
          placement="bottom"
          :width="320"
          trigger="hover"
        >
          <template #trigger>
            <div
              class="accelerator-card"
              @click="goGPU"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-sm font-semibold fg-base-100">{{ item.title }}</div>
                  <div class="mt-1 truncate text-xs text-slate-500">{{ item.name }}</div>
                </div>
                <div class="text-right">
                  <div class="text-base font-semibold fg-base-100">{{ item.util }}%</div>
                  <div class="text-xs text-slate-500">{{ item.extra }}</div>
                </div>
              </div>
              <n-progress
                class="mt-3"
                type="line"
                :show-indicator="false"
                :height="8"
                :processing="false"
                :color="progressColor(item.util)"
                :rail-color="'#e2e8f0'"
                :percentage="item.util"
              />
            </div>
          </template>
          <div class="space-y-2">
            <n-tag>{{ item.name }}</n-tag>
            <n-tag>{{ t("monitor.gpuUtil") }}: {{ item.rawUtil }}</n-tag>
            <n-tag>{{ t("monitor.temperature") }}: {{ item.temperature }}</n-tag>
            <n-tag>{{ t("monitor.powerUsage") }}: {{ item.power }}</n-tag>
            <n-tag>{{ t("monitor.memoryUsage") }}: {{ item.memory }}</n-tag>
          </div>
        </n-popover>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { computeSize } from "@/utils/util"
import { useRouter } from "vue-router"
import { computed, onMounted, onUnmounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import ProgressRing from "./ProgressRing.vue"
import emitter from "@/utils/emitter"
import { clearMemoryCaches, relieveCPU } from "@/api/modules/host"

const router = useRouter()
const { t } = useI18n()
const message = useMessage()

const diskExpanded = ref(false)
const acceleratorExpanded = ref(false)
let uptimeTimer: ReturnType<typeof setInterval> | null = null
const handleLowPowerMode = (value: unknown) => {
	if (typeof value !== "boolean") return
	lowPowerMode.value = value
	startUptimeTimer()
}

const baseInfo = ref<Dashboard.BaseInfo>({
	websiteNumber: 0,
	databaseNumber: 0,
	cronjobNumber: 0,
	appInstalledNumber: 0,

	hostname: "",
	os: "",
	platform: "",
	platformFamily: "",
	platformVersion: "",
	kernelArch: "",
	kernelVersion: "",
	virtualizationSystem: "",

	cpuCores: 0,
	cpuLogicalCores: 0,
	cpuModelName: "",
	currentInfo: {} as Dashboard.CurrentInfo,

	ipv4Addr: "",
	systemProxy: ""
})

const currentInfo = ref<Dashboard.CurrentInfo>({
	uptime: 0,
	timeSinceUptime: "",
	procs: 0,

	load1: 0,
	load5: 0,
	load15: 0,
	loadUsagePercent: 0,

	cpuPercent: [] as Array<number>,
	cpuUsedPercent: 0,
	cpuUsed: 0,
	cpuTotal: 0,

	memoryTotal: 0,
	memoryAvailable: 0,
	memoryUsed: 0,
	memoryUsedPercent: 0,
	swapMemoryTotal: 0,
	swapMemoryAvailable: 0,
	swapMemoryUsed: 0,
	swapMemoryUsedPercent: 0,

	ioReadBytes: 0,
	ioWriteBytes: 0,
	ioCount: 0,
	ioReadTime: 0,
	ioWriteTime: 0,

	diskData: [],
	gpuData: [],
	xpuData: [],

	netBytesSent: 0,
	netBytesRecv: 0,
	shotTime: new Date()
})

const cpuShowAll = ref(false)
const lowPowerMode = ref(localStorage.getItem("gopanel_low_power_mode") === "1")
const memoryCleaning = ref(false)
const cpuRelieving = ref(false)

const visibleDiskData = computed(() =>
	diskExpanded.value ? currentInfo.value.diskData : currentInfo.value.diskData.slice(0, 4)
)
const hasMoreDisks = computed(() => currentInfo.value.diskData.length > 4)

const acceleratorList = computed(() => {
	const gpuItems = (currentInfo.value.gpuData || []).map(item => ({
		key: `gpu-${item.index}`,
		title: `GPU-${item.index}`,
		name: item.productName,
		util: parseUtil(item.gpuUtil),
		rawUtil: item.gpuUtil,
		temperature: item.temperature.replace(/C/g, "°C"),
		power: item.powerUsage,
		memory: item.memoryUsage,
		extra: item.temperature.replace(/C/g, "°C")
	}))
	const xpuItems = (currentInfo.value.xpuData || []).map(item => ({
		key: `xpu-${item.deviceID}`,
		title: `XPU-${item.deviceID}`,
		name: item.deviceName,
		util: parseUtil(item.memoryUtil),
		rawUtil: item.memoryUtil,
		temperature: item.temperature,
		power: item.power,
		memory: `${item.memoryUsed}/${item.memory}`,
		extra: item.temperature
	}))
	return [...gpuItems, ...xpuItems]
})

const visibleAccelerators = computed(() =>
	acceleratorExpanded.value ? acceleratorList.value : acceleratorList.value.slice(0, 3)
)
const hasMoreAccelerators = computed(() => acceleratorList.value.length > 3)

const acceptParams = (current: Dashboard.CurrentInfo, base: Dashboard.BaseInfo): void => {
	currentInfo.value = current
	baseInfo.value = base
	currentInfo.value.diskData = currentInfo.value.diskData || []
	currentInfo.value.gpuData = currentInfo.value.gpuData || []
	currentInfo.value.xpuData = currentInfo.value.xpuData || []
	diskExpanded.value = localStorage.getItem("dashboard_disk_show") === "more"
	acceleratorExpanded.value = localStorage.getItem("dashboard_accelerator_show") === "more"
}

function loadStatus(val: number) {
	if (val < 30) {
		return t("home.runSmoothly")
	}
	if (val < 70) {
		return t("home.runNormal")
	}
	if (val < 80) {
		return t("home.runSlowly")
	}
	return t("home.runJam")
}

const goGPU = () => {
	router.push({ name: "GPU" })
}

const loadWidth = () => {
	if (!cpuShowAll.value || currentInfo.value.cpuPercent.length < 24) {
		return 310
	}
	let line = Math.floor(currentInfo.value.cpuPercent.length / 16)
	return line * 141 + 28
}

function formatNumber(val: number) {
	return Number(val.toFixed(2))
}

function parseUtil(value: string) {
	return formatNumber(Number(String(value).replace(/[^\d.]/g, "")) || 0)
}

function shortText(value: string, max: number) {
	return value.length > max ? `${value.substring(0, max - 3)}...` : value
}

function formatUptime(uptime: number) {
	if (!uptime) return "--"
	const days = Math.floor(uptime / 86400)
	const hours = Math.floor((uptime % 86400) / 3600)
	const minutes = Math.floor((uptime % 3600) / 60)
	const seconds = Math.floor(uptime % 60)
	const parts: string[] = []
	if (days > 0) parts.push(`${days}天`)
	if (hours > 0) parts.push(`${hours}小时`)
	if (minutes > 0) parts.push(`${minutes}分钟`)
	if (!parts.length || seconds > 0) parts.push(`${seconds}秒`)
	return parts.join("")
}

function progressColor(value: number) {
	if (value >= 85) return "#f43f5e"
	if (value >= 65) return "#f59e0b"
	return "#2563eb"
}

function toggleDiskExpanded() {
	diskExpanded.value = !diskExpanded.value
	localStorage.setItem("dashboard_disk_show", diskExpanded.value ? "more" : "hide")
}

function toggleAcceleratorExpanded() {
	acceleratorExpanded.value = !acceleratorExpanded.value
	localStorage.setItem("dashboard_accelerator_show", acceleratorExpanded.value ? "more" : "hide")
}

onMounted(() => {
	startUptimeTimer()
	emitter.on("gopanel:lowPowerMode", handleLowPowerMode)
})

onUnmounted(() => {
	if (uptimeTimer) {
		clearInterval(uptimeTimer)
		uptimeTimer = null
	}
	emitter.off("gopanel:lowPowerMode", handleLowPowerMode)
})

function setLowPowerMode(value: boolean) {
	lowPowerMode.value = value
	localStorage.setItem("gopanel_low_power_mode", value ? "1" : "0")
	emitter.emit("gopanel:lowPowerMode", value)
	startUptimeTimer()
}

function startUptimeTimer() {
	if (uptimeTimer) {
		clearInterval(uptimeTimer)
		uptimeTimer = null
	}
	const interval = lowPowerMode.value ? 5000 : 1000
	uptimeTimer = setInterval(() => {
		if (currentInfo.value.uptime > 0) {
			currentInfo.value.uptime += lowPowerMode.value ? 5 : 1
		}
	}, interval)
}

async function handleMemoryClean() {
	if (memoryCleaning.value) return
	memoryCleaning.value = true
	try {
		const res: any = await clearMemoryCaches({ mode: 3 })
		const data = res?.data || {}
		if (data.needPrivilege) {
			message.warning(data.message || "权限不足：需要管理员权限执行清理内核缓存")
		} else if (data.message) {
			message.success(data.message)
		} else {
			message.success("已触发清理内存")
		}
	} catch (err: any) {
	} finally {
		memoryCleaning.value = false
	}
}

async function handleCpuRelieve() {
	if (cpuRelieving.value) return
	cpuRelieving.value = true
	try {
		if (!lowPowerMode.value) {
			setLowPowerMode(true)
		}
		const res: any = await relieveCPU({ level: 10 })
		const data = res?.data || {}
		message.success(data.message || "已释放 CPU（降低刷新频率 + 降低进程优先级）")
	} catch (err: any) {
	} finally {
		cpuRelieving.value = false
	}
}

defineExpose({
	acceptParams
})
</script>

<style scoped lang="scss">
.info-strip {
	border-radius: 22px;
	padding: 22px 24px;
	display: grid;
	grid-template-columns: minmax(0, 1.3fr) minmax(0, 2fr);
	gap: 20px;
	box-shadow: 0 1px 2px rgba(37, 99, 235, 0.05);
}

.info-strip__main {
	min-width: 0;
	display: flex;
	flex-direction: column;
	justify-content: center;
	gap: 8px;
}

.info-strip__eyebrow {
	font-size: 12px;
	font-weight: 700;
	letter-spacing: 0.08em;
	text-transform: uppercase;
	color: #2563eb;
}

.info-strip__title {
	font-size: 24px;
	line-height: 1.1;
	font-weight: 700;
	color: #0f172a;
}

.info-strip__desc {
	font-size: 13px;
	line-height: 1.6;
	color: #64748b;
	word-break: break-word;
}

.info-strip__items {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 12px;
}

.info-chip {
	border: 1px solid rgba(147, 197, 253, 0.45);
	background: rgba(255, 255, 255, 0.9);
	border-radius: 16px;
	padding: 12px 14px;
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 6px;
}

.info-chip--action {
	border-color: rgba(37, 99, 235, 0.35);
	background: rgba(219, 234, 254, 0.45);
}

.info-chip__label {
	font-size: 12px;
	font-weight: 600;
	color: #64748b;
}

.info-chip__value {
	font-size: 14px;
	font-weight: 600;
	color: #0f172a;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.status-card {
	border-radius: 20px;
	padding: 22px;
	min-height: 224px;
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	gap: 18px;
	box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.status-card__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
}

.status-card__label {
	font-size: 13px;
	font-weight: 600;
	color: #64748b;
}

.status-card__value {
	margin-top: 6px;
	font-size: 30px;
	line-height: 1.1;
	font-weight: 700;
	color: #0f172a;
}

.status-card__meta {
	display: flex;
	flex-wrap: wrap;
	gap: 10px;
	font-size: 13px;
	color: #64748b;
}

.status-card__desc {
	font-size: 13px;
	line-height: 1.6;
	color: #94a3b8;
}

.disk-card,
.accelerator-card {
	border: 1px solid #e2e8f0;
	border-radius: 16px;
	padding: 16px;
	background: #f8fafc;
	transition: all 0.2s ease;
}

.accelerator-card {
	cursor: pointer;
}

.disk-card:hover,
.accelerator-card:hover,
.status-card:hover {
	border-color: #93c5fd;
	background: #fff;
}

.tagCPUClass {
	justify-content: flex-start !important;
	text-align: left !important;
	width: 140px;
}

@media (max-width: 1280px) {
	.info-strip {
		grid-template-columns: 1fr;
	}

	.info-strip__items {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
}

@media (max-width: 640px) {
	.info-strip__items {
		grid-template-columns: 1fr;
	}
}
</style>
