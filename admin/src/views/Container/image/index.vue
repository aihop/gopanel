<template>
  <div class="py-4">
    <!-- Header with Action Buttons -->
    <n-space class="mb-4">
      <n-button
        type="primary"
        @click="openImageActionDrawer('pull')"
      >{{ $t('container.imagePull') }}</n-button>
      <n-button @click="openImageActionDrawer('import')">{{ $t('container.imageImport') }}</n-button>
      <!-- <n-button @click="openImageActionDrawer('build')">构建镜像</n-button>
			<n-button @click="showClearBuildCacheConfirmation">清理构建缓存</n-button> -->
      <n-button
        type="error"
        @click="showClearUnusedImagesConfirmation"
      >{{ $t('container.imageDelete') }}</n-button>
    </n-space>

    <!-- Image List Section -->
    <n-card>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">{{ $t('container.image') }}</h2>
        <n-space>
          <n-popover
            trigger="click"
            placement="bottom-start"
            :width="300"
          >
            <template #trigger>
              <div class="rounded-full border border-gray-200 bg-base-100 p-2 px-5 text-sm">列表设置</div>
            </template>
            <div class="flex items-center gap-4 text-nowrap bg-base-100">
              刷新频率
              <n-select :options="[
									{ label: '不刷新', value: 0 },
									{ label: '5秒/次', value: 5 },
									{ label: '10秒/次', value: 10 },
									{ label: '30秒/次', value: 30 },
									{ label: '60秒/次', value: 60 },
									{ label: '120秒/次', value: 120 },
									{ label: '300秒/次', value: 300 }
								]" />
            </div>
          </n-popover>
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

      <!-- Images Table -->
      <n-data-table
        :columns="columns"
        :data="imageData"
        :pagination="pagination"
        :bordered="false"
      />
    </n-card>

    <!-- Unified Image Action Drawer -->
    <n-drawer
      v-model:show="showImageActionDrawer"
      :width="500"
      placement="right"
    >
      <n-drawer-content
        :title="drawerTitle"
        closable
      >
        <template #header>
          <div class="flex items-center">
            <div
              class="flex cursor-pointer items-center gap-2 text-gray-500"
              @click="showImageActionDrawer = false"
            >
              <Icon name="mdi:arrow-left" />
              返回
            </div>
            <n-divider vertical />
            {{ drawerTitle }}
          </div>
        </template>

        <n-form
          ref="imageActionFormRef"
          :model="imageActionFormValue"
          class="p-4"
        >
          <!-- Fields for Pull Image -->
          <template v-if="drawerMode === 'pull'">
            <n-form-item
              label="来源"
              path="fromRegistry"
            >
              <n-checkbox v-model:checked="imageActionFormValue.fromRegistry">镜像仓库</n-checkbox>
            </n-form-item>
            <n-form-item
              label="仓库名"
              path="registryName"
              :rule="{ required: true, message: '请选择仓库名', trigger: ['blur', 'change'] }"
            >
              <n-select
                v-model:value="imageActionFormValue.registryName"
                placeholder="请选择仓库名"
                :options="registryOptions"
                filterable
              />
            </n-form-item>
            <n-form-item
              label="镜像名"
              path="imageName"
              :rule="{ required: true, message: '请输入镜像名', trigger: 'blur' }"
            >
              <n-input
                v-model:value="imageActionFormValue.imageName"
                placeholder="例如：nginx:latest 或 library/nginx:latest"
              />
            </n-form-item>
          </template>

          <!-- Fields for Import Image -->
          <template v-if="drawerMode === 'import'">
            <n-form-item
              label="路径"
              path="path"
              :rule="{ required: true, message: '请输入镜像文件路径', trigger: 'blur' }"
            >
              <n-input
                v-model:value="imageActionFormValue.path"
                placeholder="请输入服务器上的镜像文件绝对路径 (.tar, .tar.gz, etc.)"
              >
                <template #prefix>
                  <n-icon name="folder_open" />
                </template>
              </n-input>
            </n-form-item>
          </template>

          <!-- Fields for Build Image -->
          <template v-if="drawerMode === 'build'">
            <n-form-item
              label="名称"
              path="imageNameAndTag"
              :rule="{ required: true, message: '请输入镜像名称及Tag', trigger: 'blur' }"
            >
              <n-input
                v-model:value="imageActionFormValue.imageNameAndTag"
                placeholder="镜像名称及 Tag, 例: nginx:latest"
              />
            </n-form-item>

            <n-form-item
              label="Dockerfile"
              path="dockerfileSourceType"
              :rule="{ required: true, message: '请选择Dockerfile来源或内容' }"
            >
              <n-radio-group
                v-model:value="imageActionFormValue.dockerfileSourceType"
                name="dockerfileSource"
              >
                <n-radio value="edit">编辑</n-radio>
                <n-radio
                  class="ml-4"
                  value="path"
                >路径选择</n-radio>
              </n-radio-group>
            </n-form-item>

            <n-form-item
              :show-label="false"
              v-if="imageActionFormValue.dockerfileSourceType === 'path'"
              path="dockerfilePath"
            >
              <n-input
                v-model:value="imageActionFormValue.dockerfilePath"
                placeholder="请输入Dockerfile在服务器上的路径"
              >
                <template #prefix>
                  <!-- Ensure Icon component is correctly set up -->
                  <Icon name="mdi:folder-outline" />
                </template>
              </n-input>
            </n-form-item>

            <n-form-item
              v-if="imageActionFormValue.dockerfileSourceType === 'edit'"
              label=" "
              path="dockerfileContent"
            >
              <FtEditor v-model:value="imageActionFormValue.dockerfileContent" />
            </n-form-item>

            <n-form-item
              label="标签"
              path="imageBuildLabels"
            >
              <n-input
                v-model:value="imageActionFormValue.imageBuildLabels"
                type="textarea"
                placeholder="一行一个, 例:\nkey1=value1\nkey2=value2"
                :autosize="{ minRows: 3, maxRows: 5 }"
              />
            </n-form-item>
          </template>
        </n-form>

        <template #footer>
          <n-space>
            <n-button @click="showImageActionDrawer = false">取消</n-button>
            <n-button
              type="primary"
              @click="handleImageAction"
            >{{ primaryButtonText }}</n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- Pull Image Component -->
    <pull-image
      ref="pullImageRef"
      @search="fetchImageData"
    />

    <!-- Prune Image Component -->
    <prune-image
      ref="pruneImageRef"
      @search="fetchImageData"
    />
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted, computed } from "vue"
import {
	NButton,
	NSpace,
	NTag,
	NText,
	NSpin,
	NDrawer,
	NDrawerContent,
	NForm,
	NFormItem,
	NCheckbox,
	NSelect,
	NInput,
	NIcon,
	NPopover,
	NDivider,
	NDynamicTags,
	NRadio,
	NRadioGroup,
	NRadioButton,
	useDialog,
	useMessage
} from "naive-ui"
import type { DataTableColumns, FormInst } from "naive-ui"
import { containerImageListAPI, imagePull, listImageRepo, imageRemove, containerPrune } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"
import FtEditor from "@/components/FtEditor/index.vue"
import dayjs from "@/utils/dayjs"
import PullImage from "./pull/index.vue"
import PruneImage from "./prune/index.vue"
import { computeSize } from "@/utils/util"

const dialog = useDialog()
const message = useMessage()
const pullImageRef = ref()
const pruneImageRef = ref()

type ImageRow = {
	id: string
	isUsed: boolean
	tags: string[]
	size: string
	createdAt: string
}

const imageData = ref<ImageRow[]>([])
const loading = ref(true)
const repos = ref<Container.RepoOptions[]>([])

// --- Unified Drawer State ---
const showImageActionDrawer = ref(false)
const drawerMode = ref<"pull" | "import" | "build" | null>(null)
const imageActionFormRef = ref<FormInst | null>(null)
const imageActionFormValue = ref({
	// Pull specific
	fromRegistry: false,
	registryName: "",
	imageName: "",
	// Import specific
	path: "",
	// Build specific (revised based on new image)
	imageNameAndTag: "", // Combined name and tag
	dockerfileSourceType: "path", // 'path' or 'edit'
	dockerfilePath: "", // Path if sourceType is 'path'
	dockerfileContent: "", // Content if sourceType is 'edit'
	imageBuildLabels: "" // For --label, multi-line string
})

const drawerTitle = computed(() => {
	if (drawerMode.value === "pull") return "拉取镜像"
	if (drawerMode.value === "import") return "导入镜像"
	if (drawerMode.value === "build") return "构建镜像"
	return ""
})

const primaryButtonText = computed(() => {
	if (drawerMode.value === "pull") return "拉取"
	if (drawerMode.value === "import") return "导入"
	if (drawerMode.value === "build") return "构建"
	return "确定"
})

const registryOptions = ref([
	{ label: "Docker Hub", value: "Docker Hub" }
	// Add other registry options here
])

const openImageActionDrawer = (mode: "pull" | "import" | "build") => {
	if (mode === "pull") {
		pullImageRef.value?.acceptParams({ repos: repos.value })
		return
	}

	drawerMode.value = mode
	// Reset form to defaults or mode-specific state
	imageActionFormValue.value = {
		fromRegistry: false,
		registryName: "",
		imageName: "",
		path: "",

		// Reset for build mode according to new image
		imageNameAndTag: "",
		dockerfileSourceType: "path",
		dockerfilePath: mode === "build" ? "Dockerfile" : "",
		dockerfileContent: "",
		imageBuildLabels: ""
	}
	imageActionFormRef.value?.restoreValidation()
	showImageActionDrawer.value = true
}

const handleImageAction = () => {
	imageActionFormRef.value?.validate(async errors => {
		if (!errors) {
			try {
				if (drawerMode.value === "pull") {
					const params = {
						fromRepo: imageActionFormValue.value.fromRegistry,
						repoID: imageActionFormValue.value.registryName === "Docker Hub" ? 1 : 0, // 默认使用 Docker Hub
						imageName: imageActionFormValue.value.imageName
					}

					const res = await imagePull(params)
					message.success("镜像拉取任务已开始")
					showImageActionDrawer.value = false
					fetchImageData() // 刷新镜像列表
				} else if (drawerMode.value === "import") {
					console.log("Importing image from:", imageActionFormValue.value.path)
					// Implement image import logic here
				} else if (drawerMode.value === "build") {
					console.log("Building image with config:", {
						nameAndTag: imageActionFormValue.value.imageNameAndTag,
						dockerfileSource: imageActionFormValue.value.dockerfileSourceType,
						dockerfilePath:
							imageActionFormValue.value.dockerfileSourceType === "path"
								? imageActionFormValue.value.dockerfilePath
								: undefined,
						dockerfileContent:
							imageActionFormValue.value.dockerfileSourceType === "edit"
								? imageActionFormValue.value.dockerfileContent
								: undefined,
						labels: imageActionFormValue.value.imageBuildLabels
					})
					// Implement image build logic here
				}
			} catch (error) {
				console.error("操作失败:", error)
				message.error("操作失败")
			}
		} else {
			console.log(`Validation errors for ${drawerMode.value} mode:`, errors)
		}
	})
}

const showClearBuildCacheConfirmation = () => {
	dialog.warning({
		title: "清理构建缓存",
		content: "清理构建缓存将删除所有构建产生的缓存，该操作无法回滚，是否继续？",
		positiveText: "确认",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const res = await containerPrune({
					pruneType: "buildcache",
					withTagAll: false
				})
				if (res.code === 0) {
					message.success(
						`清理成功，共删除 ${res.data.deletedNumber} 个构建缓存，释放空间 ${computeSize(res.data.spaceReclaimed)}`
					)
				} else {
					message.error(res.msg || "清理构建缓存失败")
				}
			} catch (error: any) {
				console.error("清理构建缓存失败:", error)
				message.error(error.msg || "清理构建缓存时发生错误")
			}
		}
	})
}

const showClearUnusedImagesConfirmation = () => {
	pruneImageRef.value?.acceptParams()
}

const fetchImageData = async () => {
	loading.value = true
	try {
		const params = {
			page: pagination.value.page,
			pageSize: pagination.value.pageSize,
			info: searchForm.value.info || ""
		}
		const response = await containerImageListAPI(params)
		if (response && response.data && response.data.items) {
			imageData.value = response.data.items.map((item: any) => ({
				id: item.id,
				isUsed: item.isUsed,
				tags: item.tags,
				size: item.size,
				createdAt: item.createdAt
			}))
			pagination.value.itemCount = response.data.total
		} else {
			imageData.value = []
			pagination.value.itemCount = 0
		}
	} catch (error) {
		console.error("获取镜像列表失败:", error)
		message.error("获取镜像列表失败")
		imageData.value = []
		pagination.value.itemCount = 0
	} finally {
		loading.value = false
	}
}

const fetchRepos = async () => {
	try {
		const res = await listImageRepo()
		if (res.data) {
			repos.value = Array.isArray(res.data) ? res.data : [res.data]
		}
	} catch (error) {
		console.error("获取镜像源列表失败:", error)
		message.error("获取镜像源列表失败")
	}
}

onMounted(async () => {
	await Promise.all([fetchImageData(), fetchRepos()])
})

const handleDeleteImage = (row: ImageRow, specificTag?: string) => {
	const targetName = specificTag || row.id
	const isTagDelete = !!specificTag
	const contentMsg = isTagDelete
		? `您确定要删除镜像标签 "${targetName}" 吗？此操作不可撤销。`
		: `您确定要删除镜像及其所有标签 "${row.tags.join(", ")}" 吗？此操作不可撤销。`

	dialog.warning({
		title: "确认删除",
		content: contentMsg,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const payload: Container.BatchDelete = { names: [targetName] }
				const response = await imageRemove(payload)
				if (response.code === 0) {
					message.success("镜像删除成功")
					await fetchImageData() // 刷新列表
				} else {
					message.error(response.msg || "删除镜像失败")
				}
			} catch (error: any) {
				console.error("删除镜像失败:", error)
				message.error(error.msg || "删除镜像时发生错误")
			}
		}
	})
}

const createColumns = (): DataTableColumns<ImageRow> => [
	{
		title: "ID",
		key: "id",
		render(row) {
			return h(
				NText,
				{
					type: "primary",
					onClick: () => console.log(`Clicked ID: ${row.id}`),
					class: "cursor-pointer hover:underline"
				},
				{ default: () => row.id.substring(7, 19) }
			)
		}
	},
	{
		title: "状态",
		key: "isUsed",
		render(row) {
			const statusType = row.isUsed ? "success" : "warning"
			const statusText = row.isUsed ? "已使用" : "未使用"
			return h(NTag, { type: statusType, size: "small", bordered: false }, { default: () => statusText })
		}
	},
	{
		title: "标签",
		key: "tags",
		render(row) {
			return h(
				NSpace,
				{ vertical: true, size: "small" },
				{
					default: () =>
						row.tags.map(tag =>
							h(
								NSpace,
								{ align: "center", size: "small", style: "display: inline-flex; margin-right: 8px;" },
								{
									default: () => [
										h(NTag, { bordered: false, size: "small" }, { default: () => tag }),
										h(
											NButton,
											{
												text: true,
												type: "error",
												size: "tiny",
												onClick: () => handleDeleteImage(row, tag)
											},
											{ default: () => "删" }
										)
									]
								}
							)
						)
				}
			)
		}
	},
	{
		title: "大小",
		key: "size"
	},
	{
		title: "时间",
		key: "createdAt",
		render(row) {
			return h(NText, null, { default: () => dayjs(row.createdAt).format("YYYY-MM-DD HH:mm") })
		}
	},
	{
		title: "操作",
		key: "actions",
		render(row) {
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{ text: true, type: "error", onClick: () => handleDeleteImage(row) },
						{ default: () => "删除全部" }
					)
				]
			})
		}
	}
]

const columns = createColumns()

const pagination = ref({
	page: 1,
	pageSize: 10,
	showSizePicker: true,
	pageSizes: [10, 20, 50],
	itemCount: 0,
	onChange: (page: number) => {
		pagination.value.page = page
	},
	onUpdatePageSize: (pageSize: number) => {
		pagination.value.pageSize = pageSize
		pagination.value.page = 1
	}
})

// 添加搜索表单
const searchForm = ref({
	info: ""
})

// 添加搜索处理函数
const handleSearch = () => {
	pagination.value.page = 1
	fetchImageData()
}
</script>

<style scoped>
/* You can add component-specific styles here if needed, though Tailwind is preferred for general styling. */
</style>
