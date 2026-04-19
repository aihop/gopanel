<template>
  <div class="py-4">
    <n-space class="mb-4 items-center justify-between">
      <n-button
        type="primary"
        @click="openAddRepoDrawer()"
      >添加仓库</n-button>
      <n-space>
        <n-input
          v-model:value="searchValue"
          placeholder="搜索"
          clearable
          @keyup.enter="handleSearch"
          style="width: 200px"
        >
          <template #suffix>
            <n-icon
              name="search"
              @click="handleSearch"
              style="cursor: pointer"
            />
          </template>
        </n-input>
        <n-button @click="handleSearch">查找</n-button>
      </n-space>
    </n-space>

    <!-- Repository List Section -->
    <n-card>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">仓库</h2>
      </div>
      <n-data-table
        :columns="columns"
        :data="repositoryData"
        :pagination="pagination"
        :bordered="false"
        :row-key="rowKey"
      />
    </n-card>

    <!-- Add Repository Drawer -->
    <n-drawer
      v-model:show="showAddRepoDrawer"
      :width="500"
      placement="right"
    >
      <n-drawer-content closable>
        <template #header>
          <div class="flex items-center">
            <div
              class="flex cursor-pointer items-center gap-1 text-gray-500"
              @click="showAddRepoDrawer = false"
            >
              <n-icon
                name="mdi:arrow-left"
                size="18"
              />
              {{ $t("commons.button.back") }}
            </div>
            <n-divider vertical />
            {{ drawerMode === "edit" ? "编辑仓库" : "添加仓库" }}
          </div>
        </template>

        <n-form
          ref="addRepoFormRef"
          :model="addRepoFormValue"
          :rules="rules"
          class="space-y-3 p-4"
        >
          <n-form-item
            :label="$t('commons.table.name')"
            path="name"
          >
            <n-input
              v-model:value="addRepoFormValue.name"
              placeholder="请输入仓库名称"
              :disabled="drawerMode === 'edit'"
            />
          </n-form-item>
          <n-form-item
            label="认证"
            path="authRequired"
          >
            <n-radio-group v-model:value="addRepoFormValue.authRequired">
              <n-radio :value="true">是</n-radio>
              <n-radio :value="false">否</n-radio>
            </n-radio-group>
          </n-form-item>
          <template v-if="addRepoFormValue.authRequired">
            <n-form-item
              label="用户名"
              path="username"
            >
              <n-input
                v-model:value="addRepoFormValue.username"
                placeholder="请输入用户名"
              />
            </n-form-item>
            <n-form-item
              :label="$t('commons.login.password')"
              path="password"
            >
              <n-input
                type="password"
                show-password-on="mousedown"
                v-model:value="addRepoFormValue.password"
                placeholder="请输入密码"
              />
            </n-form-item>
          </template>
          <n-form-item
            label="下载地址"
            path="downloadUrl"
          >
            <n-input
              v-model:value="addRepoFormValue.downloadUrl"
              placeholder="例如: index.docker.io"
            />
          </n-form-item>
          <n-form-item
            label="协议"
            path="protocol"
          >
            <n-radio-group v-model:value="addRepoFormValue.protocol">
              <n-radio value="http">http</n-radio>
              <n-radio value="https">https</n-radio>
            </n-radio-group>
            <div
              v-if="addRepoFormValue.protocol === 'http'"
              class="mt-1 text-xs text-gray-500"
            >
              http 仓库添加授信需要重启 docker 服务
            </div>
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space>
            <n-button @click="showAddRepoDrawer = false">{{ $t("commons.button.cancel") }}</n-button>
            <n-button
              type="primary"
              @click="handleSaveRepo"
              :loading="isSubmitting"
            >{{ $t("commons.button.confirm") }}</n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from "vue"
import {
	NButton,
	NSpace,
	NTag,
	NText,
	NInput,
	NIcon,
	NDataTable,
	NCard,
	NDrawer,
	NDrawerContent,
	NForm,
	NFormItem,
	NRadioGroup,
	NRadio,
	NDivider,
	useMessage,
	useDialog
} from "naive-ui"
import type { DataTableColumns, FormInst, FormRules } from "naive-ui"
import {
	listImageRepo,
	createImageRepo,
	updateImageRepo,
	deleteImageRepo,
	checkRepoStatus,
	searchImageRepo
} from "@/api/modules/container"

// Define the type for a repository row
type RepositoryRow = {
	key: string | number
	name: string
	downloadUrl: string
	protocol?: string
	status?: string
	statusType?: "success" | "error" | "warning" | "info"
	creationTime?: string
	id: number
}

const rowKey = (row: RepositoryRow) => row.key

const message = useMessage()
const dialog = useDialog() // Initialize dialog service
const drawerMode = ref<"add" | "edit">("add") // To manage add/edit mode
const editingRepoId = ref<number | null>(null) // To store ID of repo being edited

// Initialize with an empty array, will be populated by API call
const repositoryData = ref<RepositoryRow[]>([])

// 搜索输入绑定变量
const searchValue = ref("")

// --- Add Repository Drawer State & Logic ---
const showAddRepoDrawer = ref(false)
const addRepoFormRef = ref<FormInst | null>(null)
const addRepoFormValue = ref({
	name: "",
	authRequired: false,
	username: "",
	password: "",
	downloadUrl: "",
	protocol: "https" // Default to https
})
const isSubmitting = ref(false) // Added for loading state

const rules: FormRules = {
	name: [{ required: true, message: "请输入仓库名称", trigger: "blur" }],
	authRequired: [{ required: true, type: "boolean", message: "请选择是否需要认证", trigger: "change" }],
	username: [
		{
			required: true,
			validator: (rule, value) => {
				if (addRepoFormValue.value.authRequired && !value) {
					return new Error("请输入用户名")
				}
				return true
			},
			trigger: ["input", "blur"]
		}
	],
	password: [
		{
			required: true,
			validator: (rule, value) => {
				if (addRepoFormValue.value.authRequired && !value) {
					return new Error("请输入密码")
				}
				return true
			},
			trigger: ["input", "blur"]
		}
	],
	downloadUrl: [{ required: true, message: "请输入下载地址", trigger: "blur" }],
	protocol: [{ required: true, message: "请选择协议", trigger: "change" }]
}

const openAddRepoDrawer = (repoToEdit?: RepositoryRow) => {
	if (repoToEdit) {
		drawerMode.value = "edit"
		editingRepoId.value = repoToEdit.id
		addRepoFormValue.value = {
			name: repoToEdit.name,
			downloadUrl: repoToEdit.downloadUrl,
			protocol: repoToEdit.protocol || "https", // Default if not present
			authRequired: false, // Default to false as auth details are not in listImageRepo response
			username: "",
			password: "" // Always clear password for edit
		}
	} else {
		drawerMode.value = "add"
		editingRepoId.value = null
		addRepoFormValue.value = {
			name: "",
			authRequired: false,
			username: "",
			password: "",
			downloadUrl: "",
			protocol: "https"
		}
	}
	addRepoFormRef.value?.restoreValidation()
	showAddRepoDrawer.value = true
}

const handleSaveRepo = async () => {
	addRepoFormRef.value?.validate(async errors => {
		if (!errors) {
			isSubmitting.value = true
			const formValues = addRepoFormValue.value
			let response: any

			if (drawerMode.value === "edit") {
				const payload = {
					id: editingRepoId.value,
					name: formValues.name,
					downloadUrl: formValues.downloadUrl,
					protocol: formValues.protocol,
					auth: formValues.authRequired,
					...(formValues.authRequired && { username: formValues.username, password: formValues.password })
				}
				try {
					console.log("API Payload for updateImageRepo:", payload)
					response = await updateImageRepo(payload) // Assuming updateImageRepo takes the full payload with id
					if (response.code === 0) {
						message.success(response.msg)
						showAddRepoDrawer.value = false
						await fetchRepositoryData()
					} else {
						message.error(response.msg)
					}
				} catch (error: any) {
					console.error("Error updating image repository:", error)
					message.error(error.message)
				}
			} else {
				// Add mode
				const payload: any = {
					name: formValues.name,
					downloadUrl: formValues.downloadUrl,
					protocol: formValues.protocol,
					auth: formValues.authRequired
				}
				if (formValues.authRequired) {
					payload.username = formValues.username
					payload.password = formValues.password
				}
				try {
					console.log("API Payload for createImageRepo:", payload)
					response = await createImageRepo(payload)
					if (response.code === 0) {
						message.success("仓库添加成功")
						showAddRepoDrawer.value = false
						await fetchRepositoryData()
					} else {
						message.error(response.msg || "添加仓库失败")
					}
				} catch (error: any) {
					console.error("Error creating image repository:", error)
					message.error(error.message || "添加仓库时发生错误")
				}
			}
			isSubmitting.value = false
		} else {
			console.log("Add/Edit repository form validation errors:", errors)
			message.error("请检查表单输入是否正确")
		}
	})
}
 
// Define columns for the data table
const createColumns = (): DataTableColumns<RepositoryRow> => [
	{
		title: "名称",
		key: "name"
		// render(row) {
		//   return h(NText, { type: 'primary', class: 'cursor-pointer hover:underline' }, { default: () => row.name });
		// }
	},
	{
		title: "下载地址",
		key: "downloadUrl"
	},
	{
		title: "协议",
		key: "protocol"
	},
	{
		title: "状态",
		key: "status",
		render(row) {
			return h(
				NTag,
				{ type: row.statusType || "default", bordered: false, size: "small" },
				{ default: () => row.status }
			)
		}
	},
	{
		title: "创建时间",
		key: "creationTime"
	},
	{
		title: "操作",
		key: "actions",
		render(row) {
			const isSystemRepo = row.id === 1
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{
							text: true,
							type: "primary",
							onClick: () => handleCheckRepoStatus(row),
							disabled: isSystemRepo
						},
						{ default: () => "同步" }
					),
					h(
						NButton,
						{
							text: true,
							type: "primary",
							onClick: () => openAddRepoDrawer(row),
							disabled: isSystemRepo
						},
						{ default: () => "编辑" }
					),
					h(
						NButton,
						{
							text: true,
							type: "error",
							onClick: () => handleDeleteRow(row),
							disabled: isSystemRepo
						},
						{ default: () => "删除" }
					)
				]
			})
		}
	}
]

// 同步镜像仓库调用 checkRepoStatus
const handleCheckRepoStatus = async (row: RepositoryRow) => {
	const res = await checkRepoStatus(row.id)
	console.log(res)
	if (res.code == 0) {
		message.success(res.msg)
	} else {
		message.error(res.msg)
	}
}

// Function to handle single row deletion
const handleDeleteRow = (row: RepositoryRow) => {
	// 判断如果ID是1，不允许删除
	if (row.id === 1) {
		message.warning("系统默认仓库不允许删除")
		return
	}

	dialog.warning({
		title: "确认删除",
		content: `您确定要删除仓库 "${row.name}" 吗？此操作不可撤销。`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const payload = { ids: [row.id] }
				console.log("API Payload for deleteImageRepo:", payload)
				const response = await deleteImageRepo(payload)
				if (response.code === 0) {
					message.success(`仓库 "${row.name}" 删除成功`)
					await fetchRepositoryData() // Refresh the list
				} else {
					message.error(response.msg || "删除仓库失败")
				}
			} catch (error: any) {
				console.error("Error deleting image repository:", error)
				message.error(error.message || "删除仓库时发生错误")
			}
		},
		onNegativeClick: () => {
			// message.info('取消删除'); // Optional: feedback for cancellation
		}
	})
}

const columns = createColumns()

const pagination = ref({
	page: 1,
	limit: 10,
	showSizePicker: true,
	pageSizes: [10, 20, 30, 50],
	itemCount: 0,
	onChange: (page: number) => {
		pagination.value.page = page
		fetchRepositoryData()
	},
	onUpdatePageSize: (limit: number) => {
		pagination.value.limit = limit
		pagination.value.page = 1
		fetchRepositoryData()
	}
})

// 搜索触发
const handleSearch = () => {
	pagination.value.page = 1
	fetchRepositoryData()
}

// Fetch data from API
const fetchRepositoryData = async () => {
	try {
		const params: any = {
			page: pagination.value.page,
			limit: pagination.value.limit
		}
		if (searchValue.value && searchValue.value.trim() !== "") {
			params.info = searchValue.value.trim()
		}
		const response = await searchImageRepo(params)
		if (response.code === 0 && response.data) {
			const items = Array.isArray(response.data.items) ? response.data.items : []
			repositoryData.value = items.map(repo => {
				// 兼容status字段不存在的情况
				const statusRaw = (repo as any).status
				let status = ""
				let statusType: "success" | "error" | "info" = "info"
				if (statusRaw === "Success") {
					status = "成功"
					statusType = "success"
				} else if (statusRaw === "Error") {
					status = "失败"
					statusType = "error"
				} else if (typeof statusRaw === "string") {
					status = statusRaw
					statusType = "info"
				}
				return {
					key: repo.id,
					id: repo.id,
					name: repo.name,
					downloadUrl: repo.downloadUrl,
					protocol: repo.protocol || "https",
					status,
					statusType,
					creationTime: repo.createdAt ? new Date(repo.createdAt).toLocaleString() : ""
				}
			})
			pagination.value.itemCount = response.data.total || 0
		} else {
			repositoryData.value = []
			pagination.value.itemCount = 0
			message.error(response.msg)
		}
	} catch (error: any) {
		repositoryData.value = []
		pagination.value.itemCount = 0
		console.error("Error fetching repository data:", error)
		message.error(error.message )
	}
}
 
onMounted(() => {
	fetchRepositoryData()
})

</script>
