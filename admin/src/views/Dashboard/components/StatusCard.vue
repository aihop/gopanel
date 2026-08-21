<template>
	<div class="contents">
		<DashboardInfoStrip
			class="bg-base-accent border-base-accent 2xl:col-span-2"
			:base-info="baseInfo"
			:current-info="currentInfo"
			:low-power-mode="lowPowerMode"
			:memory-cleaning="memoryCleaning"
			:cpu-relieving="cpuRelieving"
			@set-low-power-mode="setLowPowerMode"
			@memory-clean="handleMemoryClean"
			@cpu-relieve="handleCpuRelieve"
		/>

		<DashboardAIControl>
			<DashboardResourceCards
				class="bg-base-100 border-base-accent"
				:base-info="baseInfo"
				:current-info="currentInfo"
				:cpu-show-all="cpuShowAll"
				:memory-cleaning="memoryCleaning"
				@toggle-cpu-show-all="cpuShowAll = $event"
				@memory-clean="handleMemoryClean"
			/>

			<DashboardStoragePanels
				:current-info="currentInfo"
				:visible-disk-data="visibleDiskData"
				:has-more-disks="hasMoreDisks"
				:disk-expanded="diskExpanded"
				:accelerator-list="acceleratorList"
				:visible-accelerators="visibleAccelerators"
				:has-more-accelerators="hasMoreAccelerators"
				:accelerator-expanded="acceleratorExpanded"
				@toggle-disk-expanded="toggleDiskExpanded"
				@toggle-accelerator-expanded="toggleAcceleratorExpanded"
				@go-gpu="goGPU"
			/>

			<slot />
		</DashboardAIControl>
	</div>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { useRouter } from "vue-router"
import { computed, onMounted, onUnmounted, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import DashboardInfoStrip from "./DashboardInfoStrip.vue"
import DashboardAIControl from "./DashboardAIControl.vue"
import DashboardResourceCards from "./DashboardResourceCards.vue"
import DashboardStoragePanels from "./DashboardStoragePanels.vue"
import emitter from "@/utils/emitter"
import { clearMemoryCaches, relieveCPU } from "@/api/modules/host"
import { parseUtil } from "./dashboardStatusHelpers"

const router = useRouter()
const message = useMessage()
const { t } = useI18n()

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
	return val
}

const goGPU = () => {
	router.push({ name: "GPU" })
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
			message.warning(data.message || t("home.memoryCleanPrivilegeRequired"))
		} else if (data.message) {
			message.success(data.message)
		} else {
			message.success(t("home.memoryCleanTriggered"))
		}
	} catch (err: any) {
		message.error(err?.message || t("home.memoryCleanFailed"))
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
		message.success(data.message || t("home.cpuRelieved"))
	} catch (err: any) {
		message.error(err?.message || t("home.cpuRelieveFailed"))
	} finally {
		cpuRelieving.value = false
	}
}

defineExpose({
	acceptParams
})
</script>
