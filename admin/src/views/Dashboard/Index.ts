import type { Dashboard } from "@/api/interface/dashboard"
import { ref, reactive } from 'vue'

const ioReadBytes = ref<Array<number>>([])
const ioWriteBytes = ref<Array<number>>([])
const netBytesSents = ref<Array<number>>([])
const netBytesRecvs = ref<Array<number>>([])
const timeIODatas = ref<Array<string>>([])
const timeNetDatas = ref<Array<string>>([])
const searchInfo = reactive({
	ioOption: "all",
	netOption: "all",
	scope: "all"
})

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
	ipv4Addr: "",
	systemProxy: "",
	cpuCores: 0,
	cpuLogicalCores: 0,
	cpuModelName: "",
	currentInfo: null
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
const currentChartInfo = reactive({
	ioReadBytes: 0,
	ioWriteBytes: 0,
	ioCount: 0,
	ioTime: 0,

	netBytesSent: 0,
	netBytesRecv: 0
})

const chartsOption = ref({ ioChart1: null, networkChart: null }) as any

export {
	ioReadBytes,
	ioWriteBytes,
	netBytesSents,
	netBytesRecvs,
	timeIODatas,
	timeNetDatas,
	searchInfo,
	baseInfo,
	currentChartInfo,
	currentInfo,
	chartsOption
}
