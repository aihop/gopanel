<template>
  <div class="mt-2 rounded-[28px] bg-base-100 px-8 py-4">
    <n-tabs
      v-model:value="activeTab"
      type="line"
      size="large"
      animated
      @update:value="handleTabChange"
    >
      <n-tab-pane
        name="process"
        tab="进程"
      >
        <div
          class="search-inputs flex items-center"
          style="gap: 16px; margin-bottom: 16px"
        >
          <n-input
            v-model:value="processSearch.pid"
            style="width: 150px"
            placeholder="进程ID"
            class="search-input"
          />
          <n-input
            v-model:value="processSearch.name"
            style="width: 150px"
            placeholder="名称"
            class="search-input"
          />
          <n-input
            v-model:value="processSearch.username"
            style="width: 150px"
            placeholder="用户"
            class="search-input"
          />
          <n-button
            type="primary"
            class="search-button"
            @click="handleSearch"
          >
            <template #icon>
              <Icon
                name="ion:search-outline"
                :size="18"
              />
            </template>
            搜索
          </n-button>
          <n-button
            class="arrange-button"
            @click="handleProcessReset"
          >
            <template #icon>
              <Icon
                name="ion:settings-outline"
                :size="18"
              />
            </template>
            重置
          </n-button>
        </div>
        <n-data-table
          :columns="processColumns"
          :data="processData"
          :loading="loading"
          :pagination="processWsReady ? pagination : false"
          :bordered="false"
          :scroll-x="900"
        />
      </n-tab-pane>
      <n-tab-pane
        name="network"
        tab="网络"
      >
        <div
          class="search-inputs flex items-center"
          style="gap: 16px; margin-bottom: 16px"
        >
          <n-input
            v-model:value="networkSearch.processID"
            style="width: 150px"
            placeholder="进程ID"
            class="search-input"
          />
          <n-input
            v-model:value="networkSearch.processName"
            style="width: 150px"
            placeholder="进程名称"
            class="search-input"
          />
          <n-input
            v-model:value="networkSearch.port"
            style="width: 150px"
            placeholder="端口"
            class="search-input"
          />
          <n-button
            type="primary"
            class="search-button"
            @click="handleNetworkSearch"
          >
            <template #icon>
              <Icon
                name="ion:search-outline"
                :size="18"
              />
            </template>
            搜索
          </n-button>
          <n-button
            class="arrange-button"
            @click="handleNetworkReset"
          >
            <template #icon>
              <Icon
                name="ion:settings-outline"
                :size="18"
              />
            </template>
            重置
          </n-button>
        </div>
        <n-data-table
          :columns="networkColumns"
          :data="networkData"
          :pagination="pagination"
          :bordered="false"
        />
      </n-tab-pane>
    </n-tabs>

    <n-drawer
      v-model:show="showDetailDrawer"
      :width="500"
      placement="right"
    >
      <n-drawer-content
        :title="detailDrawerTitle"
        closable
      >
        <n-tabs
          type="line"
          animated
        >
          <n-tab-pane
            name="basicInfo"
            tab="基本信息"
          >
            <n-descriptions
              label-placement="left"
              bordered
              :column="1"
              size="small"
            >
              <n-descriptions-item label="名称">{{ selectedProcess?.name }}</n-descriptions-item>
              <n-descriptions-item label="状态">
                <n-tag
                  :type="getStatusType(selectedProcess?.status)"
                  size="small"
                >
                  {{ selectedProcess?.status }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item label="进程ID">{{ selectedProcess?.PID }}</n-descriptions-item>
              <n-descriptions-item label="父进程ID">{{ selectedProcess?.PPID }}</n-descriptions-item>
              <n-descriptions-item label="线程">{{ selectedProcess?.numThreads }}</n-descriptions-item>
              <n-descriptions-item label="连接">
                {{ selectedProcess?.numConnections ?? "N/A" }}
              </n-descriptions-item>
              <n-descriptions-item label="磁盘读">
                {{ selectedProcess?.diskRead ?? "N/A" }}
              </n-descriptions-item>
              <n-descriptions-item label="磁盘写">
                {{ selectedProcess?.diskWrite ?? "N/A" }}
              </n-descriptions-item>
              <n-descriptions-item label="用户">{{ selectedProcess?.username }}</n-descriptions-item>
              <n-descriptions-item label="启动时间">{{ selectedProcess?.startTime }}</n-descriptions-item>
              <n-descriptions-item label="启动命令">
                {{ selectedProcess?.cmdLine ?? "N/A" }}
              </n-descriptions-item>
            </n-descriptions>
          </n-tab-pane>
          <n-tab-pane
            name="memoryInfo"
            tab="内存信息"
          >
            <n-descriptions
              v-if="selectedProcess?.memoryInfo"
              label-placement="left"
              bordered
              :columns="2"
              size="small"
            >
              <n-descriptions-item label="rss">{{ selectedProcess.memoryInfo.rss }}</n-descriptions-item>
              <n-descriptions-item label="swap">
                {{ selectedProcess.memoryInfo.swap }}
              </n-descriptions-item>
              <n-descriptions-item label="vms">{{ selectedProcess.memoryInfo.vms }}</n-descriptions-item>
              <n-descriptions-item label="hwm">{{ selectedProcess.memoryInfo.hwm }}</n-descriptions-item>
              <n-descriptions-item label="data">
                {{ selectedProcess.memoryInfo.data }}
              </n-descriptions-item>
              <n-descriptions-item label="stack">
                {{ selectedProcess.memoryInfo.stack }}
              </n-descriptions-item>
              <n-descriptions-item label="locked">
                {{ selectedProcess.memoryInfo.locked }}
              </n-descriptions-item>
            </n-descriptions>
            <p v-else>暂无内存信息</p>
          </n-tab-pane>
          <n-tab-pane
            name="fileOpen"
            tab="文件打开"
          >
            <n-data-table
              v-if="selectedProcess?.openFiles?.length"
              :columns="openFilesColumns"
              :data="selectedProcess.openFiles"
              :pagination="false"
              :bordered="true"
              size="small"
            />
            <p v-else>暂无打开文件信息</p>
          </n-tab-pane>
          <n-tab-pane
            name="envVar"
            tab="环境变量"
          >
            <n-code
              v-if="selectedProcess?.environmentVariables"
              language="js"
              :code="selectedProcess.environmentVariables"
              show-line-numbers
            />
            <p v-else>暂无环境变量信息</p>
          </n-tab-pane>
          <n-tab-pane
            name="networkLink"
            tab="网络连接"
          >
            <n-data-table
              v-if="selectedProcess?.connects?.length"
              :columns="drawerNetworkConnectionsColumns"
              :data="selectedProcess.connects"
              :pagination="false"
              :bordered="true"
              size="small"
            />
            <p v-else>暂无网络连接信息</p>
          </n-tab-pane>
        </n-tabs>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns } from "naive-ui"
import { ProcessList, StopProcess } from "@/api/modules/process"
import { useAuthStore } from "@/store/auth"
import { MsgSuccess } from "@/utils/message"
import { t } from "@/i18n"
import {
	NButton,
	NCard,
	NCode,
	NDataTable,
	NDescriptions,
	NDescriptionsItem,
	NDrawer,
	NDrawerContent,
	NInput,
	NSpace,
	NTabPane,
	NTabs,
	NTag
} from "naive-ui"
import { computed, h, onMounted, onUnmounted, reactive, ref } from "vue"

const showDetailDrawer = ref(false)
const selectedProcess = ref<any>(null)
const processSearch = reactive({
	type: "ps",
	pid: "",
	username: "",
	name: ""
})

const loading = ref(false)
let processSocket = ref<WebSocket | null>(null)
let processInterval: any = null
const oldData = ref<any[]>([])
const isGetData = ref(true)
const processWsReady = ref(false)
const dialog = useDialog()

const detailDrawerTitle = computed(() => {
	return selectedProcess.value ? `详情 - ${selectedProcess.value.name}` : "详情"
})

function buildProcessSearchData() {
	return {
		type: processSearch.type,
		pid: processSearch.pid ? Number.parseInt(processSearch.pid) : undefined,
		username: processSearch.username,
		name: processSearch.name
	}
}

function normalizeProcessRows(items: ProcessData[]) {
	return items.map((item: ProcessData) => ({
		...item,
		memoryInfo: {
			rss: item.rss,
			swap: item.swap,
			vms: item.vms,
			hwm: item.hwm,
			data: item.data,
			stack: item.stack,
			locked: item.locked
		},
		environmentVariables: item.envs?.join("\n") || ""
	}))
}

async function loadInitialProcessList() {
	loading.value = true
	processWsReady.value = false
	try {
		const res = await ProcessList(buildProcessSearchData())
		const list = Array.isArray(res.data) ? res.data : []
		oldData.value = list
		processData.value = normalizeProcessRows(list)
	} finally {
		loading.value = false
	}
}

function openDetailDrawer(row: any) {
	selectedProcess.value = row
	showDetailDrawer.value = true
}

function getStatusType(status: string | undefined) {
	if (status === "睡眠" || status === "ESTABLISHED" || status === "info") return "info"
	if (status === "空闲" || status === "LISTEN" || status === "success") return "success"
	if (status === "NONE" || status === "warning" || status === "CLOSE_WAIT") return "warning"
	if (status === "运行中") return "error"
	return "info"
}

interface ProcessMemoryInfo {
	rss: string
	swap: string
	vms: string
	hwm: string
	data: string
	stack: string
	locked: string
}

interface ProcessOpenFile {
	path: string
	fd: number
}

interface ProcessConnection {
	type: string
	status: string
	localaddr: {
		ip: string
		port: number
	}
	remoteaddr: {
		ip: string
		port: number
	}
	PID: number
	name: string
}

interface ProcessData {
	PID: number
	name: string
	PPID: number
	username: string
	status: string
	startTime: string
	numThreads: number
	numConnections: number
	cpuPercent: string
	diskRead: string
	diskWrite: string
	cmdLine: string
	rss: string
	vms: string
	hwm: string
	data: string
	stack: string
	locked: string
	swap: string
	cpuValue: number
	rssValue: number
	envs: string[]
	openFiles: ProcessOpenFile[]
	connects: ProcessConnection[]
	memoryInfo?: ProcessMemoryInfo
}

// 修复表格列类型
const processColumns: DataTableColumns<ProcessData> = [
	{ title: "PID", key: "PID", sorter: true },
	{ title: "名称", key: "name", sorter: true },
	{ title: "父进程ID", key: "PPID", sorter: true },
	{ title: "线程", key: "numThreads" },
	{ title: "用户", key: "username" },
	{
		title: "CPU",
		key: "cpuPercent",
		sorter: (row1, row2) => row1.cpuValue - row2.cpuValue
	},
	{
		title: "内存",
		key: "rss",
		sorter: (row1, row2) => row1.rssValue - row2.rssValue
	},
	{ title: "连接", key: "numConnections" },
	{
		title: "状态",
		key: "status",
		render(row) {
			return h(NTag, { type: getStatusType(row.status) }, { default: () => row.status })
		},
		filter: true,
		filterOptions: [
			{ label: "运行中", value: "running" },
			{ label: "睡眠", value: "sleep" },
			{ label: "停止", value: "stop" },
			{ label: "空闲", value: "idle" },
			{ label: "等待", value: "wait" },
			{ label: "锁定", value: "lock" },
			{ label: "僵尸", value: "zombie" }
		]
	},
	{ title: "启动时间", key: "startTime" },
	{
		title: "操作",
		key: "actions",
		fixed: "right",
		width: 150,
		render(row) {
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{
							strong: true,
							tertiary: true,
							type: "primary",
							ghost: true,
							size: "small",
							onClick: () => openDetailDrawer(row)
						},
						{ default: () => "详情" }
					),
					h(
						NButton,
						{
							strong: true,
							tertiary: true,
							type: "error",
							size: "small",
							onClick: () => handleStopProcess(row)
						},
						{ default: () => "结束" }
					)
				]
			})
		}
	}
]

// WebSocket相关逻辑
function isWsOpen(ws: WebSocket | null) {
	return ws?.readyState === 1
}

function closeSocket(wsRef: typeof processSocket, intervalRef: any) {
	if (isWsOpen(wsRef.value)) {
		wsRef.value?.close()
	}
	if (intervalRef) {
		clearInterval(intervalRef)
	}
}

function initProcess() {
	const authStore = useAuthStore()
	const auth = authStore.auth || ""

	const href = window.location.href
	const protocol = href.split("//")[0] === "http:" ? "ws" : "wss"
	const ipLocal = href.split("//")[1].split("/")[0]
	const url = `${protocol}://${ipLocal}/api/process/ws?auth=${auth}`
	processSocket.value = new WebSocket(url)

	processSocket.value.onopen = () => {
		isGetData.value = true
		processSocket.value?.send(JSON.stringify(buildProcessSearchData()))
	}

	processSocket.value.onmessage = message => {
		isGetData.value = false
		const newData = JSON.parse(message.data)
		if (Array.isArray(newData) && (newData.length === 0 || newData[0].hasOwnProperty("PID"))) {
			processWsReady.value = true
			oldData.value = newData
			processData.value = normalizeProcessRows(newData)
			loading.value = false
		}
	}

	processSocket.value.onerror = () => {
		console.error("进程 WebSocket 连接错误")
	}

	processSocket.value.onclose = () => {
		console.log("进程 WebSocket 连接已关闭")
	}

	processInterval = setInterval(() => {
		if (isWsOpen(processSocket.value) && !isGetData.value) {
			isGetData.value = true
			processSocket.value?.send(JSON.stringify(buildProcessSearchData()))
		}
	}, 5000)
}

async function handleStopProcess(row: any) {
	await dialog.info({
		title: "您确定要定制进程吗？",
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async() => {
			try {
				await StopProcess({ PID: row.PID })
				MsgSuccess("进程已停止")
			} catch (error) {
			}
		}
	})
}

async function handleSearch() {
	await loadInitialProcessList()
	if (isWsOpen(processSocket.value) && !isGetData.value) {
		isGetData.value = true
		processSocket.value?.send(JSON.stringify(buildProcessSearchData()))
	}
}

// 进程重置事件
function handleProcessReset() {
	processSearch.pid = ""
	processSearch.username = ""
	processSearch.name = ""
	void handleSearch()
}

onMounted(() => {
	void loadInitialProcessList()
	initProcess()
})

onUnmounted(() => {
	closeSocket(processSocket, processInterval)
	closeSocket(networkSocket, networkInterval)
})

const processData = ref<ProcessData[]>([])

// Network table columns definition
const networkColumns = [
	{ title: "类型", key: "type", sorter: true },
	{ title: "PID", key: "PID", sorter: true },
	{ title: "进程名称", key: "name", sorter: true },
	{
		title: "本地地址/端口",
		key: "localaddr",
		render(row: any) {
			if (row.localaddr && row.localaddr.port) {
				return `${row.localaddr.ip}:${row.localaddr.port}`
			}
			return row.localaddr?.ip || ""
		}
	},
	{
		title: "远程地址/端口",
		key: "remoteaddr",
		render(row: any) {
			if (row.remoteaddr && row.remoteaddr.port) {
				return `${row.remoteaddr.ip}:${row.remoteaddr.port}`
			}
			return row.remoteaddr?.ip || ""
		}
	},
	{
		title: "状态",
		key: "status",
		sorter: true,
		render(row: any) {
			return h(NTag, { type: getStatusType(row.status) }, { default: () => row.status })
		}
	}
]

// Columns for Open Files in Drawer
const openFilesColumns = [
	{ title: "文件", key: "path" },
	{ title: "fd", key: "fd", width: 60 }
]

// Columns for Network Connections in Drawer
const drawerNetworkConnectionsColumns = [
	{
		title: "本地地址/端口",
		key: "localaddr",
		render(row: any) {
			if (row.localaddr.port) {
				return `${row.localaddr.ip}:${row.localaddr.port}`
			}
			return row.localaddr.ip
		}
	},
	{
		title: "远程地址/端口",
		key: "remoteaddr",
		render(row: any) {
			if (row.remoteaddr.port) {
				return `${row.remoteaddr.ip}:${row.remoteaddr.port}`
			}
			return row.remoteaddr.ip
		}
	},
	{
		title: "状态",
		key: "status",
		width: 100,
		render(row: any) {
			return h(NTag, { type: getStatusType(row.status), size: "small" }, { default: () => row.status })
		}
	}
]

const networkData = ref<any[]>([])

const pagination = ref({
	pageSize: 10
})

// 新增：网络数据 WebSocket 相关逻辑
const networkSocket = ref<WebSocket | null>(null)
let networkInterval: any = null

// 网络搜索参数
const networkSearch = reactive({
	processID: "",
	processName: "",
	port: ""
})

// 网络搜索事件
function handleNetworkSearch() {
	if (isWsOpen(networkSocket.value)) {
		const searchData: any = {
			type: "net"
		}
		if (networkSearch.processID) searchData.processID = Number(networkSearch.processID)
		if (networkSearch.processName) searchData.processName = networkSearch.processName
		if (networkSearch.port) searchData.port = Number(networkSearch.port)
		networkSocket.value?.send(JSON.stringify(searchData))
	}
}

// 网络重置事件
function handleNetworkReset() {
	networkSearch.processID = ""
	networkSearch.processName = ""
	networkSearch.port = ""
	handleNetworkSearch()
}

function initNetwork() {
	const authStore = useAuthStore()
	const auth = authStore.auth || ""

	const href = window.location.href
	const protocol = href.split("//")[0] === "http:" ? "ws" : "wss"
	const ipLocal = href.split("//")[1].split("/")[0]
	const url = `${protocol}://${ipLocal}/api/process/ws?auth=${auth}`
	networkSocket.value = new WebSocket(url)

	networkSocket.value.onopen = () => {
		const searchData: any = {
			type: "net"
		}
		if (networkSearch.processID) searchData.processID = Number(networkSearch.processID)
		if (networkSearch.processName) searchData.processName = networkSearch.processName
		if (networkSearch.port) searchData.port = Number(networkSearch.port)
		networkSocket.value?.send(JSON.stringify(searchData))
	}

	networkSocket.value.onmessage = message => {
		const newData = JSON.parse(message.data)
		if (Array.isArray(newData) && newData.length && newData[0].hasOwnProperty("type")) {
			// 直接赋值即可，字段已适配
			networkData.value = newData
		}
	}

	networkSocket.value.onerror = () => {
		console.error("网络 WebSocket 连接错误")
	}

	networkSocket.value.onclose = () => {
		console.log("网络 WebSocket 连接已关闭")
	}

	networkInterval = setInterval(() => {
		if (isWsOpen(networkSocket.value)) {
			const searchData: any = {
				type: "net"
			}
			if (networkSearch.processID) searchData.processID = Number(networkSearch.processID)
			if (networkSearch.processName) searchData.processName = networkSearch.processName
			if (networkSearch.port) searchData.port = Number(networkSearch.port)
			networkSocket.value?.send(JSON.stringify(searchData))
		}
	}, 5000)
}

// 新增：tab切换处理
const activeTab = ref("process")

function handleTabChange(tabName: string) {
	activeTab.value = tabName
	if (tabName === "process") {
		closeSocket(networkSocket, networkInterval)
		void loadInitialProcessList()
		initProcess()
	} else if (tabName === "network") {
		closeSocket(processSocket, processInterval)
		initNetwork()
	}
}
</script>

<style scoped>
/* 如果需要，可以在这里添加 Tailwind CSS 类或自定义样式 */
.n-descriptions-item-label {
	width: 80px; /* Adjust as needed */
}
</style>
