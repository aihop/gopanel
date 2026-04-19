<template>
  <div class="py-4">
    <!-- Header with Action Buttons and Search -->
    <div class="mb-4 flex items-center justify-between">
      <n-space>
        <n-button
          type="primary"
          @click="openCreateTemplateDrawer"
        >创建编排模板</n-button>
        <n-button
          type="default"
          @click="handleBulkDelete"
          :disabled="checkedRowKeys.length === 0"
        >
          删除
        </n-button>
      </n-space>
      <n-space>
        <n-button>列表设置</n-button>
        <n-input
          placeholder="搜索"
          clearable
        >
          <template #suffix>
            <n-icon name="search" />
          </template>
        </n-input>
      </n-space>
    </div>

    <!-- Template List Section -->
    <n-card>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">编排模板</h2>
      </div>

      <!-- Templates Table -->
      <n-data-table
        :columns="columns"
        :data="templateData"
        :pagination="pagination"
        :row-key="rowKey"
        v-model:checked-row-keys="checkedRowKeys"
        :bordered="false"
      />
    </n-card>

    <!-- Create Template Drawer -->
    <n-drawer
      v-model:show="showCreateTemplateDrawer"
      :width="600"
      placement="right"
    >
      <n-drawer-content closable>
        <template #header>
          <div class="flex items-center">
            <div
              class="flex cursor-pointer items-center gap-1 text-gray-500"
              @click="showCreateTemplateDrawer = false"
            >
              <n-icon
                name="mdi:arrow-left"
                size="18"
              />
              返回
            </div>
            <n-divider vertical />
            创建编排模板
          </div>
        </template>

        <n-form
          ref="createTemplateFormRef"
          :model="createTemplateFormValue"
          :rules="rules"
          class="flex h-full flex-col p-4"
        >
          <n-form-item
            label="名称"
            path="name"
          >
            <n-input
              v-model:value="createTemplateFormValue.name"
              placeholder="请输入模板名称"
            />
          </n-form-item>
          <n-form-item
            label="描述"
            path="description"
          >
            <n-input
              v-model:value="createTemplateFormValue.description"
              placeholder="请输入模板描述 (可选)"
            />
          </n-form-item>
          <n-form-item
            label="模板内容"
            path="content"
            class="flex flex-grow flex-col"
          >
            <FtEditor />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space>
            <n-button @click="showCreateTemplateDrawer = false">取消</n-button>
            <n-button
              type="primary"
              @click="handleCreateTemplate"
            >确认</n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { h, ref } from "vue"
import {
	NButton,
	NSpace,
	NText,
	NInput,
	NIcon,
	NDataTable,
	NCard,
	NDrawer,
	NDrawerContent,
	NForm,
	NFormItem,
	NDivider
} from "naive-ui"
import type { DataTableColumns, DataTableRowKey, FormInst, FormRules } from "naive-ui"
import { Icon } from "@iconify/vue" // Assuming Icon component is used
import FtEditor from "@/components/FtEditor/index.vue"

// Define the type for a template row
type TemplateRow = {
	key: string | number // Required for row-key
	name: string
	description: string
	creationTime: string
}

// Sample data is empty as per the image
const templateData = ref<TemplateRow[]>([])

const checkedRowKeys = ref<DataTableRowKey[]>([])

const rowKey = (row: TemplateRow) => row.key

// --- Create Template Drawer State & Logic ---
const showCreateTemplateDrawer = ref(false)
const createTemplateFormRef = ref<FormInst | null>(null)
const createTemplateFormValue = ref({
	name: "",
	description: "",
	content: "# Define or paste the content of your docker-compose file here\n"
})

const rules: FormRules = {
	name: [{ required: true, message: "请输入模板名称", trigger: "blur" }],
	content: [{ required: true, message: "请输入模板内容", trigger: "blur" }] // Content is also likely required
}

const openCreateTemplateDrawer = () => {
	createTemplateFormValue.value = {
		name: "",
		description: "",
		content: "# Define or paste the content of your docker-compose file here\n"
	}
	createTemplateFormRef.value?.restoreValidation()
	showCreateTemplateDrawer.value = true
}

const handleCreateTemplate = () => {
	createTemplateFormRef.value?.validate(errors => {
		if (!errors) {
			console.log("Create template form is valid. Data:", createTemplateFormValue.value)
			// Call API to create template
			showCreateTemplateDrawer.value = false
		} else {
			console.log("Create template form validation errors:", errors)
		}
	})
}
// --- End Create Template Drawer State & Logic ---

// Define columns for the data table
const createColumns = ({ deleteRow }: { deleteRow: (row: TemplateRow) => void }): DataTableColumns<TemplateRow> => [
	{
		type: "selection"
	},
	{
		title: "名称",
		key: "name",
		render(row) {
			// In a real app, name might be a link to a detail page
			return h(NText, { class: "cursor-pointer hover:underline" }, { default: () => row.name })
		}
	},
	{
		title: "描述",
		key: "description"
	},
	{
		title: "创建时间",
		key: "creationTime"
	},
	{
		title: "操作",
		key: "actions",
		render(row) {
			// Placeholder for actions. Common actions might be Edit, Delete, View.
			// Since data is empty, we don't see specific actions in the image.
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{ text: true, type: "primary", onClick: () => console.log("Edit action on:", row.name) },
						{ default: () => "编辑" }
					),
					h(NButton, { text: true, type: "error", onClick: () => deleteRow(row) }, { default: () => "删除" })
				]
			})
		}
	}
]

// Function to handle single row deletion
const handleDeleteRow = (row: TemplateRow) => {
	// In a real app, you'd call an API here
	console.log("Delete row:", row.name)
	templateData.value = templateData.value.filter(item => item.key !== row.key)
	const index = checkedRowKeys.value.indexOf(row.key)
	if (index > -1) {
		checkedRowKeys.value.splice(index, 1)
	}
	pagination.value.itemCount = templateData.value.length
}

const columns = createColumns({ deleteRow: handleDeleteRow })

// Function to handle bulk deletion
const handleBulkDelete = () => {
	// In a real app, you'd call an API here with checkedRowKeys.value
	console.log("Bulk delete rows:", checkedRowKeys.value)
	templateData.value = templateData.value.filter(row => !checkedRowKeys.value.includes(row.key))
	checkedRowKeys.value = [] // Clear selection
	pagination.value.itemCount = templateData.value.length
}

const pagination = ref({
	page: 1,
	limit: 10,
	showSizePicker: true,
	pageSizes: [10, 20, 50],
	itemCount: templateData.value.length, // Initially 0
	onChange: (page: number) => {
		pagination.value.page = page
	},
	onUpdatePageSize: (limit: number) => {
		pagination.value.limit = limit
		pagination.value.page = 1
	}
})

</script>

<style scoped>
/* You can add component-specific styles here if Tailwind isn't enough */
</style>
