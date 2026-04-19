<template>
  <!-- 抽屉 -->
  <n-drawer
    v-model:show="open"
    :width="640"
    placement="right"
    :mask-closable="false"
    :close-on-esc="false"
    :on-after-leave="handleClose"
  >
    <n-drawer-content
      :title="$t('file.upload')"
      :closable="true"
    >
      <!-- 顶部按钮 -->
      <n-space
        justify="space-between"
        class="mb-4"
      >
        <n-space>
          <n-button
            type="primary"
            ghost
            @click="upload('file')"
          >
            {{ $t("file.uploadFile") }}
          </n-button>
          <n-button
            type="primary"
            @click="upload('dir')"
          >
            {{ $t("file.uploadDirectory") }}
          </n-button>
        </n-space>
        <n-button @click="clearFiles">{{ $t("file.clearList") }}</n-button>
      </n-space>
      <div
        class="upload-drop"
        @dragover.prevent="handleDragover"
        @drop.prevent="handleDrop"
        @dragleave.prevent="handleDragleave"
      >
        <n-empty
          description=""
          class="flex-center"
          style="height: 200px"
        >
          <template #icon>
            <n-icon>
              <Icon name="mdi:cloud-upload" />
            </n-icon>
          </template>
          <p>{{ $t("file.dropHelper") }}</p>
        </n-empty>
      </div>

      <!-- 上传入口（隐藏 input） -->
      <input
        ref="uploadInput"
        type="file"
        :multiple="true"
        :webkitdirectory="false"
        style="display: none"
        @change="fileOnChange"
      />

      <!-- 进度条 -->
      <n-progress
        v-if="loading"
        type="line"
        :percentage="uploadPercent"
        indicator-placement="inside"
        processing
      />

      <!-- 文件列表 -->
      <n-scrollbar
        style="max-height: 280px; background-color: #eeeeee80"
        class="mt-4 p-4"
      >
        <n-space
          vertical
          size="small"
        >
          <n-space
            v-for="(item, index) in uploaderFiles"
            :key="item.uid"
            align="center"
            justify="space-between"
            class="mb-2"
          >
            <n-space align="center">
              <n-icon size="16">
                <Icon
                  v-if="item.raw?.webkitRelativePath"
                  name="mdi:folder"
                />
                <Icon
                  v-else
                  name="mdi:file-document"
                />
              </n-icon>
              <span class="sle">
                {{ item.raw?.webkitRelativePath || item.name }}
              </span>
            </n-space>

            <!-- 成功/删除 -->
            <n-space>
              <n-icon
                v-if="item.status === 'success'"
                color="#18a058"
              >
                <Icon name="mdi:archive-check" />
              </n-icon>
              <n-button
                v-else
                text
                type="error"
                :disabled="loading"
                @click="removeFile(index)"
              >
                <n-icon>
                  <Icon name="mdi:remove" />
                </n-icon>
              </n-button>
            </n-space>
          </n-space>
        </n-space>
      </n-scrollbar>

      <!-- 底部按钮 -->
      <template #footer>
        <n-space justify="end">
          <n-button
            :disabled="loading"
            @click="handleClose"
          >
            {{ $t("commons.button.cancel") }}
          </n-button>
          <n-button
            type="primary"
            :disabled="loading || uploaderFiles.length === 0"
            :loading="loading"
            @click="submit"
          >
            {{ $t("commons.button.confirm") }}
          </n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>

  <!-- 已存在文件弹窗 -->
  <ExistFileDialog ref="dialogExistFileRef" />
</template>

<script setup lang="ts">
import { BatchCheckFiles, ChunkUploadFileData, UploadFileData } from "@/api/modules/file"
import { TimeoutEnum } from "@/enums/http-enum"
import { MsgSuccess, MsgWarning } from "@/utils/message"
import { NButton, NDrawer, NDrawerContent, NEmpty, NIcon, NProgress, NScrollbar, NSpace } from "naive-ui"
import { nextTick, reactive, ref } from "vue"
import { useI18n } from "vue-i18n"
import ExistFileDialog from "./ExistFile.vue"

/* -------------------- 事件 -------------------- */
const em = defineEmits(["close"])
const { t } = useI18n()
/* -------------------- 状态 -------------------- */
const open = ref(false)
const loading = ref(false)
const uploadPercent = ref(0)
const path = ref("")
const uploadHelper = ref("")
const dialogExistFileRef = ref()

const uploaderFiles = ref<any[]>([])
const tmpFiles = ref<any[]>([])
const breakFlag = ref(false)
const CHUNK_SIZE = 1024 * 1024 * 3 // 单个分片大小受服务器定义的限制
const MAX_SINGLE_FILE_SIZE = 1024 * 1024 * 4 // 超过这个大小的文件进行分片上传
// 如果需要设置更大，需要go部分的代码设置 app := fiber.New(fiber.Config{ BodyLimit: 10 * 1024 * 1024,}) ，否则默认4MB

const uploadInput = ref<HTMLInputElement>()
const state = reactive({ uploadEle: null as HTMLInputElement | null })

function handleClose() {
	open.value = false
	clearFiles()
	em("close")
}

interface UploadFileProps {
	path: string
}

function acceptParams(props: UploadFileProps) {
	path.value = props.path
	open.value = true
	uploadPercent.value = 0
	uploadHelper.value = ""

	nextTick(() => {
		state.uploadEle = uploadInput.value!
	})
}

/* -------------------- 上传相关 -------------------- */
function upload(type: "file" | "dir") {
	if (!state.uploadEle) return
	state.uploadEle.webkitdirectory = type === "dir"
	state.uploadEle.value = ""
	state.uploadEle.click()
}

function fileOnChange(e: Event) {
	const target = e.target as HTMLInputElement
	const files = Array.from(target.files || [])
	if (!files.length) return
	files.forEach(file => handleFile(file, ""))
	if (breakFlag.value) {
		MsgWarning(t("file.uploadOverLimit"))
	} else {
		uploaderFiles.value.push(...tmpFiles.value)
		initTempFiles()
	}
}

function handleFile(file: File, subPath: string) {
	if (tmpFiles.value.length >= 1000) {
		breakFlag.value = true
		return
	}
	tmpFiles.value.push(convertFileToUploadFile(file, subPath))
}

function convertFileToUploadFile(file: File, subPath: string) {
	return {
		name: subPath || file.name,
		size: file.size,
		status: "ready" as const,
		uid: Date.now(),
		raw: file
	}
}

/* -------------------- 拖拽 -------------------- */
const handleDragover = (e: DragEvent) => e.preventDefault()
const handleDragleave = (e: DragEvent) => e.preventDefault()

async function handleDrop(e: DragEvent) {
	initTempFiles()
	e.preventDefault()
	const items = e.dataTransfer?.items
	if (!items) return

	const entries = Array.from(items)
		.map(i => i.webkitGetAsEntry())
		.filter(Boolean)
	await Promise.all(entries.map(en => traverseFileTree(en)))
	if (!breakFlag.value) {
		uploaderFiles.value.push(...tmpFiles.value)
	} else {
		MsgWarning(t("file.uploadOverLimit"))
	}
	initTempFiles()
}

async function traverseFileTree(entry: any, rootPath = "") {
	if (!entry) return
	if (entry.isFile) {
		await new Promise<void>(res =>
			entry.file((f: File) => {
				const fullPath = (entry.fullPath || entry.name).replace(/^\//, "")
				handleFile(f, fullPath)
				res()
			})
		)
	} else if (entry.isDirectory) {
		const reader = entry.createReader()
		const readDir = async () => {
			const entries: any[] = await new Promise(r => reader.readEntries(r))
			if (!entries.length) return
			for (const en of entries) {
				await traverseFileTree(en, rootPath)
				if (breakFlag.value) return
			}
			await readDir()
		}
		await readDir()
	}
}

function initTempFiles() {
	tmpFiles.value = []
	breakFlag.value = false
}

/* -------------------- 列表操作 -------------------- */
const removeFile = (index: number) => uploaderFiles.value.splice(index, 1)
function clearFiles() {
	uploaderFiles.value = []
}

/* -------------------- 上传流程 -------------------- */
async function submit() {
	const files = uploaderFiles.value.slice()
	const paths = [...new Set(files.map(f => `${path.value}/${f.raw.webkitRelativePath || f.name}`))]
	const { data: exist } = await BatchCheckFiles(paths)

	if (exist.length) {
		const map = new Map(files.map(f => [`${path.value}/${f.raw.webkitRelativePath || f.name}`, f.size]))
		exist.forEach((e: any) => (e.uploadSize = map.get(e.path)))
		dialogExistFileRef.value.acceptParams({
			paths: exist,
			onConfirm: (action: "skip" | "overwrite", skipped: string[] = []) => handleFileUpload(action, skipped)
		})
	} else {
		await uploadFile(files)
	}
}

function handleFileUpload(action: "skip" | "overwrite", skipped: string[] = []) {
	const files = uploaderFiles.value.slice()
	if (action === "skip") {
		const filtered = files.filter(f => !skipped.includes(`${path.value}/${f.raw.webkitRelativePath || f.name}`))
		uploaderFiles.value = filtered
		uploadFile(filtered)
	} else {
		uploadFile(files)
	}
}

async function uploadFile(files: any[]) {
	if (!files.length) return clearFiles()
	loading.value = true
	let success = 0
	try {
		for (let i = 0; i < files.length; i++) {
			const f = files[i]
			uploadHelper.value = t("file.fileUploadStart", [f.name])
			const ok = f.size <= MAX_SINGLE_FILE_SIZE ? await uploadSingle(f) : await uploadLarge(f)
			if (ok) {
				success++
				f.status = "success"
			} else {
				f.status = "fail"
			}
		}
		if (success === files.length) {
			clearFiles()
			MsgSuccess(t("file.uploadSuccess"))
		}
	} finally {
		loading.value = false
		uploadHelper.value = ""
	}
}

async function uploadSingle(file: any) {
	const fd = new FormData()
	fd.append("file", file.raw)
	fd.append("path", getUploadPath(file))
	fd.append("overwrite", "true")
	uploadPercent.value = 0
	try {
		await UploadFileData(fd, {
			onUploadProgress: e => {
				const total = e.total || file.size || 1
				uploadPercent.value = Math.round((e.loaded / total) * 100)
			},
			timeout: 40000
		})
		return true
	} catch {
		return false
	}
}

async function uploadLarge(file: any) {
	const size = file.size
	const chunks = Math.ceil(size / CHUNK_SIZE)
	let uploaded = 0
	for (let c = 0; c < chunks; c++) {
		const start = c * CHUNK_SIZE
		const end = Math.min(start + CHUNK_SIZE, size)
		const chunk = file.raw.slice(start, end)
		const fd = new FormData()
		fd.append("filename", getFilename(file.name))
		fd.append("path", getUploadPath(file))
		fd.append("chunk", chunk)
		fd.append("chunkIndex", c.toString())
		fd.append("chunkCount", chunks.toString())
		try {
			await ChunkUploadFileData(fd, {
				onUploadProgress: e => {
					const total = e.total || chunk.size || 1
					uploadPercent.value = Math.round(((uploaded + e.loaded / total) * 100) / chunks)
				},
				timeout: TimeoutEnum.T_60S
			})
			uploaded++
		} catch {
			return false
		}
	}
	return uploaded === chunks
}

const getUploadPath = (file: any) => `${path.value}/${getDir(file.raw.webkitRelativePath || file.name)}`

const getDir = (p: string) => p.split("/").slice(0, -1).join("/") || ""
const getFilename = (p: string) => p.split("/").pop() || ""

defineExpose({ acceptParams })
</script>

<style scoped>
.upload-drop {
	border: 1px dashed #a0a0a0;
	border-radius: 6px;
}
.upload-drop:hover {
	border-color: #1352ff;
}
.sle {
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}
.flex-center {
	display: flex;
	align-items: center;
	justify-content: center;
}
</style>
