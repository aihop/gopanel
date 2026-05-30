<template>
  <FileToolbar
    :selected-count="selects.length"
    :move-open="moveOpen"
    :paste-count="fileMove.count"
    :create-options="options"
    @create="handleCreate"
    @upload="openUpload"
    @move="openMove"
    @compress="openCompress(pathToFiles(selects))"
    @batch-role="openBatchRole(pathToFiles(selects))"
    @delete="batchDelFiles"
    @paste="openPaste"
    @cancel-move="closeMove"
  />

  <FileBrowserFilters
    :path="searchParams.path"
    :search="searchParams.search || ''"
    :show-hidden="!!searchParams.showHidden"
    :expand="searchParams.expand"
    :copied-path="copiedPath"
    :path-segments="pathSegments"
    @update:path="searchParams.path = $event"
    @update:search="searchParams.search = $event"
    @update:show-hidden="searchParams.showHidden = $event"
    @update:expand="searchParams.expand = $event"
    @load="loadData"
    @search="handleSearch"
    @copy-path="handleCopyPath"
    @go-parent="goToParentDirectory"
    @go-path="goToPath"
  >
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
  </FileBrowserFilters>

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
  <Wget
    ref="wgetRef"
    @close="loadData"
  />
  <Chown
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
    @close="handleCloseMovePage"
  />
</template>

<script lang="ts" setup>
import type { File } from "@/api/interface/file"
import { ComputeDirSize, GetFileContent, GetFilesList } from "@/api/modules/file"
import { Languages } from "@/global/mimetype"
import { MsgWarning, MsgSuccess, MsgError } from "@/utils/message"
import { computeSize, copyText, getFileType } from "@/utils/util"
import { useMessage } from "naive-ui"
import { onMounted, reactive, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useAuthStore } from "@/store/auth"
import CodeEditor from "./components/file-management/CodeEditor.vue"
import Decompress from "./components/file-management/Decompress.vue"
import Compress from "./components/file-management/Compress.vue"
import DeleteFile from "./components/file-management/DeleteFile.vue"
import Preview from "./components/file-management/Preview.vue"
import Upload from "./components/file-management/Upload.vue"
import CreateFile from "./components/file-management/Create.vue"
import Wget from "./components/file-management/Wget.vue"
import Chown from "./components/file-management/Chown.vue"
import BatchRole from "./components/file-management/BatchRole.vue"
import FileRename from "./components/file-management/Rename.vue"
import Move from "./components/file-management/Move.vue"
import FileToolbar from "./components/file-management/FileToolbar.vue"
import FileBrowserFilters from "./components/file-management/FileBrowserFilters.vue"
import { createFileTableColumns } from "./components/file-management/fileTableColumns"
import { useFilePathNavigation } from "./components/file-management/useFilePathNavigation"
import { useFileManagementActions } from "./components/file-management/useFileManagementActions"

const STORAGE_KEY = "files.lastPath" // 本地存储 key，改名可避免冲突

const { t } = useI18n()
const authStore = useAuthStore()

// 响应式数据
const loading = ref(false)
const fileList = ref<File.File[]>([])
const message = useMessage()
const totalItems = ref(0)
const copiedPath = ref(false)
let copiedPathTimer: ReturnType<typeof setTimeout> | null = null
const codeReq = reactive({ path: "", expand: false, page: 1, limit: 100 })

// 代码编辑器相关
const codeEditorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
const deleteRef = ref<InstanceType<typeof DeleteFile> | null>(null)
const uploadRef = ref<InstanceType<typeof Upload> | null>(null)
const deCompressRef = ref<InstanceType<typeof Decompress> | null>(null)
const compressRef = ref<InstanceType<typeof Compress> | null>(null)
const createRef = ref<InstanceType<typeof CreateFile> | null>(null)
const wgetRef = ref<InstanceType<typeof Wget> | null>(null)
const batchRoleRef = ref<InstanceType<typeof BatchRole> | null>(null)
const renameRef = ref<InstanceType<typeof FileRename> | null>(null)
const moveRef = ref<InstanceType<typeof Move> | null>(null)

const selects = ref<string[]>([])
const previewRef = ref()
// 搜索参数
const searchParams = ref<File.ReqFile>({
	path: "/",
	search: "",
	expand: true,
	showHidden: true,
	page: 1,
	limit: 50,
	containSub: false,
	sortBy: "name",
	sortOrder: "ascending"
})

const {
	fileMove,
	moveOpen,
	pathToFiles,
	openUpload,
	openBatchRole,
	delFile,
	openDeCompress,
	openCompress,
	handleCreate,
	openRename,
	batchDelFiles,
	openMove,
	closeMove,
	openPaste,
	closeMovePage,
	openPreview: openPreviewModal
} = useFileManagementActions({
	t,
	fileList,
	selects,
	searchParams,
	uploadRef,
	deleteRef,
	deCompressRef,
	compressRef,
	createRef,
	wgetRef,
	batchRoleRef,
	renameRef,
	moveRef,
	previewRef
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

const options = [
	{
		label: "📁 " + t("file.dir"),
		key: "dir"
	},
	{
		label: "📄 " + t("file.file"),
		key: "file"
	},
	{
		label: "⬇ " + t("file.downloadRemote"),
		key: "downloadRemote"
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
		pagination.page = page
		searchParams.value.page = page
		loadData()
	},
	onUpdatePageSize: (pageSize: number) => {
		pagination.pageSize = pageSize
		pagination.page = 1
		searchParams.value.limit = pageSize
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

// 加载数据
async function loadData() {
	loading.value = true
	try {
		const response = await GetFilesList(searchParams.value)
		if (response.code === 0 && response.data) {
			fileList.value = response.data.items || []
			totalItems.value = response.data.itemTotal || 0
			// 处理边界情况：如果当前页超出范围，重置到第一页
			const maxPage = Math.ceil(totalItems.value / searchParams.value.limit)
			if (searchParams.value.page > maxPage && maxPage > 0) {
				searchParams.value.page = 1
				message.warning("当前页码超出范围，已重置到第一页")
			}
			// 更新分页信息
			pagination.page = searchParams.value.page
			pagination.pageSize = searchParams.value.limit
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

const {
	handleSearch,
	pathSegments,
	goToParentDirectory,
	goToPath
} = useFilePathNavigation({
	searchParams,
	loadData
})

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

const openDirSize = async (row: File.File) => {
	if (!row.isDir) return
	try {
		const res = await ComputeDirSize({ path: row.path })
		const size = res?.data?.size
		if (size !== undefined && size !== null) {
			MsgSuccess(`${row.name} 大小：${computeSize(size)}`)
		} else {
			MsgError(t("commons.msg.operationFailed"))
		}
	} catch (e: any) {
		MsgError(e?.msg || t("commons.msg.operationFailed"))
	}
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
	openPreviewModal(item, fileType)
}

// 组件挂载时加载数据
onMounted(() => {
	const saved = localStorage.getItem(STORAGE_KEY)
	if (saved && saved !== "") {
		searchParams.value.path = saved
		searchParams.value.page = 1
	} else {
		// Default handle
		if (authStore.user && authStore.user.role === 'SUB_ADMIN' && authStore.user.fileBaseDir) {
			searchParams.value.path = authStore.user.fileBaseDir
		} else {
			searchParams.value.path = "/"
		}
	}
	loadData()
})


const columns = createFileTableColumns({
	t,
	getAuth: () => authStore.auth,
	onEnterDirectory: handleEnterDirectory,
	onOpenView: openView,
	onDelete: delFile,
	onBatchRole: openBatchRole,
	onCompress: openCompress,
	onDecompress: openDeCompress,
	onRename: openRename,
	onDirSize: openDirSize,
	onError: (text) => message.error(text)
})

const handleCloseMovePage = (submit: Boolean) => closeMovePage(submit, loadData)
</script>
