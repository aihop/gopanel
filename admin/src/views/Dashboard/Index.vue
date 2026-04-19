<template>
  <div class="grid grid-cols-1 gap-8 2xl:grid-cols-[minmax(0,1fr)_300px]">
    <div class="min-w-0 space-y-8">
      <StatusCard ref="statusRef"></StatusCard>

      <n-card
        :title="$t('menu.monitor')"
        :bordered="false"
        class="shadow"
      >
        <!-- header 右侧插槽 -->
        <template #header-extra>
          <n-space>
            <!-- 单选按钮组 -->
            <n-radio-group
              v-model:value="chartOption"
              style="float: right; margin-left: 5px"
              @update:value="changeOption"
            >
              <n-radio-button value="network">{{ $t("home.network") }}</n-radio-button>
              <n-radio-button value="io">{{ $t("home.io") }}</n-radio-button>
            </n-radio-group>

            <!-- 网卡下拉框 -->
            <n-select
              v-if="chartOption === 'network'"
              v-model:value="searchInfo.netOption"
              :options="netSelectOptions"
              placeholder=""
              style="width: 200px; float: right"
              @update:value="onLoadBaseInfo(false, 'network')"
            >
              <!-- <template #prefix>{{ $t("home.networkCard") }}:</template> -->
            </n-select>

            <!-- 磁盘下拉框 -->
            <n-select
              v-if="chartOption === 'io'"
              v-model:value="searchInfo.ioOption"
              :options="ioSelectOptions"
              placeholder=""
              style="width: 200px; float: right"
              @update:value="onLoadBaseInfo(false, 'io')"
            >
              <!-- <template #prefix>{{ $t("home.disk") }}:</template> -->
            </n-select>
          </n-space>
        </template>

        <template #default>
          <!-- Network 标签 -->
          <n-space
            v-if="chartOption === 'network'"
            class="monitor-tags"
          >
            <n-tag type="info">
              {{ $t("monitor.up") }}: {{ computeSizeFromKBs(currentChartInfo.netBytesSent) }}
            </n-tag>
            <n-tag type="info">
              {{ $t("monitor.down") }}: {{ computeSizeFromKBs(currentChartInfo.netBytesRecv) }}
            </n-tag>
            <n-tag type="info">
              {{ $t("home.totalSend") }}: {{ computeSize(currentInfo.netBytesSent) }}
            </n-tag>
            <n-tag type="info">
              {{ $t("home.totalRecv") }}: {{ computeSize(currentInfo.netBytesRecv) }}
            </n-tag>
          </n-space>

          <!-- IO 标签 -->
          <n-space
            v-if="chartOption === 'io'"
            class="monitor-tags"
          >
            <n-tag type="info">{{ $t("monitor.read") }}: {{ currentChartInfo.ioReadBytes }} MB</n-tag>
            <n-tag type="info">{{ $t("monitor.write") }}: {{ currentChartInfo.ioWriteBytes }} MB</n-tag>
            <n-tag type="info">
              {{ $t("home.rwPerSecond") }}: {{ currentChartInfo.ioCount }}
              {{ $t("commons.units.time") }}/s
            </n-tag>
            <n-tag type="info">{{ $t("home.ioDelay") }}: {{ currentChartInfo.ioTime }} ms</n-tag>
          </n-space>

          <!-- IO 图表 -->
          <div
            v-if="chartOption === 'io'"
            style="margin-top: 40px"
            class="mobile-monitor-chart"
          >
            <TrendSvg
              :title="$t('home.io')"
              :metric="`${currentChartInfo.ioReadBytes} MB / ${currentChartInfo.ioWriteBytes} MB`"
              :points="ioReadBytes"
              :secondary-points="ioWriteBytes"
              :labels="timeIODatas"
              :primary-label="$t('monitor.read')"
              :secondary-label="$t('monitor.write')"
              :badge="`${$t('home.ioDelay')}: ${currentChartInfo.ioTime} ms`"
            />
          </div>

          <!-- Network 图表 -->
          <div
            v-if="chartOption === 'network'"
            style="margin-top: 40px"
            class="mobile-monitor-chart"
          >
            <TrendSvg
              :title="$t('home.network')"
              :metric="`${computeSizeFromKBs(currentChartInfo.netBytesSent)} / ${computeSizeFromKBs(currentChartInfo.netBytesRecv)}`"
              :points="netBytesSents"
              :secondary-points="netBytesRecvs"
              :labels="timeNetDatas"
              :primary-label="$t('monitor.up')"
              :secondary-label="$t('monitor.down')"
              :badge="searchInfo.netOption"
            />
          </div>
        </template>
      </n-card>
    </div>
    <div class="space-y-8 self-start 2xl:sticky 2xl:top-6">
      <div class="rounded-2xl bg-base-accent border-base-accent p-5 shadow-sm">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">GoPanel Guide</div>
        <div class="mt-3 text-lg font-semibold fg-base-100">{{ $t("home.homeHelper") }}</div>
        <div class="mt-2 text-sm leading-6 text-slate-500">
          {{ $t("home.homeHelperDesc") }}
        </div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-base-100 p-5 shadow-sm">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">Popular Apps</div>
        <div class="mt-3 text-lg font-semibold fg-base-100">{{ $t("home.popularApps") }}</div>
        <div class="mt-2 text-sm leading-6 text-slate-500">
          {{ $t("home.popularAppsDesc") }}
        </div>
        <div class="mt-5">
          <n-spin :show="popularAppsLoading">
            <div
              v-if="popularApps.length"
              class="space-y-3"
            >
              <div
                v-for="item in popularApps"
                :key="item.id"
                class="rounded-xl border border-slate-200 bg-slate-50/80 p-4"
              >
                <div class="flex items-start gap-3">
                  <div class="flex h-11 w-11 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-slate-200 bg-white">
                    <img
                      v-if="item.icon"
                      :src="item.icon"
                      :alt="item.name"
                      class="h-8 w-8 object-contain"
                    />
                    <span
                      v-else
                      class="text-sm font-semibold text-slate-400"
                    >
                      {{ item.name.slice(0, 1).toUpperCase() }}
                    </span>
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0">
                        <div class="truncate text-sm font-semibold fg-base-100">{{ item.name }}</div>
                        <div class="mt-1 line-clamp-2 text-xs leading-5 text-slate-500">
                          {{ item.shortDescZh || item.description || "暂无应用说明" }}
                        </div>
                      </div>
                      <n-tag
                        size="small"
                        type="info"
                        :bordered="false"
                      >
                        {{ item.type || "App" }}
                      </n-tag>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div
              v-else
              class="rounded-xl border border-dashed border-slate-200 bg-slate-50/60 px-4 py-6 text-center text-sm text-slate-400"
            >
              暂无热门应用
            </div>
          </n-spin>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { App } from "@/api/interface/apps"
import { appsSearchAPI } from "@/api/modules/apps"
import { loadBaseInfo, loadCurrentInfo } from "@/api/modules/dashboard"
import { getIOOptions, getNetworkOptions } from "@/api/modules/host"
import { computeSize, computeSizeFromKBs, dateFormatForSecond } from "@/utils/util"
import { computed, onMounted, ref, onUnmounted } from "vue"
import { useI18n } from "vue-i18n"
import StatusCard from "./components/StatusCard.vue"
import TrendSvg from "./components/TrendSvg.vue"
import emitter from "@/utils/emitter"
import {
	baseInfo,
	currentChartInfo,
	currentInfo,
	ioReadBytes,
	ioWriteBytes,
	netBytesRecvs,
	netBytesSents,
	searchInfo,
	timeIODatas,
	timeNetDatas
} from "./Index"

const { t } = useI18n()

const statusRef = ref()

const chartOption = ref("network")
const popularApps = ref<App.AppDTO[]>([])
const popularAppsLoading = ref(false)

let timer: ReturnType<typeof setInterval> | null = null
let isInit = ref<boolean>(true)
let isActive = ref(true)
const lowPowerMode = ref(localStorage.getItem("gopanel_low_power_mode") === "1")
const handleLowPowerMode = (value: unknown) => {
	if (typeof value !== "boolean") return
	lowPowerMode.value = value
	startPollingTimer()
}

function syncStatusCard() {
	statusRef.value?.acceptParams(currentInfo.value, baseInfo.value)
}

const netSelectOptions = computed(() =>
	(netOptions.value || []).map((value: string) => ({
		label: value === "all" ? t("commons.table.all") : value,
		value
	}))
)

const ioSelectOptions = computed(() =>
	(ioOptions.value || []).map((value: string) => ({
		label: value === "all" ? t("commons.table.all") : value,
		value
	}))
)

async function onLoadBaseInfo(isInit: boolean, range: string) {
	if (range === "all" || range === "io") {
		ioReadBytes.value = []
		ioWriteBytes.value = []
		timeIODatas.value = []
	} else if (range === "all" || range === "network") {
		netBytesSents.value = []
		netBytesRecvs.value = []
		timeNetDatas.value = []
	}
	const res = await loadBaseInfo(searchInfo.ioOption, searchInfo.netOption)
	baseInfo.value = res.data

	const resData = res.data.currentInfo
	currentInfo.value.ioReadBytes = resData.ioReadBytes
	currentInfo.value.ioWriteBytes = resData.ioWriteBytes
	currentInfo.value.ioCount = resData.ioCount
	currentInfo.value.ioReadTime = resData.ioReadTime
	currentInfo.value.ioWriteTime = resData.ioWriteTime
	currentInfo.value.netBytesSent = resData.netBytesSent
	currentInfo.value.netBytesRecv = resData.netBytesRecv
	currentInfo.value.uptime = resData.uptime

	loadAppCurrentInfo()
	syncStatusCard()

	if (isInit) {
		startPollingTimer()
	}
}

function startPollingTimer() {
	if (timer) {
		clearInterval(timer)
		timer = null
	}
	const interval = lowPowerMode.value ? 30000 : 5000
	timer = setInterval(() => {
		if (isActive.value) {
			loadAppCurrentInfo()
		}
	}, interval)
}

async function loadAppCurrentInfo() {
	await Promise.all([onLoadCurrentInfo("gpu"), onLoadCurrentInfo("basic"), onLoadCurrentInfo("ioNet")])
	if (!isActive.value) {
		return
	}
	syncStatusCard()
}

async function onLoadCurrentInfo(scope: string) {
	const req = {
		scope,
		ioOption: searchInfo.ioOption,
		netOption: searchInfo.netOption
	}
	const res = await loadCurrentInfo(req)
	const resData = res.data

	if (scope === "ioNet") {
		let timeInterval = Number(res.data.uptime - currentInfo.value.uptime) || 3
		currentChartInfo.netBytesSent =
			res.data.netBytesSent - currentInfo.value.netBytesSent > 0
				? Number(((res.data.netBytesSent - currentInfo.value.netBytesSent) / 1024 / timeInterval).toFixed(2))
				: 0
		netBytesSents.value.push(currentChartInfo.netBytesSent)

		if (netBytesSents.value.length > 20) {
			netBytesSents.value.splice(0, 1)
		}

		currentChartInfo.netBytesRecv =
			res.data.netBytesRecv - currentInfo.value.netBytesRecv > 0
				? Number(((res.data.netBytesRecv - currentInfo.value.netBytesRecv) / 1024 / timeInterval).toFixed(2))
				: 0
		netBytesRecvs.value.push(currentChartInfo.netBytesRecv)
		if (netBytesRecvs.value.length > 20) {
			netBytesRecvs.value.splice(0, 1)
		}

		currentChartInfo.ioReadBytes =
			res.data.ioReadBytes - currentInfo.value.ioReadBytes > 0
				? Number(
						((res.data.ioReadBytes - currentInfo.value.ioReadBytes) / 1024 / 1024 / timeInterval).toFixed(2)
					)
				: 0
		ioReadBytes.value.push(currentChartInfo.ioReadBytes)
		if (ioReadBytes.value.length > 20) {
			ioReadBytes.value.splice(0, 1)
		}

		currentChartInfo.ioWriteBytes =
			res.data.ioWriteBytes - currentInfo.value.ioWriteBytes > 0
				? Number(
						((res.data.ioWriteBytes - currentInfo.value.ioWriteBytes) / 1024 / 1024 / timeInterval).toFixed(
							2
						)
					)
				: 0
		ioWriteBytes.value.push(currentChartInfo.ioWriteBytes)
		if (ioWriteBytes.value.length > 20) {
			ioWriteBytes.value.splice(0, 1)
		}
		currentChartInfo.ioCount = Math.round(Number((res.data.ioCount - currentInfo.value.ioCount) / timeInterval))
		let ioReadTime = res.data.ioReadTime - currentInfo.value.ioReadTime
		let ioWriteTime = res.data.ioWriteTime - currentInfo.value.ioWriteTime
		let ioChoose = ioReadTime > ioWriteTime ? ioReadTime : ioWriteTime
		currentChartInfo.ioTime = Math.round(Number(ioChoose / timeInterval))

		timeIODatas.value.push(dateFormatForSecond(res.data.shotTime))
		if (timeIODatas.value.length > 20) {
			timeIODatas.value.splice(0, 1)
		}
		timeNetDatas.value.push(dateFormatForSecond(res.data.shotTime))
		if (timeNetDatas.value.length > 20) {
			timeNetDatas.value.splice(0, 1)
		}

		currentInfo.value.ioReadBytes = resData.ioReadBytes
		currentInfo.value.ioWriteBytes = resData.ioWriteBytes
		currentInfo.value.ioCount = resData.ioCount
		currentInfo.value.ioReadTime = resData.ioReadTime
		currentInfo.value.ioWriteTime = resData.ioWriteTime

		currentInfo.value.netBytesSent = resData.netBytesSent
		currentInfo.value.netBytesRecv = resData.netBytesRecv
	}
	if (scope === "gpu") {
		currentInfo.value.gpuData = resData.gpuData
		currentInfo.value.xpuData = resData.xpuData
	}
	if (scope === "basic") {
		currentInfo.value.uptime = resData.uptime
		currentInfo.value.timeSinceUptime = resData.timeSinceUptime
		currentInfo.value.procs = resData.procs

		currentInfo.value.load1 = resData.load1
		currentInfo.value.load5 = resData.load5
		currentInfo.value.load15 = resData.load15
		currentInfo.value.loadUsagePercent = resData.loadUsagePercent

		currentInfo.value.cpuPercent = resData.cpuPercent
		currentInfo.value.cpuUsedPercent = resData.cpuUsedPercent
		currentInfo.value.cpuUsed = resData.cpuUsed
		currentInfo.value.cpuTotal = resData.cpuTotal

		currentInfo.value.memoryTotal = resData.memoryTotal
		currentInfo.value.memoryAvailable = resData.memoryAvailable
		currentInfo.value.memoryUsed = resData.memoryUsed
		currentInfo.value.memoryUsedPercent = resData.memoryUsedPercent

		currentInfo.value.swapMemoryTotal = resData.swapMemoryTotal
		currentInfo.value.swapMemoryAvailable = resData.swapMemoryAvailable
		currentInfo.value.swapMemoryUsed = resData.swapMemoryUsed
		currentInfo.value.swapMemoryUsedPercent = resData.swapMemoryUsedPercent

		currentInfo.value.timeSinceUptime = res.data.timeSinceUptime
		currentInfo.value.shotTime = resData.shotTime
		currentInfo.value.diskData = resData.diskData
	}
}

async function changeOption() {
	isInit.value = true
}

const ioOptions = ref()
const netOptions = ref()

async function onLoadNetworkOptions() {
	const res = await getNetworkOptions()
	netOptions.value = res.data
	searchInfo.netOption = netOptions.value && netOptions.value[0]
}

async function onLoadIOOptions() {
	const res = await getIOOptions()
	ioOptions.value = res.data
	searchInfo.ioOption = ioOptions.value && ioOptions.value[0]
}

async function loadPopularApps() {
	popularAppsLoading.value = true
	try {
		const res = await appsSearchAPI({
			page: 1,
			pageSize: 10,
			recommend: true
		})
		
		popularApps.value = (res.data as { items?: any[] })?.items || []
	} catch {
		popularApps.value = []
	} finally {
		popularAppsLoading.value = false
	}
}

onMounted(async () => {
	onLoadNetworkOptions()
	onLoadIOOptions()

	loadPopularApps()
	emitter.on("gopanel:lowPowerMode", handleLowPowerMode)
	onLoadBaseInfo(true, "all")
})

onUnmounted(() => {
	isActive.value = false
	if (timer) {
		clearInterval(timer as ReturnType<typeof setInterval>)
	}
	emitter.off("gopanel:lowPowerMode", handleLowPowerMode)
})
</script>
