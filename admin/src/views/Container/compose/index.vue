<template>
  <div class="py-4">
    <ComposeListToolbar
      :search-name="searchName"
      @create="openCreateModal"
      @search="handleSearchEnter"
      @update:search-name="searchName = $event"
    />

    <n-card title="编排">
      <n-data-table
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="false"
        :bordered="false"
        class="mb-4"
      />
      <div class="flex items-center justify-end">
        <n-pagination
          v-model:page="paginationReactive.page"
          v-model:page-size="paginationReactive.limit"
          :item-count="paginationReactive.itemCount"
          :page-sizes="paginationReactive.pageSizes"
          show-size-picker
          show-quick-jumper
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        >
          <template #prefix="{ itemCount }">共 {{ itemCount }} 条</template>
        </n-pagination>
      </div>
    </n-card>
  </div>

  <ComposeCreateDrawer
    :show="showCreateModal"
    :base-dir="baseDir"
    :compose-form="composeForm"
    :active-tab="activeTab"
    :log-content="logContent"
    :env-placeholder="envPlaceholder"
    @update:show="showCreateModal = $event"
    @update:active-tab="activeTab = $event"
    @confirm="handleConfirmCreate"
  />

  <ComposeDeleteModal
    :show="showDeleteModal"
    :row="deleteRow"
    :delete-with-file="deleteWithFile"
    :delete-confirm-input="deleteConfirmInput"
    :delete-error="deleteError"
    @update:show="showDeleteModal = $event"
    @update:delete-with-file="deleteWithFile = $event"
    @update:delete-confirm-input="deleteConfirmInput = $event"
    @confirm="handleDeleteCompose"
  />

  <EditDrawer
    ref="editDrawerRef"
    @search="search"
  />
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, onUnmounted } from "vue"
import {
	NDataTable,
	NPagination,
	NCard,
} from "naive-ui"
import {
	containerComposeListAPI,
	composeOperator
} from "@/api/modules/container"
import { appsGetBaseDir } from "@/api/modules/apps"
import EditDrawer from "./edit/index.vue"
import ComposeCreateDrawer from "./ComposeCreateDrawer.vue"
import ComposeDeleteModal from "./ComposeDeleteModal.vue"
import ComposeListToolbar from "./ComposeListToolbar.vue"
import { createComposeColumns } from "./composeTableColumns"
import type { RowData } from "./composeTypes"
import { useComposeCreateFlow } from "./useComposeCreateFlow"

const loading = ref(false)
const paginationReactive = reactive({
	page: 1,
	limit: 10,
	itemCount: 0,
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100],
	onChange: (page: number) => {
		paginationReactive.page = page
		search()
	},
	onUpdatePageSize: (limit: number) => {
		paginationReactive.limit = limit
		paginationReactive.page = 1
		search()
	}
})
const searchName = ref()
const tableData = ref<RowData[]>([])

const search = async () => {
	const params: any = {
		page: paginationReactive.page,
		limit: paginationReactive.limit
	}
	if (searchName.value && String(searchName.value).trim() !== "") {
		params.info = String(searchName.value).trim()
	} else {
		params.info = ""
	}
	loading.value = true
	await containerComposeListAPI(params)
		.then(res => {
			loading.value = false
			if (res.data && res.data.items) {
				tableData.value = res.data.items.map((item: any, index: number) => ({
					key: index,
					name: item.name,
					source: item.createdBy || "自定义", // 使用createdBy字段作为来源
					directory: item.workdir,
					status:
						item.containers && Array.isArray(item.containers)
							? `${item.containers.filter((c: any) => c.state === "running").length}/${item.containerNumber}`
							: "0/0",
					createdTime: item.createdAt,
					containerNumber: item.containerNumber,
					configFile: item.configFile,
					workdir: item.workdir,
					path: item.path,
					containers: item.containers || [],
					env: item.env
				}))
				paginationReactive.itemCount = res.data.total
			}
		})
		.finally(() => {
			loading.value = false
		})
}

const columns = createComposeColumns({
	edit: handleEdit,
	remove: (row: RowData) => openDeleteModal(row)
})

const baseDir = ref("")
const {
	showCreateModal,
	composeForm,
	envPlaceholder,
	activeTab,
	logContent,
	openCreateModal,
	handleConfirmCreate,
	stopLogPolling
} = useComposeCreateFlow()

onMounted(async () => {
	try {
		const res = await appsGetBaseDir()
		if (res.code == 0) {
			baseDir.value = res.data || "/opt/gopanel/data/docker/compose/"
		}
	} catch {
		baseDir.value = "/opt/gopanel/data/docker/compose/"
	}
	search()
})

onUnmounted(() => {
	stopLogPolling()
})

const showDeleteModal = ref(false)
const deleteRow = ref<RowData | null>(null)
const deleteWithFile = ref(false)
const deleteConfirmInput = ref("")
const deleteError = ref("")

function openDeleteModal(row: RowData) {
	deleteRow.value = row
	deleteWithFile.value = false
	deleteConfirmInput.value = ""
	deleteError.value = ""
	showDeleteModal.value = true
}

async function handleDeleteCompose() {
	if (!deleteRow.value) return
	if (deleteConfirmInput.value !== deleteRow.value.name) {
		deleteError.value = "请输入正确的名称以确认删除"
		return
	}
	deleteError.value = ""
	try {
		await composeOperator({
			name: deleteRow.value.name,
			path: deleteRow.value.path,
			operation: "delete",
			withFile: deleteWithFile.value
		})
		showDeleteModal.value = false
		// 删除后刷新列表
		search()
	} catch (e) {
		deleteError.value = "删除失败，请重试"
	}
}

function handlePageChange(page: number) {
	paginationReactive.page = page
	search()
}
function handlePageSizeChange(limit: number) {
	paginationReactive.limit = limit
	paginationReactive.page = 1
	search()
}

// 搜索输入框回车事件
function handleSearchEnter() {
	paginationReactive.page = 1
	search()
}

const editDrawerRef = ref()

function handleEdit(row: RowData) {
	if (editDrawerRef.value && editDrawerRef.value.acceptParams) {
		editDrawerRef.value.acceptParams({
			name: row.name,
			path: row.path,
			content: row.configFile,
			env: row.env ? row.env.split("\n") : [],
			envStr: row.env || "",
			createdBy: row.source
		})
	}
}
</script>
