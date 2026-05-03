<template>
  <div class="py-4">
    <n-card>
      <ImageListToolbar
        :search-value="searchForm.info"
        @update:search-value="searchForm.info = $event"
        @search="handleSearch"
        @pull="openPullDrawer"
        @load="openLoadDrawer"
        @build="openBuildDrawer"
        @prune="showClearUnusedImagesConfirmation"
      />
      <n-data-table
        :columns="columns"
        :data="imageData"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
      />
    </n-card>

    <pull-image
      ref="pullImageRef"
      @search="fetchImageData"
    />

    <load-image
      ref="loadImageRef"
      @search="fetchImageData"
    />

    <build-image
      ref="buildImageRef"
      @search="fetchImageData"
    />

    <push-image
      ref="pushImageRef"
      @search="fetchImageData"
    />

    <save-image
      ref="saveImageRef"
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
import { h, ref, onMounted } from "vue"
import {
	NButton,
	NSpace,
	NTag,
	NText,
	useDialog,
	useMessage
} from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import { containerImageListAPI, listImageRepo, imageRemove } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"
import dayjs from "@/utils/dayjs"
import PullImage from "./pull/index.vue"
import LoadImage from "./load/index.vue"
import BuildImage from "./build/index.vue"
import PushImage from "./push/index.vue"
import SaveImage from "./save/index.vue"
import PruneImage from "./prune/index.vue"
import ImageListToolbar from "./ImageListToolbar.vue"

const dialog = useDialog()
const message = useMessage()
const pullImageRef = ref()
const loadImageRef = ref()
const buildImageRef = ref()
const pushImageRef = ref()
const saveImageRef = ref()
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

const openPullDrawer = () => {
	pullImageRef.value?.acceptParams({ repos: repos.value })
}

const openLoadDrawer = () => {
	loadImageRef.value?.acceptParams()
}

const openBuildDrawer = () => {
	buildImageRef.value?.acceptParams()
}

const getPushableTags = (row: ImageRow) => row.tags.filter(tag => tag && !tag.includes(":<none>"))

const openPushDrawer = (row: ImageRow) => {
	const tags = getPushableTags(row)
	if (tags.length === 0) {
		message.warning("当前镜像没有可推送的标签")
		return
	}
	if (repos.value.length === 0) {
		message.warning("请先在镜像仓库中添加并同步可用仓库")
		return
	}
	pushImageRef.value?.acceptParams({ repos: repos.value, tags })
}

const openSaveDrawer = (row: ImageRow) => {
	const tags = getPushableTags(row)
	if (tags.length === 0) {
		message.warning("当前镜像没有可导出的标签")
		return
	}
	saveImageRef.value?.acceptParams({ repos: repos.value, tags })
}

const showClearUnusedImagesConfirmation = () => {
	pruneImageRef.value?.acceptParams()
}

const fetchImageData = async () => {
	loading.value = true
	try {
		const params = {
			page: pagination.value.page,
			limit: pagination.value.limit,
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
	if (row.isUsed) {
		message.warning("已使用的镜像不允许删除")
		return
	}
	const targetName = specificTag || row.id
	const isTagDelete = !!specificTag
	const contentMsg = isTagDelete
		? `您确定要删除镜像标签 "${targetName}" 吗？此操作不可撤销。`
		: `您确定要删除镜像及其所有标签 "${row.tags.join(", ")}" 吗？此操作不可撤销。`

	const dialogReactive = dialog.warning({
		title: "确认删除",
		content: contentMsg,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			dialogReactive.loading = true
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
			} finally {
				dialogReactive.loading = false
			}
		}
	})
}

const createColumns = (): DataTableColumns<ImageRow> => [
	{
		title: "ID",
		key: "id",
		render(row: ImageRow) {
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
		render(row: ImageRow) {
			const statusType = row.isUsed ? "success" : "warning"
			const statusText = row.isUsed ? "已使用" : "未使用"
			return h(NTag, { type: statusType, size: "small", bordered: false }, { default: () => statusText })
		}
	},
	{
		title: "标签",
		key: "tags",
		render(row: ImageRow) {
			const tags = getPushableTags(row)
			if (tags.length === 0) {
				return h(NText, null, { default: () => "-" })
			}
			return h(
				NSpace,
				{ vertical: true, size: "small" },
				{
					default: () =>
						tags.map(tag =>
							h(
								NSpace,
								{ align: "center", size: "small", style: "display: inline-flex; margin-right: 8px;" },
								{
									default: () => [
										h(NTag, { bordered: false, size: "small" }, { default: () => tag }),
										...(!row.isUsed
											? [
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
											: [])
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
		key: "size",
		render(row: ImageRow) {
			return h(NText, null, { default: () => row.size || "-" })
		}
	},
	{
		title: "时间",
		key: "createdAt",
		render(row: ImageRow) {
			return h(NText, null, { default: () => dayjs(row.createdAt).format("YYYY-MM-DD HH:mm") })
		}
	},
	{
		title: "操作",
		key: "actions",
		render(row: ImageRow) {
			const hasPushableTags = getPushableTags(row).length > 0
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{ text: true, type: "primary", disabled: !hasPushableTags || repos.value.length === 0, onClick: () => openPushDrawer(row) },
						{ default: () => "推送" }
					),
					h(
						NButton,
						{ text: true, disabled: !hasPushableTags, onClick: () => openSaveDrawer(row) },
						{ default: () => "导出" }
					),
					...(!row.isUsed
						? [
								h(
									NButton,
									{ text: true, type: "error", onClick: () => handleDeleteImage(row) },
									{ default: () => "删除全部" }
								)
							]
						: [])
				]
			})
		}
	}
]

const columns = createColumns()

const pagination = ref({
	page: 1,
	limit: 10,
	showSizePicker: true,
	pageSizes: [10, 20, 50],
	itemCount: 0,
	onChange: (page: number) => {
		pagination.value.page = page
		fetchImageData()
	},
	onUpdatePageSize: (limit: number) => {
		pagination.value.limit = limit
		pagination.value.page = 1
		fetchImageData()
	}
})

const searchForm = ref({
	info: ""
})

const handleSearch = () => {
	pagination.value.page = 1
	fetchImageData()
}
</script>

<style scoped>
/* You can add component-specific styles here if needed, though Tailwind is preferred for general styling. */
</style>
