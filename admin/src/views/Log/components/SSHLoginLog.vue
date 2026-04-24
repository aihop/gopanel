<template>
	<div class="ssh-log-wrap">
		<div class="mb-4 flex justify-between items-center gap-4 flex-wrap">
			<div class="flex gap-4 flex-wrap">
				<n-input
					:value="searchParams.username"
					placeholder="搜索用户"
					clearable
					class="w-48"
					@update:value="value => searchParams.username = value || ''"
					@keyup.enter="handleSearch"
				/>
				<n-input
					:value="searchParams.ip"
					placeholder="搜索来源 IP"
					clearable
					class="w-56"
					@update:value="value => searchParams.ip = value || ''"
					@keyup.enter="handleSearch"
				/>
				<n-select
					:value="searchParams.status"
					:options="statusOptions"
					placeholder="状态"
					clearable
					class="w-32"
					@update:value="handleStatusChange"
				/>
				<n-button
					type="primary"
					@click="handleSearch"
				>
					搜索
				</n-button>
			</div>
			<n-button
				ghost
				@click="loadData"
			>
				刷新
			</n-button>
		</div>

		<n-alert
			v-if="warning"
			type="warning"
			:show-icon="true"
			class="mb-4"
		>
			{{ warning }}
		</n-alert>

		<n-data-table
			:columns="columns"
			:data="tableData"
			:loading="loading"
			:pagination="pagination"
			:row-key="rowKey"
			@update:page="handlePageChange"
			@update:page-size="handlePageSizeChange"
		/>
	</div>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref } from "vue"
import { NTag } from "naive-ui"
import { getSSHLoginLogs } from "@/api/modules/log"
import { Log } from "@/api/interface/log"
import { formatTime } from "@/utils/date"

const loading = ref(false)
const warning = ref("")
const tableData = ref<Log.SSHLoginLog[]>([])

const searchParams = reactive<Log.SearchSSHLog>({
	page: 1,
	limit: 10,
	ip: "",
	status: "",
	username: ""
})

const pagination = reactive({
	page: 1,
	limit: 10,
	itemCount: 0,
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100]
})

const statusOptions = [
	{ label: "成功", value: "Success" },
	{ label: "失败", value: "Failed" }
]

const rowKey = (row: Log.SSHLoginLog) => {
	return `${row.createdAt}-${row.username}-${row.sourceIp}-${row.sourcePort}-${row.status}`
}

const columns = [
	{
		title: "时间",
		key: "createdAt",
		width: 180,
		render(row: Log.SSHLoginLog) {
			return formatTime(row.createdAt)
		}
	},
	{ title: "用户", key: "username", width: 120 },
	{ title: "来源 IP", key: "sourceIp", width: 150 },
	{ title: "端口", key: "sourcePort", width: 90 },
	{
		title: "认证方式",
		key: "authMethod",
		width: 140
	},
	{
		title: "状态",
		key: "status",
		width: 100,
		render(row: Log.SSHLoginLog) {
			return h(
				NTag,
				{
					type: row.status === "Success" ? "success" : "error",
					bordered: false,
					size: "small"
				},
				{ default: () => row.status === "Success" ? "成功" : "失败" }
			)
		}
	},
	{
		title: "摘要",
		key: "message",
		ellipsis: { tooltip: true as true }
	},
	{
		title: "来源",
		key: "source",
		width: 140,
		ellipsis: { tooltip: true as true }
	}
]

const loadData = async () => {
	loading.value = true
	try {
		const res = await getSSHLoginLogs(searchParams)
		tableData.value = res.data.items || []
		pagination.itemCount = res.data.total || 0
		warning.value = res.data.warning || (!res.data.supported ? "当前平台暂不支持 SSH 登录日志采集" : "")
	} catch (error: any) {
		tableData.value = []
		pagination.itemCount = 0
		warning.value = error?.message || "SSH 登录日志读取失败"
	} finally {
		loading.value = false
	}
}

const handleSearch = () => {
	searchParams.page = 1
	pagination.page = 1
	loadData()
}

const handleStatusChange = (value: string | null) => {
	searchParams.status = value || ""
	handleSearch()
}

const handlePageChange = (page: number) => {
	searchParams.page = page
	pagination.page = page
	loadData()
}

const handlePageSizeChange = (limit: number) => {
	searchParams.limit = limit
	pagination.limit = limit
	searchParams.page = 1
	pagination.page = 1
	loadData()
}

onMounted(() => {
	loadData()
})
</script>
