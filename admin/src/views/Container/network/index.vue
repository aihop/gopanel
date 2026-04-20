<template>
  <div class="py-4">
    <!-- Header with Action Buttons -->
    <n-space class="mb-4">
      <n-button
        type="primary"
        @click="showCreateNetworkDrawer = true"
      >{{ $t('container.createNetwork') }}</n-button>
      <n-button @click="handleCleanNetworks">{{ $t('container.cleanNetwork') }}</n-button>
      <n-button
        type="error"
        @click="handleBulkDelete"
        :disabled="checkedRowKeys.length === 0"
      >删除</n-button>
    </n-space>

    <!-- Network List Section -->
    <n-card>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">网络</h2>
        <n-space>
          <!-- <n-button>列表设置</n-button> -->
          <n-input
            placeholder="搜索"
            clearable
            @update:value="handleSearch"
          >
            <template #suffix>
              <n-icon name="search" />
            </template>
          </n-input>
        </n-space>
      </div>

      <!-- Networks Table -->
      <n-data-table
        :columns="columns"
        :data="networkData"
        :pagination="pagination"
        :row-key="rowKey"
        v-model:checked-row-keys="checkedRowKeys"
        :bordered="false"
        :loading="loading"
        :row-class-name="row => (isSystem(row.name) ? 'system-network' : '')"
      />
    </n-card>

    <!-- Create Network Component -->
    <create-network
      v-model:show="showCreateNetworkDrawer"
      @success="fetchNetworkList"
    />
  </div>
</template>

<script setup lang="ts">
import { h, ref, computed, onMounted } from "vue"
import { NButton, NSpace, NTag, NText, NInput, NIcon, NDataTable, NCard, useDialog, useMessage } from "naive-ui"
import type { DataTableColumns, DataTableRowKey } from "naive-ui"
import { containerNetworkListAPI, deleteNetwork, containerPrune } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"
import CreateNetwork from "./create/index.vue"
import { computeSize } from "@/utils/util"

// 搜索参数
const searchParams = ref({
	info: "",
	page: 1,
	limit: 10
})

// 表格数据
const networkData = ref<Container.NetworkInfo[]>([])
const loading = ref(false)
const message = useMessage()
const dialog = useDialog()

// 获取网络列表数据
const fetchNetworkList = async () => {
	loading.value = true
	const res = await containerNetworkListAPI(searchParams.value)
	if (res.code === 0) {
		networkData.value = res.data.items
		pagination.value.itemCount = res.data.total
	}
	loading.value = false
}

// 搜索处理
const handleSearch = (value: string) => {
	searchParams.value.info = value
	searchParams.value.page = 1 // 重置到第一页
	fetchNetworkList()
}

// 分页处理
const pagination = ref({
	page: 1,
	limit: 10,
	showSizePicker: true,
	pageSizes: [10, 20, 50],
	itemCount: 0,
	onChange: (page: number) => {
		searchParams.value.page = page
		fetchNetworkList()
	},
	onUpdatePageSize: (limit: number) => {
		searchParams.value.limit = limit
		searchParams.value.page = 1
		fetchNetworkList()
	}
})

// 判断是否为系统网络
const isSystem = (val: string) => {
	return val === "bridge" || val === "none" || val === "host" || val === "gopanel-network"
}

// 表格列定义
const createColumns = ({
	deleteRow
}: {
	deleteRow: (row: Container.NetworkInfo) => void
}): DataTableColumns<Container.NetworkInfo> => [
	{
		type: "selection",
		disabled: row => isSystem(row.name)
	},
	{
		title: "名称",
		key: "name",
		render(row) {
			return h("div", { class: "flex items-center gap-2" }, [
				h(NText, { type: "primary", class: "cursor-pointer hover:underline" }, { default: () => row.name }),
				isSystem(row.name) && h(NTag, { type: "info", size: "small" }, { default: () => "system" })
			])
		}
	},
	{
		title: "驱动",
		key: "driver",
		render(row) {
			const type = row.driver === "null" ? "info" : "default"
			return h(NTag, { type, bordered: false, size: "small" }, { default: () => row.driver })
		}
	},
	{
		title: "子网",
		key: "subnet",
		render(row) {
			return row.subnet || h(NText, { depth: 3 }, { default: () => "" })
		}
	},
	{
		title: "网关",
		key: "gateway",
		render(row) {
			return row.gateway || h(NText, { depth: 3 }, { default: () => "" })
		}
	},
	{
		title: "标签",
		key: "labels",
		render(row) {
			if (!row.labels || row.labels.length === 0) {
				return h(NText, { depth: 3 }, { default: () => "" })
			}
			return h(
				NSpace,
				{ vertical: true, size: "small" },
				{
					default: () =>
						row.labels.map(label =>
							h(
								NTag,
								{
									bordered: false,
									size: "small",
									style: {
										maxWidth: "200px",
										overflow: "hidden",
										textOverflow: "ellipsis",
										whiteSpace: "nowrap"
									}
								},
								{ default: () => label }
							)
						)
				}
			)
		}
	},
	{
		title: "创建时间",
		key: "createdAt",
		render(row) {
			return new Date(row.createdAt).toLocaleString()
		}
	},
	{
		title: "操作",
		key: "actions",
		render(row) {
			return h(
				NButton,
				{
					text: true,
					type: "error",
					onClick: () => deleteRow(row),
					disabled: isSystem(row.name)
				},
				{ default: () => "删除" }
			)
		}
	}
]

// 删除单行
const handleDeleteRow = async (row: Container.NetworkInfo) => {
	if (isSystem(row.name)) {
		message.warning("系统网络不允许删除")
		return
	}

	dialog.warning({
		title: "删除网络",
		content: `确定要删除网络 "${row.name}" 吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await deleteNetwork({ names: [row.name] })
				message.success("删除成功")
				fetchNetworkList()
			} catch (error) {
				message.error("删除失败")
				console.error("删除失败:", error)
			}
		}
	})
}

const columns = createColumns({ deleteRow: handleDeleteRow })

// 批量删除
const handleBulkDelete = () => {
	if (checkedRowKeys.value.length === 0) return

	// 检查选中的网络是否包含系统网络
	const selectedNetworks = networkData.value.filter(item => checkedRowKeys.value.includes(item.id))
	const hasSystemNetwork = selectedNetworks.some(item => isSystem(item.name))

	if (hasSystemNetwork) {
		message.warning("选中的网络包含系统网络，系统网络不允许删除")
		return
	}

	dialog.warning({
		title: "批量删除",
		content: `确定要删除选中的 ${checkedRowKeys.value.length} 个网络吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const names = selectedNetworks.map(item => item.name)
				await deleteNetwork({ names })
				message.success("删除成功")
				checkedRowKeys.value = []
				fetchNetworkList()
			} catch (error) {
				message.error("删除失败")
				console.error("删除失败:", error)
			}
		}
	})
}

// 清理网络
const handleCleanNetworks = () => {
	dialog.warning({
		title: "清理网络",
		content: "清理网络将删除所有未被使用的网络，该操作无法回滚，是否继续？",
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const res = await containerPrune({
					pruneType: "network",
					withTagAll: false
				})
				message.success(
					`清理成功，共删除 ${res.data.deletedNumber} 个网络，释放空间 ${computeSize(res.data.spaceReclaimed)}`
				)
				fetchNetworkList()
			} catch (error) {
				message.error("清理失败")
				console.error("清理失败:", error)
			}
		}
	})
}

// 创建网络抽屉状态
const showCreateNetworkDrawer = ref(false)

// 初始化
onMounted(() => {
	fetchNetworkList()
})

const checkedRowKeys = ref<DataTableRowKey[]>([])
const rowKey = (row: Container.NetworkInfo) => row.id
</script>

<style scoped>
.py-4 {
	padding-top: 1rem;
	padding-bottom: 1rem;
}

:deep(.system-network) {
	background-color: var(--n-table-color-striped);
}
</style>
