<template>
  <div class="py-4">
    <!-- Header with Action Buttons -->
    <n-space class="mb-4">
      <n-button
        type="primary"
        @click="showCreateVolumeDrawer = true"
      >创建存储卷</n-button>
      <n-button @click="handleCleanVolumes">清理存储卷</n-button>
      <n-button
        type="error"
        @click="handleBulkDelete"
        :disabled="checkedRowKeys.length === 0"
      >删除</n-button>
    </n-space>

    <!-- Volume List Section -->
    <n-card>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">存储卷</h2>
        <n-space>
          <n-button>列表设置</n-button>
          <n-input
            v-model:value="searchForm.info"
            placeholder="搜索"
            clearable
            @keyup.enter="handleSearch"
          >
            <template #suffix>
              <n-icon name="search" />
            </template>
          </n-input>
        </n-space>
      </div>

      <!-- Volumes Table -->
      <n-data-table
        :columns="columns"
        :data="volumeData"
        :pagination="pagination"
        :row-key="rowKey"
        v-model:checked-row-keys="checkedRowKeys"
        :bordered="false"
        :loading="loading"
      />
    </n-card>

    <!-- Create Volume Component -->
    <create-volume
      v-model:show="showCreateVolumeDrawer"
      @success="fetchVolumeList"
    />
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from "vue"
import { NButton, NSpace, NText, NInput, NIcon, NDataTable, NCard, useDialog, useMessage } from "naive-ui"
import type { DataTableColumns, DataTableRowKey, FormInst, FormRules } from "naive-ui"
import CreateVolume from "./create/index.vue"
import { containerVolumeListAPI, deleteVolume, containerPrune } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"
import type { SearchWithPage } from "@/api/interface"
import { computeSize } from "@/utils/util"

const message = useMessage()
const loading = ref(false)
const volumeData = ref<Container.VolumeInfo[]>([])
const searchForm = ref<SearchWithPage>({
	info: "",
	page: 1,
	limit: 10
})

const checkedRowKeys = ref<DataTableRowKey[]>([])

const rowKey = (row: Container.VolumeInfo) => row.name

const dialog = useDialog()

// --- Create Volume Drawer State & Logic ---
const showCreateVolumeDrawer = ref(false)
const createVolumeFormRef = ref<FormInst | null>(null)
const createVolumeFormValue = ref({
	name: "",
	driver: "local", // Default driver
	enableNFS: false,
	// nfsServerAddress: '', // if NFS is enabled
	// nfsPath: '', // if NFS is enabled
	driverOpts: "", // Store as newline-separated string for textarea
	labels: "" // Store as newline-separated string for textarea
})

const rules: FormRules = {
	name: [{ required: true, message: "请输入名称", trigger: "blur" }],
	driver: [{ required: true, message: "请选择模式", trigger: "change" }]
	// Add rules for NFS fields if they become required when enableNFS is true
}

const openCreateVolumeDrawer = () => {
	// Reset form to defaults
	createVolumeFormValue.value = {
		name: "",
		driver: "local",
		enableNFS: false,
		// nfsServerAddress: '',
		// nfsPath: '',
		driverOpts: "",
		labels: ""
	}
	createVolumeFormRef.value?.restoreValidation()
	showCreateVolumeDrawer.value = true
}

const handleCreateVolume = () => {
	createVolumeFormRef.value?.validate(errors => {
		if (!errors) {
			console.log("Create volume form is valid. Data:", createVolumeFormValue.value)
			// Prepare data for API call
			const apiPayload = {
				...createVolumeFormValue.value
				// Example: parse driverOpts and labels strings if API expects objects
				// driver_opts: parseKeyValuePairs(createVolumeFormValue.value.driverOpts),
				// labels: parseKeyValuePairs(createVolumeFormValue.value.labels),
			}
			console.log("API Payload would be:", apiPayload)
			// Call API to create volume
			showCreateVolumeDrawer.value = false
		} else {
			console.log("Create volume form validation errors:", errors)
		}
	})
}
// --- End Create Volume Drawer State & Logic ---

// 获取存储卷列表
const fetchVolumeList = async () => {
	try {
		loading.value = true
		const res = await containerVolumeListAPI(searchForm.value)
		volumeData.value = res.data.items || []
		pagination.value.itemCount = res.data.total || 0
	} catch (error) {
		volumeData.value = []
		pagination.value.itemCount = 0
	} finally {
		loading.value = false
	}
}

// 搜索
const handleSearch = () => {
	searchForm.value.page = 1
	fetchVolumeList()
}

// 删除单个存储卷
const handleDeleteRow = async (row: Container.VolumeInfo) => {
	try {
		await deleteVolume({ names: [row.name] })
		message.success("删除成功")
		fetchVolumeList()
	} catch (error) {
		message.error("删除失败")
		console.error("删除失败:", error)
	}
}

// 批量删除存储卷
const handleBulkDelete = async () => {
	if (checkedRowKeys.value.length === 0) return

	try {
		const names = checkedRowKeys.value
			.map(key => {
				const volume = volumeData.value.find(v => v.name === key)
				return volume?.name || ""
			})
			.filter(Boolean)

		await deleteVolume({ names })
		message.success("删除成功")
		checkedRowKeys.value = []
		fetchVolumeList()
	} catch (error) {
		message.error("删除失败")
		console.error("删除失败:", error)
	}
}

// 清理存储卷
const handleCleanVolumes = () => {
	dialog.create({
		title: "清理存储卷",
		content: () =>
			h("div", { class: "flex items-start" }, [
				h(NIcon, { size: 24, class: "mr-2 flex-shrink-0", style: { color: "#2080f0" } } as any),
				"清理存储卷将删除所有未被使用的本地存储卷，该操作无法回滚，是否继续？"
			]),
		positiveText: "确认",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const res = await containerPrune({
					pruneType: "volume",
					withTagAll: false
				})
				message.success(
					`清理成功，共删除 ${res.data.deletedNumber} 个存储卷，释放空间 ${computeSize(res.data.spaceReclaimed)}`
				)
				fetchVolumeList()
			} catch (error) {
				message.error("清理失败")
				console.error("清理失败:", error)
			}
		}
	})
}

// 表格列定义
const columns: DataTableColumns<Container.VolumeInfo> = [
	{
		type: "selection",
		disabled: (row: Container.VolumeInfo) => false
	},
	{
		title: "名称",
		key: "name",
		render(row: Container.VolumeInfo) {
			return h(
				NText,
				{ type: "primary", class: "cursor-pointer hover:underline" },
				{ default: () => row.name || "-" }
			)
		}
	},
	{
		title: "驱动",
		key: "driver",
		render(row: Container.VolumeInfo) {
			return row.driver || "-"
		}
	},
	{
		title: "挂载点",
		key: "mountpoint",
		render(row: Container.VolumeInfo) {
			return row.mountpoint || "-"
		}
	},
	{
		title: "创建时间",
		key: "createdAt",
		render(row: Container.VolumeInfo) {
			return row.createdAt || "-"
		}
	},
	{
		title: "操作",
		key: "actions",
		render(row: Container.VolumeInfo) {
			return h(
				NButton,
				{ text: true, type: "error", onClick: () => handleDeleteRow(row) },
				{ default: () => "删除" }
			)
		}
	}
]

// 分页配置
const pagination = ref({
	page: 1,
	limit: 10,
	showSizePicker: true,
	pageSizes: [10, 20, 50],
	itemCount: 0,
	onChange: (page: number) => {
		searchForm.value.page = page
		fetchVolumeList()
	},
	onUpdatePageSize: (limit: number) => {
		searchForm.value.limit = limit
		searchForm.value.page = 1
		fetchVolumeList()
	}
})

// 初始化
onMounted(() => {
	fetchVolumeList()
})
</script>

<style scoped>
/* You can add component-specific styles here if Tailwind isn't enough */
.py-4 {
	padding-top: 1rem;
	padding-bottom: 1rem;
}
</style>
