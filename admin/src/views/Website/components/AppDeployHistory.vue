<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <div>
    <n-drawer
      v-model:show="visible"
      :width="700"
      placement="right"
    >
      <n-drawer-content
        :title="`部署记录 - ${website?.primaryDomain || ''}`"
        closable
      >
        <div class="space-y-4">
          <div
            v-if="bindingRuntimeText"
            class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
          >
            <div class="font-medium text-slate-800">当前绑定目标</div>
            <div class="mt-1">{{ bindingRuntimeText }}</div>
          </div>
          <div class="flex justify-between items-center mb-6">
            <div class="text-sm text-slate-500">
              <span v-if="website?.type === 'proxy' && website?.appInstallId">
                部署记录用于回滚与审计；您也可以为当前容器配置生成快照。
              </span>
              <span v-else-if="isImageWebsiteSource(website)">
                部署记录用于回滚与审计；也可直接指定镜像标签发起部署。
              </span>
              <span v-else>
                部署记录用于回滚与审计。
              </span>
            </div>
            <div class="flex space-x-2">
              <n-button
                type="default"
                size="small"
                @click="fetchData"
              >刷新状态</n-button>
              <n-button
                v-if="website?.type === 'static'"
                type="primary"
                size="small"
                @click="showUploadModal = true"
              >手动发布</n-button>
              <n-button
                v-if="isImageWebsiteSource(website)"
                type="primary"
                size="small"
                @click="openImageDeployModal"
              >部署镜像</n-button>
              <n-button
                v-if="website?.type === 'proxy' && website?.appInstallId"
                type="primary"
                size="small"
                @click="handleCreateSnapshot"
                :loading="snapshotLoading"
              >生成快照</n-button>
            </div>
          </div>

          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium text-slate-800">部署历史</div>
              <div class="text-xs text-slate-400">记录实际生效过的部署结果</div>
            </div>
            <n-data-table
              :loading="loading"
              :columns="columns"
              :data="deployments"
              :bordered="true"
              size="small"
            />
          </div>

          <div
            v-if="selectedDeploy"
            class="mt-8"
          >
            <div class="font-semibold mb-2">版本日志 ({{ selectedDeploy.version }})</div>
            <div class="bg-black text-green-400 p-4 rounded-md font-mono text-xs h-64 overflow-y-auto whitespace-pre-wrap">
              {{ selectedDeploy.logText || '暂无日志信息' }}
            </div>
          </div>
        </div>
      </n-drawer-content>
    </n-drawer>

    <!-- Upload Modal -->
    <n-modal
      v-model:show="showUploadModal"
      preset="dialog"
      title="手动发布"
    >
      <div class="mt-4">
        <div
          v-if="bindingRuntimeText"
          class="mb-4 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
        >
          {{ bindingRuntimeText }}
        </div>
        <div class="mb-4 text-slate-500 text-sm">上传 `.zip` 产物后，系统会创建一条新的部署记录并切换流量。</div>
        <n-upload
          :custom-request="customRequest"
          accept=".zip"
        >
          <n-button>选择 .zip 文件上传</n-button>
        </n-upload>
      </div>
    </n-modal>

    <!-- Image Deploy Modal -->
    <n-modal
      v-model:show="showImageDeployModal"
      preset="dialog"
      title="部署镜像"
      :show-icon="false"
    >
      <div class="mt-4">
        <div
          v-if="bindingRuntimeText"
          class="mb-4 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
        >
          {{ bindingRuntimeText }}
        </div>
        <div class="mb-4 text-slate-500 text-sm">
          请输入或选择要部署的容器镜像地址。例如：<code>shoply-base:v1.0.1</code>。系统将自动拉取并启动新容器，完成版本更迭。
        </div>
        <n-auto-complete
          v-model:value="targetImageTag"
          :options="localImageOptions"
          placeholder="例如: nginx:latest"
          :get-show="() => true"
          clearable
        />
        <div class="mt-6 flex justify-end">
          <n-button
            type="primary"
            @click="handleImageDeploy"
            :loading="imageDeployLoading"
            :disabled="!targetImageTag"
          >
            开始部署
          </n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, h } from "vue"
import { NButton, NTag, NSpace, useMessage, useDialog, NModal, NUpload } from "naive-ui"
import type { DataTableColumns, UploadCustomRequestOptions } from "naive-ui"
import { formatTime } from "@/utils/date"
import { UploadFileData, ChunkUploadFileData } from "@/api/modules/file"
import { hasWebsiteRuntimeMeta, isImageWebsiteSource, resolveWebsiteBindingMeta } from "@/utils/websiteRuntime"
import { AppDeployDeleteAPI, AppDeployListAPI, AppDeploySwitchAPI, AppDeployTriggerAPI, AppDeploySnapshotAPI } from "@/api/modules/website"
import { listAllImage } from "@/api/modules/container"
import { ListAppInstalled } from "@/api/modules/apps"
import type { Website } from "@/api/interface/website"
import type { App } from "@/api/interface/apps"
import type { Container } from "@/api/interface/container"

type WebsiteVersionTarget = Website.WebsiteDTO & {
	sourceType?: string
	engineEnv?: string
	status?: string | boolean
}

type AppDeployRow = Website.AppDeployRecord & {
	createdAt?: string
}

const visible = ref(false)
const showUploadModal = ref(false)
const showImageDeployModal = ref(false)
const loading = ref(false)
const snapshotLoading = ref(false)
const imageDeployLoading = ref(false)
const deployments = ref<AppDeployRow[]>([])
const website = ref<WebsiteVersionTarget | null>(null)
const selectedDeploy = ref<AppDeployRow | null>(null)
const localImageOptions = ref<{ label: string; value: string }[]>([])
const targetImageTag = ref("")
const appInstallMap = ref<Record<number, App.AppInstalledInfo>>({})

const message = useMessage()
const dialog = useDialog()

const emit = defineEmits(["confirm"])

function getErrorMessage(error: unknown, fallback: string) {
	if (error && typeof error === "object") {
		const maybe = error as { message?: string }
		if (typeof maybe.message === "string" && maybe.message.trim()) {
			return maybe.message
		}
	}
	return fallback
}

function getDeploySourceLabel(row: AppDeployRow) {
	switch (String(row.sourceType || "").trim()) {
		case "image":
			return "手动镜像"
		case "upload":
			return "手动归档"
		case "compose":
			return "配置快照"
		default:
			return "部署记录"
	}
}

function getDeploySourceTagType(row: AppDeployRow): "success" | "info" | "warning" | "default" {
	switch (String(row.sourceType || "").trim()) {
		case "image":
		case "upload":
			return "default"
		case "compose":
			return "success"
		default:
			return "default"
	}
}

const bindingRuntimeText = computed(() => {
	if (!website.value) return ""
	const binding = resolveWebsiteBindingMeta(website.value, {
		appInstallMap: appInstallMap.value
	}, {
		includeSourceInDetail: true,
		kindFallback: "Runtime",
		userFallback: "镜像默认",
		runtimePrefix: "运行时：",
		runUserPrefix: "用户："
	})
	if (binding) return binding.detail
	if (isImageWebsiteSource(website.value)) {
		return `自定义镜像 · 当前镜像：${website.value.engineEnv || website.value.proxy || "-"}`
	}
	return ""
})

async function handleCreateSnapshot() {
	snapshotLoading.value = true
	try {
		await AppDeploySnapshotAPI({ websiteId: website.value.id })
		message.success("配置快照创建成功")
		fetchData()
	} catch (error) {
		message.error("创建快照失败")
	} finally {
		snapshotLoading.value = false
	}
}

const UPLOAD_DIR = "/opt/gopanel/upload"
// 服务端默认请求体大小限制在 4MB 左右，超过这个大小的文件必须走分片上传，否则会被拒绝
const MAX_SINGLE_FILE_SIZE = 1024 * 1024 * 4
const CHUNK_SIZE = 1024 * 1024 * 3

async function uploadSingle(file: File, onProgress: UploadCustomRequestOptions["onProgress"]) {
	const formData = new FormData()
	formData.append("file", file)
	formData.append("path", UPLOAD_DIR)
	await UploadFileData(formData, {
		onUploadProgress: ({ loaded, total }) => {
			onProgress({ percent: Math.round((loaded / (total || 1)) * 100) })
		}
	})
}

async function uploadLarge(file: File, onProgress: UploadCustomRequestOptions["onProgress"]) {
	const chunkCount = Math.ceil(file.size / CHUNK_SIZE)
	for (let i = 0; i < chunkCount; i++) {
		const start = i * CHUNK_SIZE
		const end = Math.min(start + CHUNK_SIZE, file.size)
		const formData = new FormData()
		formData.append("filename", file.name)
		formData.append("path", UPLOAD_DIR)
		formData.append("chunk", file.slice(start, end))
		formData.append("chunkIndex", i.toString())
		formData.append("chunkCount", chunkCount.toString())
		formData.append("overwrite", "true")
		await ChunkUploadFileData(formData, {})
		onProgress({ percent: Math.round(((i + 1) / chunkCount) * 100) })
	}
}

async function customRequest({ file, onFinish, onError, onProgress }: UploadCustomRequestOptions) {
	if (!file.file) {
		onError()
		return
	}
	const rawFile = file.file

	try {
		if (rawFile.size <= MAX_SINGLE_FILE_SIZE) {
			await uploadSingle(rawFile, onProgress)
		} else {
			await uploadLarge(rawFile, onProgress)
		}

		// If upload succeeds, trigger deployment
		await AppDeployTriggerAPI({
			websiteId: website.value.id,
			zipPath: `${UPLOAD_DIR}/${rawFile.name}`
		})

		showUploadModal.value = false
		message.success("上传成功，开始后台部署...")
		fetchData()
		emit("confirm")
		onFinish()
	} catch (error) {
		message.error(getErrorMessage(error, "上传或部署失败"))
		onError()
	}
}

async function openImageDeployModal() {
	showImageDeployModal.value = true
	targetImageTag.value = website.value.engineEnv || ""
	try {
		const res = await listAllImage()
		if (res.data) {
			localImageOptions.value = res.data.map((item: Container.ImageInfo) => ({
				label: item.tags && item.tags.length > 0 ? item.tags[0] : item.name,
				value: item.tags && item.tags.length > 0 ? item.tags[0] : item.name
			}))
		}
	} catch (error) {
		console.error(error)
	}
}

async function loadBindingMeta() {
	if (!website.value) return
	try {
		if (website.value.appInstallId) {
			const res = await ListAppInstalled()
			const list: App.AppInstalledInfo[] = Array.isArray(res.data) ? res.data : []
			appInstallMap.value = Object.fromEntries(list.map((item) => [item.id, item]))
		}
	} catch (error) {
		console.error(error)
	}
}

async function handleImageDeploy() {
	if (!targetImageTag.value) {
		message.warning("请输入镜像地址")
		return
	}
	imageDeployLoading.value = true
	try {
		await AppDeployTriggerAPI({
			websiteId: website.value.id,
			zipPath: "",
			imageTag: targetImageTag.value
		})
		message.success("新镜像部署任务已提交")
		showImageDeployModal.value = false
		fetchData()
		emit("confirm")
	} catch (error) {
		message.error(getErrorMessage(error, "提交部署失败"))
	} finally {
		imageDeployLoading.value = false
	}
}

function open(row: WebsiteVersionTarget) {
	website.value = row
	visible.value = true
	selectedDeploy.value = null
	appInstallMap.value = {}
	if (!hasWebsiteRuntimeMeta(row)) {
		loadBindingMeta()
	}
	fetchData()
}

async function fetchDeployments() {
	if (!website.value) return
	loading.value = true
	try {
		const res = await AppDeployListAPI({ websiteId: website.value.id })
		deployments.value = Array.isArray(res.data) ? (res.data as AppDeployRow[]) : []
	} catch (error) {
		console.error(error)
	} finally {
		loading.value = false
	}
}

async function fetchData() {
	await fetchDeployments()
}

function viewLogs(row: AppDeployRow) {
	selectedDeploy.value = row
}

function handleRollback(row: AppDeployRow) {
	dialog.warning({
		title: "切换版本确认",
		content: `确认切换到部署记录 [${row.version}] 吗？`,
		positiveText: "确认切换",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await AppDeploySwitchAPI({ deployId: row.id })
				message.success("切换成功")
				fetchData()
				emit("confirm")
			} catch (error) {
				message.error(getErrorMessage(error, "版本切换失败"))
			}
		}
	})
}

function handleDelete(row: AppDeployRow) {
	dialog.warning({
		title: "删除部署记录",
		content: `确认删除部署记录 [${row.version}] 吗？删除后不可恢复。`,
		positiveText: "确认删除",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await AppDeployDeleteAPI({ deployId: row.id })
				message.success("删除成功")
				fetchData()
			} catch (error) {
				message.error(getErrorMessage(error, "版本删除失败"))
			}
		}
	})
}

const columns: DataTableColumns<AppDeployRow> = [
	{
		title: "部署记录",
		key: "version",
		render(row: AppDeployRow) {
			return h("div", { class: "flex flex-col gap-1" }, [
				h("div", { class: "flex items-center space-x-2" }, [
					h("span", { class: "font-mono" }, row.version),
					row.isActive ? h(NTag, { type: "success", size: "small", round: true }, { default: () => "线上运行中" }) : null,
					h(NTag, { type: getDeploySourceTagType(row), size: "small", round: true }, { default: () => getDeploySourceLabel(row) })
				]),
				row.imageTag ? h("div", { class: "font-mono text-xs text-gray-500" }, row.imageTag) : null
			])
		}
	},
	{
		title: "创建时间",
		key: "createdAt",
		render(row: AppDeployRow) {
			return formatTime(row.createdAt || "")
		}
	},
	{
		title: "状态",
		key: "status",
		render(row: AppDeployRow) {
			const type = row.status === "Running" ? "success" : (row.status === "Failed" ? "error" : "info")
			return h(NTag, { type: type, size: "small" }, { default: () => row.status })
		}
	},
	{
		title: "操作",
		key: "actions",
		width: 220,
		render(row: AppDeployRow) {
			const btns = [
				h(
					NButton,
					{
						size: "small",
						type: "info",
						quaternary: true,
						onClick: () => viewLogs(row)
					},
					{ default: () => "查看日志" }
				)
			]
			
			if (!row.isActive) {
				btns.push(
					h(
						NButton,
						{
							size: "small",
							type: "warning",
							quaternary: true,
							style: { marginLeft: '8px' },
							onClick: () => handleRollback(row)
						},
						{ default: () => "切换上线" }
					)
				)
				btns.push(
					h(
						NButton,
						{
							size: "small",
							type: "error",
							quaternary: true,
							onClick: () => handleDelete(row)
						},
						{ default: () => "删除" }
					)
				)
			}
			
			return h(NSpace, null, { default: () => btns })
		}
	}
]

defineExpose({
	open
})
</script>
