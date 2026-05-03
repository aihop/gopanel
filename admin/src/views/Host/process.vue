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
        <ProcessSearchToolbar
          mode="process"
          :process-search="processSearch"
          :network-search="networkSearch"
          @update:process-pid="processSearch.pid = $event"
          @update:process-name="processSearch.name = $event"
          @update:process-username="processSearch.username = $event"
          @search="handleSearch"
          @reset="handleProcessReset"
        />
        <n-data-table
          :columns="processColumns"
          :data="processData"
          :loading="loading"
          :pagination="processWsReady ? processPagination : false"
          :bordered="false"
          :scroll-x="900"
        />
      </n-tab-pane>
      <n-tab-pane
        name="network"
        tab="网络"
      >
        <ProcessSearchToolbar
          mode="network"
          :process-search="processSearch"
          :network-search="networkSearch"
          @update:network-process-id="networkSearch.processID = $event"
          @update:network-process-name="networkSearch.processName = $event"
          @update:network-port="networkSearch.port = $event"
          @search="handleNetworkSearch"
          @reset="handleNetworkReset"
        />
        <n-data-table
          :columns="networkColumns"
          :data="networkData"
          :pagination="networkPagination"
          :bordered="false"
        />
      </n-tab-pane>
    </n-tabs>

    <ProcessDetailDrawer
      :show="showDetailDrawer"
      :detail-drawer-title="detailDrawerTitle"
      :selected-process="selectedProcess"
      :get-status-type="getStatusType"
      :open-files-columns="openFilesColumns"
      :drawer-network-connections-columns="drawerNetworkConnectionsColumns"
      @update:show="showDetailDrawer = $event"
    />
  </div>
</template>

<script setup lang="ts">
import type { PaginationProps } from "naive-ui"
import { ProcessList, StopProcess } from "@/api/modules/process"
import { useAuthStore } from "@/store/auth"
import { MsgSuccess } from "@/utils/message"
import { t } from "@/i18n"
import { useDialog } from "naive-ui"
import { computed, onMounted, onUnmounted, reactive, ref } from "vue"
import ProcessSearchToolbar from "./components/ProcessSearchToolbar.vue"
import ProcessDetailDrawer from "./components/ProcessDetailDrawer.vue"
import {
	createDrawerNetworkConnectionsColumns,
	createNetworkColumns,
	createProcessColumns,
	normalizeProcessRows,
	openFilesColumns,
	type ProcessData,
	type ProcessStatusTagType
} from "./components/processColumns"

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

async function loadInitialProcessList() {
	loading.value = true
	processWsReady.value = false
	try {
		const res = await ProcessList(buildProcessSearchData())
		const list = Array.isArray(res.data) ? res.data : []
		oldData.value = list
		processData.value = normalizeProcessRows(list)
		processPagination.value.itemCount = list.length
	} finally {
		loading.value = false
	}
}

function openDetailDrawer(row: any) {
	selectedProcess.value = row
	showDetailDrawer.value = true
}

function getStatusType(status: string | undefined): ProcessStatusTagType {
	if (status === "睡眠" || status === "ESTABLISHED" || status === "info") return "info"
	if (status === "空闲" || status === "LISTEN" || status === "success") return "success"
	if (status === "NONE" || status === "warning" || status === "CLOSE_WAIT") return "warning"
	if (status === "运行中") return "error"
	return "info"
}

const processColumns = createProcessColumns(getStatusType, openDetailDrawer, handleStopProcess)

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
			processPagination.value.itemCount = newData.length
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
	processPagination.value.page = 1
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

const networkColumns = createNetworkColumns(getStatusType)
const drawerNetworkConnectionsColumns = createDrawerNetworkConnectionsColumns(getStatusType)

const networkData = ref<any[]>([])

const processPagination = ref<PaginationProps>({
	page: 1,
	pageSize: 10,
	itemCount: 0,
	pageSizes: [10, 20, 50, 100],
	showSizePicker: true,
	onChange: (page: number) => {
		processPagination.value.page = page
	},
	onUpdatePageSize: (pageSize: number) => {
		processPagination.value.pageSize = pageSize
		processPagination.value.page = 1
	}
})

const networkPagination = ref<PaginationProps>({
	page: 1,
	pageSize: 10,
	itemCount: 0,
	pageSizes: [10, 20, 50, 100],
	showSizePicker: true,
	onChange: (page: number) => {
		networkPagination.value.page = page
	},
	onUpdatePageSize: (pageSize: number) => {
		networkPagination.value.pageSize = pageSize
		networkPagination.value.page = 1
	}
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
	networkPagination.value.page = 1
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
			networkPagination.value.itemCount = newData.length
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
