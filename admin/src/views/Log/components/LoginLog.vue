<template>
  <div class="login-log-wrap">
    <div class="mb-4 flex justify-between items-center">
      <div class="flex gap-4">
        <n-input
          v-model:value="searchParams.ip"
          placeholder="搜索IP"
          clearable
          @keyup.enter="handleSearch"
          class="w-64"
        />
        <n-select
          v-model:value="searchParams.status"
          :options="statusOptions"
          placeholder="状态"
          clearable
          class="w-32"
          @update:value="handleSearch"
        />
        <n-button
          type="primary"
          @click="handleSearch"
        >搜索</n-button>
      </div>
      <div>
        <n-button
          type="error"
          ghost
          @click="handleClean"
        >清空</n-button>
      </div>
    </div>

    <n-data-table
      :columns="columns"
      :data="tableData"
      :loading="loading"
      :pagination="pagination"
      :row-key="(row) => row.id"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from "vue"
import { getLoginLogs, cleanLogs } from "@/api/modules/log"
import { Log } from "@/api/interface/log"
import { NTag, useMessage, useDialog } from "naive-ui"
import { formatTime } from "@/utils/date"

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const tableData = ref<Log.LoginLog[]>([])

const searchParams = reactive<Log.SearchLgLog>({
	page: 1,
	limit: 10,
	ip: "",
	status: ""
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

const columns = [
	{ title: "IP地址", key: "ip", width: 150 },
	{ title: "地理位置", key: "address", width: 150 },
	{ 
		title: "状态", 
		key: "status", 
		width: 100,
		render(row: Log.LoginLog) {
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
	{ title: "详情", key: "message", ellipsis: { tooltip: true as true } },
	{ title: "浏览器", key: "agent", ellipsis: { tooltip: true as true } },
	{ title: "登录时间", key: "createdAt", width: 180, render(row: Log.LoginLog) {
		return formatTime(row.createdAt)
	} }
]

const loadData = async () => {
	loading.value = true
	try {
		const res = await getLoginLogs(searchParams)
		tableData.value = res.data.items || []
		pagination.itemCount = res.data.total
	} catch (error) {
		console.error(error)
	} finally {
		loading.value = false
	}
}

const handleSearch = () => {
	searchParams.page = 1
	pagination.page = 1
	loadData()
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

const handleClean = () => {
	dialog.warning({
		title: "清空登录日志",
		content: "确定要清空所有登录日志吗？此操作不可恢复！",
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await cleanLogs({ logType: "login" })
				message.success("清空成功")
				loadData()
			} catch (error) {}
		}
	})
}

onMounted(() => {
	loadData()
})
</script>