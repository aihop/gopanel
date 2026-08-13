<template>
	<div class="dashboard-root grid grid-cols-1 gap-8 2xl:grid-cols-[minmax(0,1fr)_300px]">
		<StatusCard ref="statusRef">
			<DashboardMonitorCard @reload="onLoadBaseInfo(false, $event)" />
		</StatusCard>
	</div>
</template>

<script setup lang="ts">
import { loadBaseInfo, loadCurrentInfo } from "@/api/modules/dashboard"
import { dateFormatForSecond } from "@/utils/util"
import { onMounted, ref, onUnmounted } from "vue"
import StatusCard from "./components/StatusCard.vue"
import DashboardMonitorCard from "./components/DashboardMonitorCard.vue"
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

const statusRef = ref()

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

onMounted(async () => {
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

<style>
/* Dark mode overrides for Dashboard page */
.theme-dark .dashboard-root .text-slate-900,
.theme-dark .dashboard-root .text-slate-800 {
	color: var(--fg-default-color) !important;
}
.theme-dark .dashboard-root .text-slate-700 {
	color: var(--fg-secondary-color) !important;
}
.theme-dark .dashboard-root .text-slate-500,
.theme-dark .dashboard-root .text-slate-400 {
	color: var(--fg-secondary-color) !important;
}
.theme-dark .dashboard-root .border-slate-200 {
	border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .dashboard-root .bg-slate-50,
.theme-dark .dashboard-root .bg-slate-50\/80,
.theme-dark .dashboard-root .bg-slate-50\/60,
.theme-dark .dashboard-root .bg-slate-50\/90 {
	background-color: color-mix(in srgb, var(--bg-default-color) 95%, transparent) !important;
}
.theme-dark .dashboard-root .bg-white,
.theme-dark .dashboard-root .bg-white\/90,
.theme-dark .dashboard-root .bg-white\/85,
.theme-dark .dashboard-root .bg-white\/80 {
	background-color: var(--bg-default-color) !important;
}
.theme-dark .dashboard-root .hover\:bg-white:hover {
	background-color: var(--bg-default-color) !important;
}
.theme-dark .dashboard-root .hover\:border-blue-300:hover {
	border-color: color-mix(in srgb, var(--primary-color) 50%, transparent) !important;
}
.theme-dark .dashboard-root [class*="border-\[rgba\(147\,197\,253"] {
	border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .dashboard-root [class*="bg-white\/"] {
	background-color: var(--bg-default-color) !important;
}
</style>
