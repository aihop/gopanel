<template>
  <n-drawer
    v-model:show="upVisible"
    width="50%"
    :mask-closable="false"
  >
    <n-drawer-content>
      <template #header>
        <DrawerHeader
          :header="t('commons.button.import')"
          :resource="title"
          :back="handleClose"
        />
      </template>

      <div style="padding: 16px">
        <n-spin :show="loading">
          <n-alert
            v-if="type === 'mysql' || type === 'mariadb'"
            type="error"
            :title="t('database.formatHelper', [remark])"
          />
          <n-alert
            v-if="type === 'website'"
            type="warning"
            :title="t('website.websiteBackupWarn')"
            :closable="false"
            class="mb-4"
          />

          <!-- 拖拽上传区域 -->
          <div
            class="upload-dropzone"
            @dragover.prevent="onDragOver"
            @drop.prevent="onDrop"
          >
            <input
              ref="fileInput"
              class="file-input"
              type="file"
              @change="onFileSelect"
            />
            <div class="upload-inner">
              <div class="upload-icon">{{ t("database.dropHelper") }}</div>
              <div class="upload-sub">{{ t("database.clickHelper") }}</div>
              <div
                v-if="isUpload"
                class="progress-wrap mt-3"
              >
                <div class="progress-bar">
                  <div
                    class="progress-fill"
                    :style="{ width: uploadPercent + '%' }"
                  ></div>
                </div>
                <div class="progress-text">{{ uploadPercent }}%</div>
              </div>
              <div
                class="upload-tips"
                v-if="type === 'mysql' || type === 'mariadb' || type === 'postgresql'"
              >
                <div class="input-help">{{ t("database.supportUpType") }}</div>
                <div class="input-help">{{ t("database.zipFormat") }}</div>
              </div>
              <div
                class="upload-tips"
                v-else
              >
                <div class="input-help">{{ t("website.supportUpType") }}</div>
                <div class="input-help">{{ t("website.zipFormat", [type + ".json"]) }}</div>
              </div>
            </div>
          </div>

          <div style="margin-top: 12px; display: flex; gap: 12px; align-items: center">
            <n-button
              :disabled="isUpload || uploaderFiles.length !== 1"
              @click="onSubmit"
              type="primary"
            >
              {{ t("commons.button.upload") }}
            </n-button>
          </div>

          <n-divider />

          <div class="mb-4 flex items-center justify-between">
            <n-space>
              <n-button
                :disabled="selects.length === 0"
                @click="onBatchDelete(null)"
              >
                {{ t("commons.button.delete") }}
              </n-button>
              <div
                v-if="selects.length"
                style="color: var(--fg-secondary-color); font-size: 12px"
              >
                {{ selects.length }} {{ t("commons.selected") }}
              </div>
            </n-space>
          </div>

          <n-data-table
            :columns="columns"
            :data="data"
            :row-key="rowKey"
            :loading="loading"
            :pagination="paginationOptions"
            remote
            v-model:checked-rows="selects"
            @update:page="onPageChange"
            @update:pageSize="onPageSizeChange"
            :scroll-x="720"
          />
        </n-spin>
      </div>

      <n-modal
        v-model:show="open"
        :title="t('commons.button.recover') + ' - ' + name"
        :mask-closable="false"
        preset="card"
      >
        <n-form :model="{ secret }">
          <n-form-item
            v-if="type === 'app' || type === 'website'"
            :label="t('setting.compressPassword')"
          >
            <n-input
              v-model:value="secret"
              :placeholder="t('setting.backupRecoverMessage')"
            />
          </n-form-item>
        </n-form>
        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 12px">
            <n-button
              @click="handleBackupClose"
              :disabled="loading"
            >
              {{ t("commons.button.cancel") }}
            </n-button>
            <n-button
              type="primary"
              @click="onHandleRecover"
              :loading="loading"
            >
              {{ t("commons.button.confirm") }}
            </n-button>
          </div>
        </template>
      </n-modal>

      <OpDialog
        ref="opRef"
        @search="search"
      />
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import type { AxiosProgressEvent } from "axios"
import { ref, reactive, h } from "vue"
import { useI18n } from "vue-i18n"
import {
	useDialog,
	NDrawer,
	NAlert,
	NButton,
	NDataTable,
	NModal,
	NForm,
	NFormItem,
	NInput,
	NDivider,
	NSpace,
	NSpin
} from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import OpDialog from "@/components/OpDialog.vue"
import { computeSize } from "@/utils/util"
import { GetUploadList, CheckFile, ChunkUploadFileData, BatchDeleteFile } from "@/api/modules/file"
import { settingSystemBaseDirAPI } from "@/api/modules/setting"
import { backupRecoverByUploadAPI } from "@/api/modules/backup"
import { MsgError, MsgSuccess } from "@/utils/message"
import { File } from "@/api/interface/file"
const { t } = useI18n()
const dialog = useDialog()

const loading = ref(false)
const isUpload = ref(false)
const uploadPercent = ref(0)
const selects = ref<any[]>([])
const baseDir = ref("")
const opRef = ref<any>(null)

const open = ref(false)
const currentRow = ref<File.File | null>(null)

const data = ref<any[]>([])
const title = ref("")
const paginationConfig = reactive({
	currentPage: 1,
	limit: 10,
	total: 0
})

const upVisible = ref(false)
const type = ref("")
const name = ref("")
const detailName = ref("")
const detailId = ref(0)
const remark = ref("")
const secret = ref("")

const uploaderFiles = ref<File[]>([])
const fileInput = ref<HTMLInputElement | null>(null)

const paginationOptions = {
	page: paginationConfig.currentPage,
	limit: paginationConfig.limit,
	pageCount: Math.max(1, Math.ceil((paginationConfig.total || 0) / paginationConfig.limit)),
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100],
	showQuickJumper: true,
	itemCount: paginationConfig.total
}

const rowKey = (row: any) => row.name

const columns: any = [
	{ type: "selection", width: 48 },
	{ title: t("commons.table.name"), key: "name", ellipsis: true },
	{
		title: t("file.size"),
		key: "size",
		width: 120,
		render(row: any) {
			return row.size ? computeSize(row.size) : "-"
		}
	},
	{
		title: t("commons.table.createdAt"),
		key: "createdAt",
		width: 160
	},
	{
		title: t("commons.table.operate"),
		key: "actions",
		width: 150,
		render(row: any) {
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{ size: "small", onClick: () => onRecover(row) },
						{ default: () => t("commons.button.recover") }
					),
					h(
						NButton,
						{ size: "small", onClick: () => onBatchDelete(row) },
						{ default: () => t("commons.button.delete") }
					)
				]
			})
		}
	}
]

const acceptParams = async (params: {
	type: string
	name: string
	detailName: string
	detailId: number
	remark: string
}) => {
	type.value = params.type
	name.value = params.name
	detailName.value = params.detailName
	detailId.value = params.detailId
	remark.value = params.remark
	const pathRes = await settingSystemBaseDirAPI()
	if (type.value === "mysql" || type.value === "mariadb" || type.value === "postgresql") {
		title.value = `${name.value} [ ${detailName.value} ]`
		baseDir.value = detailName.value
			? `${pathRes.data}/uploads/database/${type.value}/${name.value}/${detailName.value}/`
			: `${pathRes.data}/uploads/database/${type.value}/${name.value}/`
	} else if (type.value === "website") {
		title.value = name.value
		baseDir.value = `${pathRes.data}/uploads/website/${type.value}/${detailName.value}/`
	} else if (type.value === "app") {
		title.value = name.value
		baseDir.value = `${pathRes.data}/uploads/app/${type.value}/${name.value}/`
	}

	upVisible.value = true
	search()
}

const search = async () => {
	loading.value = true
	try {
		const params = {
			page: paginationConfig.currentPage,
			limit: paginationConfig.limit,
			path: baseDir.value
		}
		const res = await GetUploadList(params)
		data.value = res.data.items || []
		paginationConfig.total = res.data.total || 0
	} catch (e) {
		console.error(e)
	} finally {
		loading.value = false
	}
}

const onRecover = async (row: File.File) => {
	currentRow.value = row
	if (type.value !== "app" && type.value !== "website") {
		await dialog.warning({
			title: t("commons.button.recover"),
			content: t("commons.msg.recoverHelper", [row.name]),
			positiveText: t("commons.button.confirm"),
			negativeText: t("commons.button.cancel"),
			onPositiveClick: () => onHandleRecover()
		})
		return
	}
	open.value = true
}

const handleBackupClose = () => {
	open.value = false
}

const onHandleRecover = async () => {
	if (!currentRow.value) return
	const params = {
		source: "LOCAL",
		type: type.value,
		name: name.value,
		detailName: detailName.value,
		detailId: detailId.value,
		file: baseDir.value + currentRow.value.name,
		secret: secret.value,
		recoverType: "upload"
	}
	loading.value = true
	try {
		await backupRecoverByUploadAPI(params)
		MsgSuccess(t("commons.msg.operationSuccess"))
		handleBackupClose()
		search()
	} catch {
		// ignore
	} finally {
		loading.value = false
	}
}

const onFileSelect = (e: Event) => {
	const input = e.target as HTMLInputElement
	const f = input.files && input.files[0]
	if (f) {
		uploaderFiles.value = [f as unknown as File]
	}
}

const onDragOver = (e: DragEvent) => {
	e.dataTransfer && (e.dataTransfer.dropEffect = "copy")
}

const onDrop = (e: DragEvent) => {
	const f = e.dataTransfer?.files[0]
	if (f) uploaderFiles.value = [f as unknown as File]
}

const validateFile = async (file: any) => {
	if (!file.name) {
		MsgError(t("commons.msg.fileNameErr"))
		return false
	}
	const reg = /^[a-zA-Z0-9\u4e00-\u9fa5]{1}[a-z:A-Z0-9_.\u4e00-\u9fa5-]{0,256}$/
	if (!reg.test(file.name)) {
		MsgError(t("commons.msg.fileNameErr"))
		return false
	}
	const res = await CheckFile(baseDir.value + file.name)
	if (res.data) {
		MsgError(t("commons.msg.fileExist"))
		return false
	}
	if (type.value === "app" || type.value === "website") {
		if (!file.name.endsWith(".tar.gz")) {
			MsgError(t("commons.msg.unSupportType"))
			return false
		}
	} else {
		if (!file.name.endsWith(".sql") && !file.name.endsWith(".tar.gz") && !file.name.endsWith(".sql.gz")) {
			MsgError(t("commons.msg.unSupportType"))
			return false
		}
	}
	return true
}

const submitUpload = async (file: any) => {
	isUpload.value = true
	const CHUNK_SIZE = 1024 * 1024
	const fileSize = file.size
	const chunkCount = Math.ceil(fileSize / CHUNK_SIZE)
	let uploadedChunkCount = 0
	uploadPercent.value = 0

	for (let i = 0; i < chunkCount; i++) {
		const start = i * CHUNK_SIZE
		const end = Math.min(start + CHUNK_SIZE, fileSize)
		const chunk = file.slice(start, end)
		const formData = new FormData()
		formData.append("filename", file.name)
		formData.append("path", baseDir.value)
		formData.append("chunk", chunk)
		formData.append("chunkIndex", i.toString())
		formData.append("chunkCount", chunkCount.toString())

		try {
			await ChunkUploadFileData(formData, {
				onUploadProgress: ((progressEvent: AxiosProgressEvent) => {
					const progress = Math.round(
						((uploadedChunkCount + progressEvent.loaded / (progressEvent.total || 1)) * 100) / chunkCount
					)
					uploadPercent.value = progress
				}) as any
			})
			uploadedChunkCount++
		} catch (err) {
			isUpload.value = false
			MsgError(t("commons.msg.uploadFailed"))
			return
		}

		if (uploadedChunkCount === chunkCount) {
			isUpload.value = false
			uploaderFiles.value = []
			uploadPercent.value = 100
			MsgSuccess(t("file.uploadSuccess"))
			search()
		}
	}
}

const onSubmit = async () => {
	if (uploaderFiles.value.length !== 1) return
	const file = uploaderFiles.value[0]
	if (!(await validateFile(file))) return
	await submitUpload(file)
}

const onBatchDelete = async (row: File.File | null) => {
	const files: string[] = []
	const names: string[] = []
	if (row) {
		files.push(baseDir.value + row.name)
		names.push(row.name)
	} else {
		selects.value.forEach((item: any) => {
			files.push(baseDir.value + item.name)
			names.push(item.name)
		})
	}
	opRef.value?.acceptParams({
		title: t("commons.button.delete"),
		names,
		msg: t("commons.msg.operatorHelper", [t("commons.button.import"), t("commons.button.delete")]),
		api: BatchDeleteFile,
		params: { paths: files, isDir: false }
	})
}

const onPageChange = (p: number) => {
	paginationConfig.currentPage = p
	search()
}
const onPageSizeChange = (size: number) => {
	paginationConfig.limit = size
	paginationConfig.currentPage = 1
	search()
}

const handleClose = () => {
	upVisible.value = false
	uploaderFiles.value = []
	isUpload.value = false
	uploadPercent.value = 0
	loading.value = false
	paginationConfig.currentPage = 1
	paginationConfig.limit = 10
	paginationConfig.total = 0
	data.value = []
	selects.value = []
	secret.value = ""
}

defineExpose({ acceptParams })
</script>

<style scoped>
.upload-dropzone {
	border: 1px dashed var(--border-color);
	border-radius: 6px;
	padding: 28px;
	display: flex;
	align-items: center;
	justify-content: center;
	position: relative;
	cursor: pointer;
	user-select: none;
}
.file-input {
	position: absolute;
	inset: 0;
	opacity: 0;
	cursor: pointer;
	width: 100%;
	height: 100%;
}
.upload-inner {
	text-align: center;
	pointer-events: none;
}
.upload-icon {
	font-weight: 600;
	margin-bottom: 6px;
}
.upload-sub {
	font-size: 12px;
	color: var(--fg-secondary-color);
}
.progress-wrap {
	margin-top: 12px;
}
.progress-bar {
	width: 320px;
	height: 10px;
	background: rgba(0, 0, 0, 0.06);
	border-radius: 6px;
	overflow: hidden;
	margin: 0 auto;
}
.progress-fill {
	height: 100%;
	background: linear-gradient(90deg, #3b82f6, #60a5fa);
	width: 0%;
	transition: width 0.2s ease;
}
.progress-text {
	margin-top: 6px;
	font-size: 12px;
	color: var(--fg-secondary-color);
}
</style>
