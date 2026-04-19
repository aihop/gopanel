<template>
  <n-card>
    <div class="flex justify-between">
      <n-space>
        <n-dropdown
          placement="bottom-start"
          trigger="click"
          :options="options"
          @select="handleCreate"
        >
          <n-button type="primary">
            {{ $t("commons.button.create") }}
          </n-button>
        </n-dropdown>
        <n-button
          type="default"
          @click="openUpload"
        >{{ $t("file.upload") }}</n-button>

        <n-button
          plain
          @click="openMove('copy')"
          :disabled="selects.length === 0"
        >
          {{ $t("file.copy") }}
        </n-button>
        <n-button
          plain
          @click="openMove('cut')"
          :disabled="selects.length === 0"
        >
          {{ $t("file.move") }}
        </n-button>
        <n-button
          plain
          @click="openCompress(pathToFiles(selects))"
          :disabled="selects.length === 0"
        >
          {{ $t("file.compress") }}
        </n-button>
        <n-button
          plain
          @click="openBatchRole(pathToFiles(selects))"
          :disabled="selects.length === 0"
        >
          {{ $t("file.editPermissions") }}
        </n-button>
        <n-button
          plain
          @click="batchDelFiles"
          :disabled="selects.length === 0"
        >
          {{ $t("commons.button.delete") }}
        </n-button>
      </n-space>
      <div v-if="moveOpen">
        <n-space>
          <n-tooltip
            trigger="hover"
            placement="bottom"
          >
            <template #trigger>
              <n-button
                plain
                @click="openPaste"
              >{{ $t("file.paste") }} ({{ fileMove.count }})</n-button>
            </template>
            {{ $t("file.paste") }}
          </n-tooltip>
          <n-tooltip
            trigger="hover"
            placement="bottom"
          >
            <template #trigger>
              <n-button
                plain
                class="close"
                @click="closeMove"
              >×</n-button>
            </template>
            {{ $t("file.cancel") }}
          </n-tooltip>
        </n-space>
      </div>
    </div>
  </n-card>

  <n-card class="mt-4">
    <!-- 面包屑导航 -->
    <div class="mb-4 flex items-center gap-2">
      <n-button
        size="small"
        :disabled="searchParams.path === '/'"
        @click="goToParentDirectory"
      >
        返回上级
      </n-button>
      <template
        v-for="(segment, idx) in pathSegments"
        :key="idx"
      >
        <n-button
          text
          type="primary"
          style="padding: 0 4px"
          @click="goToPath(idx)"
        >
          {{ segment || "/" }}
        </n-button>
        <n-text
          v-if="idx !== pathSegments.length - 1"
          style="padding: 0 2px"
        >></n-text>
      </template>
    </div>

    <!-- 搜索和过滤区域 -->
    <div class="mb-4 flex items-center gap-4">
      <n-input
        v-model:value="searchParams.path"
        placeholder="路径"
        style="width: 40%"
        @keyup.enter="loadData"
      >
        <template #suffix>
          <n-button
            quaternary
            circle
            size="tiny"
            @click="handleCopyPath"
          >
            <template #icon>
              <n-icon>
                <Icon :name="copiedPath ? 'mdi:check' : 'mdi:content-copy'" />
              </n-icon>
            </template>
          </n-button>
        </template>
      </n-input>
      <n-input
        v-model:value="searchParams.search"
        placeholder="搜索文件名"
        style="width: 25%"
        @keyup.enter="loadData"
      />
      <a-space>
        <n-switch v-model:value="searchParams.showHidden" />
        <n-text class="ml-1">显示隐藏文件</n-text>
      </a-space>
      <a-space>
        <n-switch v-model:value="searchParams.expand" />
        <n-text class="ml-1">展开目录</n-text>
      </a-space>
      <n-button
        type="primary"
        @click="handleSearch"
      >{{ $t("commons.button.search") }}</n-button>
    </div>

    <!-- 文件列表 -->
    <n-data-table
      :columns="columns"
      :data="fileList"
      :loading="loading"
      :row-key="getFileRowKey"
      :remote="true"
      :pagination="pagination"
      v-model:checked-row-keys="selects"
      min-width="1280px"
    />
  </n-card>

  <CodeEditor ref="codeEditorRef" />
  <DeleteFile
    ref="deleteRef"
    @close="loadData"
  />
  <Upload
    ref="uploadRef"
    @close="loadData"
  />
  <Decompress
    ref="deCompressRef"
    @close="loadData"
  />
  <Preview ref="previewRef" />
  <CreateFile
    ref="createRef"
    @close="loadData"
  />
  <Chown
    ref="chownRef"
    @close="loadData"
  ></Chown>
  <BatchRole
    ref="batchRoleRef"
    @close="loadData"
  />
  <FileRename
    ref="renameRef"
    @close="loadData"
  />
  <Compress
    ref="compressRef"
    @close="loadData"
  />
  <Move
    ref="moveRef"
    @close="closeMovePage"
  />
</template>

<script lang="ts" setup>
import type { File } from "@/api/interface/file"
import type { DataTableColumns } from "naive-ui"
import { GetFileContent, GetFilesList } from "@/api/modules/file"
import { userTokenAPI } from "@/api/modules/user"
import { Languages, Mimetypes } from "@/global/mimetype"
import { formatTime } from "@/utils/date"
import { MsgWarning } from "@/utils/message"
import { computeSize, copyText, dateFormat, downloadFile, getFileType, getIcon, getRandomStr } from "@/utils/util"
import { NButton, NDropdown, NIcon, NInput, NSpace, NSwitch, NText, useMessage } from "naive-ui"
import { computed, h, onMounted, reactive, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useAuthStore } from "@/store/auth"
import CodeEditor from "./components/file-management/CodeEditor.vue"
import Decompress from "./components/file-management/Decompress.vue"
import Compress from "./components/file-management/Compress.vue"
import DeleteFile from "./components/file-management/DeleteFile.vue"
import Preview from "./components/file-management/Preview.vue"
import Upload from "./components/file-management/Upload.vue"
import CreateFile from "./components/file-management/Create.vue"
import Chown from "./components/file-management/Chown.vue"
import BatchRole from "./components/file-management/BatchRole.vue"
import FileRename from "./components/file-management/Rename.vue"
import Move from "./components/file-management/Move.vue"

const STORAGE_KEY = "files.lastPath" // 本地存储 key，改名可避免冲突

const { t } = useI18n()

// 响应式数据
const loading = ref(false)
const fileList = ref<File.File[]>([])
const message = useMessage()
const totalItems = ref(0)
const copiedPath = ref(false)
let copiedPathTimer: ReturnType<typeof setTimeout> | null = null
const fileCompress = reactive({ files: [""], name: "", dst: "", operate: "compress" })
const fileDeCompress = reactive({ files: [] as string[], path: "", name: "", dst: "", mimeType: "" })
const filePreview = reactive({ path: "", name: "", extension: "", fileType: "" })
const codeReq = reactive({ path: "", expand: false, page: 1, pageSize: 100 })
const fileCreate = reactive({ path: "/", isDir: false, mode: 0o755 })
const fileRename = reactive({ path: "", oldName: "" })
const fileMove = reactive({ oldPaths: [""], allNames: [""], type: "", path: "", name: "", count: 0, isDir: false })

// 代码编辑器相关
const codeEditorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
const deleteRef = ref<InstanceType<typeof DeleteFile> | null>(null)
const uploadRef = ref<InstanceType<typeof Upload> | null>(null)
const deCompressRef = ref<InstanceType<typeof Decompress> | null>(null)
const compressRef = ref<InstanceType<typeof Compress> | null>(null)
const createRef = ref<InstanceType<typeof CreateFile> | null>(null)
const chownRef = ref<InstanceType<typeof Chown> | null>(null)
const batchRoleRef = ref<InstanceType<typeof BatchRole> | null>(null)
const renameRef = ref<InstanceType<typeof FileRename> | null>(null)
const moveRef = ref<InstanceType<typeof Move> | null>(null)

const selects = ref<string[]>([])
const moveOpen = ref(false)

const previewRef = ref()
// 搜索参数
const searchParams = ref<File.ReqFile>({
	path: "/",
	search: "",
	expand: true,
	showHidden: true,
	page: 1,
	pageSize: 50,
	containSub: false,
	sortBy: "name",
	sortOrder: "ascending"
})

watch(
	() => searchParams.value.path,
	newPath => {
		try {
			localStorage.setItem(STORAGE_KEY, newPath || "/")
		} catch (e) {
			// ignore
		}
	},
	{ immediate: false }
)

const fileUpload = reactive({ path: "" })

const options = [
	{
		label: "📁 " + t("file.dir"),
		key: "dir"
	},
	{
		label: "📄 " + t("file.file"),
		key: "file"
	}
]

// 分页配置
const pagination = reactive({
	page: 1,
	pageSize: 50,
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100],
	itemCount: 0,
	onChange: (page: number) => {
		console.log("页码变化:", page)
		pagination.page = page
		searchParams.value.page = page
		loadData()
	},
	onUpdatePageSize: (pageSize: number) => {
		pagination.pageSize = pageSize
		pagination.page = 1
		searchParams.value.pageSize = pageSize
		searchParams.value.page = 1
		loadData()
	}
})

const getFileRowKey = (row: File.File) => row.path

const handleCopyPath = async () => {
	if (!searchParams.value.path) {
		return
	}
	await copyText(searchParams.value.path)
	copiedPath.value = true
	if (copiedPathTimer) {
		clearTimeout(copiedPathTimer)
	}
	copiedPathTimer = setTimeout(() => {
		copiedPath.value = false
	}, 1500)
}

// 表格列配置
const columns: DataTableColumns<File.File> = [
	{
		type: "selection" as const
	},
	{
		title: "名称",
		key: "name",
		render(row: File.File) {
			return h(
				NSpace,
				{ align: "center" },
				{
					default: () => [
						h("span", { style: { marginRight: "8px", fontSize: "16px" } }, row.isDir ? "📁" : "📄"),
						h(
							"span",
							{
								style: {
									cursor: "pointer",
									color: "#005eeb"
								},
								onClick: row.isDir ? () => handleEnterDirectory(row) : () => openView(row)
							},
							row.name
						)
					]
				}
			)
		}
	},
	{
		title: "大小",
		key: "size",
		render(row: File.File) {
			if (row.isDir) {
				return h("span", "-")
			}
			return h("span", computeSize(row.size))
		}
	},
	{
		title: "权限",
		key: "mode",
		render(row: File.File) {
			return h("span", row.mode)
		}
	},
	{
		title: "所有者",
		key: "user",
		render(row: File.File) {
			return h("span", `${row.user}:${row.group}`)
		}
	},
	{
		title: "修改时间",
		key: "modTime",
		render(row: File.File) {
			return h("span", formatTime(row.modTime))
		}
	},
	{
		title: "操作",
		key: "actions",
		fixed: "right" as const,
		render(row: File.File) {
			return h(
				NSpace,
				{ size: "small" },
				{
					default: () => [
						h(
							NButton,
							{
								size: "small",
								type: "primary",
								text: true,
								onClick: () => handleOpen(row)
							},
							{ default: () => t("file.open") }
						),
						h(
							NButton,
							{
								size: "small",
								type: "primary",
								disabled: !!row.isDir || !row.path,
								text: true,
								onClick: () => {
									if (row.isDir || !row.path) return
									const auth = useAuthStore().auth
									if (auth) {
										downloadFile(row.path, auth)
									} else {
										message.error(t("file.downloadError"))
									}
								}
							},
							{ default: () => t("file.download") }
						),
						h(
							NDropdown,
							{
								options: [
									// { label: t("file.move"), key: "move" },
									// { label: t("file.copy"), key: "copy" },
									// { label: t("file.paste"), key: "paste" },
									{ label: t("file.copyDir"), key: "copyDir" },
									{ label: t("file.editPermissions"), key: "batchRole" },
									{ label: t("file.rename"), key: "rename" },
									{ label: t("file.compress"), key: "compress" },
									{
										label: t("file.deCompress"),
										key: "decompress",
										disabled: Mimetypes.get(row.mimeType) === undefined || row.isDir
									},
									{ label: t("commons.button.delete"), key: "delete" },
									{ label: "授权下载链接", key: "copyAuthDownload" }
								],
								onSelect: key => {
									switch (key) {
										case "move":
											console.log("移动文件:", row)
											// TODO: 实现移动功能
											break
										case "copy":
											console.log("复制文件:", row)
											// TODO: 实现复制功能
											break
										case "paste":
											console.log("粘贴文件:", row)
											// TODO: 实现粘贴功能
											break
										case "copyDir":
											// 复制路径
											if (row?.path) {
												copyText(row?.path)
											}
											break
										case "delete":
											delFile(row)
											break
										case "batchRole":
											openBatchRole([row])
											break
										case "compress":
											openCompress([row])
											break
										case "decompress":
											openDeCompress(row)
											break
										case "rename":
											openRename(row)
											break
										case "copyAuthDownload":
											// 复制授权下载链接
											if (row?.path) {
												// 接口获取下载链接
												userTokenAPI({ path: row.path, timestamp: Date.now() })
													.then(res => {
														if (res.code === 0) {
															const auth = res.data
															if (auth) {
																const href = window.location.href
																const protocol = href.split("//")[0]
																const host = href.split("//")[1].split("/")[0]
																const url = `${protocol}//${host}/api/file/download?token=${auth}&path=${encodeURIComponent(row.path)}`
																copyText(url)
															} else {
																message.error(t("file.downloadError"))
															}
														} else {
															message.error(res.msg || t("file.downloadError"))
														}
													})
													.catch(() => {
														message.error(t("file.downloadError"))
													})
											}
											break
									}
								}
							},
							{
								default: () =>
									h(
										NButton,
										{
											size: "small",
											type: "primary",
											text: true
										},
										{ default: () => t("tabs.more") }
									)
							}
						)
					]
				}
			)
		}
	}
]

// 加载数据
async function loadData() {
	loading.value = true
	try {
		const response = await GetFilesList(searchParams.value)
		if (response.code === 0 && response.data) {
			fileList.value = response.data.items || []
			totalItems.value = response.data.itemTotal || 0
			// 处理边界情况：如果当前页超出范围，重置到第一页
			const maxPage = Math.ceil(totalItems.value / searchParams.value.pageSize)
			if (searchParams.value.page > maxPage && maxPage > 0) {
				searchParams.value.page = 1
				message.warning("当前页码超出范围，已重置到第一页")
			}
			// 更新分页信息
			pagination.page = searchParams.value.page
			pagination.pageSize = searchParams.value.pageSize
			pagination.itemCount = totalItems.value
			if (!moveOpen.value) {
				selects.value = []
			}
		} else {
			fileList.value = []
			totalItems.value = 0
		}
	} catch (error) {
		message.error("加载文件列表失败，请检查网络连接")
		fileList.value = []
		totalItems.value = 0
	} finally {
		loading.value = false
	}
}

function handleEnterDirectory(row: File.File) {
	if (row.isDir) {
		searchParams.value.path = row.path
		searchParams.value.page = 1
		searchParams.value.search = "" // 进入新目录时清空搜索
		loadData()
	}
}

// 返回上级目录
function goToParentDirectory() {
	if (searchParams.value.path !== "/") {
		const parentPath = searchParams.value.path.substring(0, searchParams.value.path.lastIndexOf("/"))
		searchParams.value.path = parentPath || "/"
		searchParams.value.page = 1
		loadData()
	}
}

// 搜索处理
function handleSearch() {
	searchParams.value.page = 1
	loadData()
}

const pathSegments = computed(() => {
	// 处理根路径和多余斜杠
	const segments = searchParams.value.path.split("/").filter((s, i, arr) => i === 0 || s)
	// 保证根路径为第一个
	if (segments[0] !== "") segments.unshift("")
	return segments
})

function goToPath(idx: number) {
	let newPath = "/"
	if (idx >= 0) {
		newPath += pathSegments.value.slice(1, idx + 1).join("/")
	}
	searchParams.value.path = newPath
	searchParams.value.page = 1
	searchParams.value.search = "" // 进入新目录时清空搜索
	loadData()
}

function openView(row: File.File) {
	const fileType = getFileType(row.extension)
	const previewTypes = ["image", "video", "audio"]
	if (previewTypes.includes(fileType)) {
		return openPreview(row, fileType)
	}
	const actionMap: Record<string, (item: File.File) => void> = {
		compress: openDeCompress,
		text: item => openCodeEditor(item.path, item.extension)
	}
	const path = row.isSymlink ? row.linkPath : row.path
	const action = actionMap[fileType]
	return action ? action(row) : openCodeEditor(path, row.extension)
}

const openCodeEditor = (path: string, extension: string) => {
	codeReq.path = path
	codeReq.expand = true

	if (extension !== "") {
		const ext = extension.substring(1)
		Languages.find(language => language.value.indexOf(ext) > -1)
	}

	GetFileContent(codeReq)
		.then(res => {
			codeEditorRef.value?.openModal()
			codeEditorRef.value?.acceptParams({
				path: res.data.path,
				extension: res.data.extension
			})
		})
		.catch(() => {})
}

function openPreview(item: File.File, fileType: string) {
	if (item.mode.toString() === "-" && item.user === "-" && item.group === "-") {
		MsgWarning(t("file.fileCanNotRead"))
		return
	}
	filePreview.path = item.isSymlink ? item.linkPath : item.path
	filePreview.name = item.name
	filePreview.extension = item.extension
	filePreview.fileType = fileType

	previewRef.value?.acceptParams(filePreview)
}

// ---------------- 操作处理 -------------------

function openUpload() {
	fileUpload.path = searchParams.value.path
	if (uploadRef.value) {
		uploadRef.value.acceptParams(fileUpload)
	}
}

const openChown = (item: File.File) => {
	chownRef.value?.acceptParams(item)
}

const openBatchRole = (items: File.File[]) => {
	batchRoleRef.value?.acceptParams({ files: items })
}

async function handleOpen(row: File.File) {
	if (row.isDir) {
		handleEnterDirectory(row)
	} else {
		openView(row)
	}
}

async function delFile(row: File.File | null) {
	if (deleteRef.value === null || row === null) return
	deleteRef.value.acceptParams([row])
}

function openDeCompress(item: File.File) {
	if (Mimetypes.get(item.mimeType) === undefined) {
		MsgWarning(t("file.canNotDeCompress"))
		return
	}

	fileDeCompress.name = item.name
	fileDeCompress.path = item.path
	fileDeCompress.dst = searchParams.value.path
	fileDeCompress.mimeType = item.mimeType
	deCompressRef.value?.acceptParams(fileDeCompress)
}

const openCompress = (items: File.File[]) => {
	const paths = []
	for (const item of items) {
		paths.push(item.path)
	}
	fileCompress.files = paths
	if (paths.length === 1) {
		fileCompress.name = items[0].name
	} else {
		fileCompress.name = getRandomStr(6)
	}
	fileCompress.dst = searchParams.value.path

	compressRef.value?.acceptParams(fileCompress)
}

// 组件挂载时加载数据
onMounted(() => {
	const saved = localStorage.getItem(STORAGE_KEY)
	if (saved && saved !== "") {
		searchParams.value.path = saved
		searchParams.value.page = 1
	} else {
		// Default handle
		const authStore = useAuthStore()
		if (authStore.user && authStore.user.role === 'SUB_ADMIN' && authStore.user.fileBaseDir) {
			searchParams.value.path = authStore.user.fileBaseDir
		} else {
			searchParams.value.path = "/"
		}
	}
	loadData()
})

const handleCreate = (command: string) => {
	fileCreate.path = searchParams.value.path
	fileCreate.isDir = false
	if (command === "dir") {
		fileCreate.isDir = true
	}
	createRef.value?.acceptParams(fileCreate)
}

const openRename = (item: File.File) => {
	fileRename.path = searchParams.value.path
	fileRename.oldName = item.name
	renameRef.value?.acceptParams(fileRename)
}

const batchDelFiles = () => {
	deleteRef.value?.acceptParams(pathToFiles(selects.value))
}

const pathToFiles = (paths: string[]): File.File[] => {
	const files: File.File[] = []
	for (const path of paths) {
		const file = fileList.value.find(f => f.path === path)
		if (file) {
			files.push(file)
		}
	}
	return files
}

const openMove = (type: string) => {
	fileMove.type = type
	fileMove.name = ""
	fileMove.allNames = []
	fileMove.isDir = false
	const oldPaths = []
	for (const path of selects.value) {
		oldPaths.push(path)
	}
	fileMove.count = selects.value.length
	fileMove.oldPaths = oldPaths
	if (selects.value.length == 1) {
		const files = pathToFiles(selects.value)
		fileMove.name = files[0].name
		fileMove.isDir = files[0].isDir
	} else {
		const allNames = []
		const newFiles = pathToFiles(selects.value)
		for (const s of newFiles) {
			allNames.push(s["name"])
		}
		fileMove.allNames = allNames
	}
	moveOpen.value = true
}

const closeMove = () => {
	selects.value = []
	fileMove.oldPaths = []
	fileMove.name = ""
	fileMove.count = 0
	fileMove.isDir = false
	moveOpen.value = false
}

const openPaste = () => {
	fileMove.path = searchParams.value.path
	moveRef.value?.acceptParams(fileMove)
}
const closeMovePage = (submit: Boolean) => {
	if (submit) {
		loadData()
		closeMove()
	}
}
</script>
