<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <div>
    <n-drawer
      v-model:show="visible"
      :width="700"
      placement="right"
    >
      <n-drawer-content
        :title="`部署历史与版本管理 - ${website?.primaryDomain || ''}`"
        closable
      >
        <div class="space-y-4">
          <div class="flex justify-between items-center mb-6">
            <div class="text-sm text-slate-500">
              <span v-if="website?.type === 'deployment'">
                您可以创建当前容器配置的快照。切换版本时，系统会使用当时的 docker-compose 重新部署容器，保留数据挂载卷。
              </span>
              <span v-else-if="website?.codeSource === 'git'">
                您可以指定一个新的容器镜像标签（Tag）进行发布。系统将拉起新版本容器并无缝切换流量，历史版本支持一键秒级回滚。
              </span>
              <span v-else>
                您可以在此处查看应用的历史部署记录，并随时将流量切换到任意正常运行的历史版本。若该站点绑定了流水线 Runner，当前线上流量通常由流水线产出的 Runner 容器托管，网站仅负责代理桥接。
              </span>
            </div>
            <div class="flex space-x-2">
              <n-button
                type="default"
                size="small"
                @click="fetchData"
              >刷新状态</n-button>
              <n-button
                v-if="website?.type !== 'web_app' && website?.codeSource !== 'git'"
                type="primary"
                size="small"
                @click="showUploadModal = true"
              >发布新版本</n-button>
              <n-button
                v-if="website?.codeSource === 'git'"
                type="primary"
                size="small"
                @click="openImageDeployModal"
              >发布新镜像版本</n-button>
              <n-button
                v-if="website?.type === 'web_app' && website?.sourceType === 'docker-compose'"
                type="primary"
                size="small"
                @click="handleCreateSnapshot"
                :loading="snapshotLoading"
              >创建配置快照</n-button>
            </div>
          </div>

          <n-data-table
            :loading="loading"
            :columns="columns"
            :data="deployments"
            :bordered="true"
            size="small"
          />

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
      title="发布新版本"
    >
      <div class="mt-4">
        <div class="mb-4 text-slate-500 text-sm">上传包含最新代码的压缩包 (.zip)。系统将自动解压并创建一个新版本，然后把流量切换到新版本上。</div>
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
      title="发布新镜像版本"
      :show-icon="false"
    >
      <div class="mt-4">
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
// @ts-nocheck
import { ref, h } from "vue"
import { NButton, NTag, NSpace, useMessage, useDialog, NModal, NUpload } from "naive-ui"
import { formatTime } from "@/utils/date"
import { UploadFileData } from "@/api/modules/file"
import type { UploadCustomRequestOptions } from "naive-ui"
import { WebsiteDeployDeleteAPI, WebsiteDeployListAPI, WebsiteDeploySwitchAPI, WebsiteDeployTriggerAPI, WebsiteDeploySnapshotAPI } from "@/api/modules/website"
import { listAllImage } from "@/api/modules/container"

const visible = ref(false)
const showUploadModal = ref(false)
const showImageDeployModal = ref(false)
const loading = ref(false)
const snapshotLoading = ref(false)
const imageDeployLoading = ref(false)
const deployments = ref<any[]>([])
const website = ref<any>(null)
const selectedDeploy = ref<any>(null)

const localImageOptions = ref<{ label: string; value: string }[]>([])
const targetImageTag = ref("")

const message = useMessage()
const dialog = useDialog()

const emit = defineEmits(["confirm"])

async function handleCreateSnapshot() {
	snapshotLoading.value = true
	try {
		await WebsiteDeploySnapshotAPI({ websiteId: website.value.id })
		message.success("配置快照创建成功")
		fetchData()
	} catch (error) {
		message.error("创建快照失败")
	} finally {
		snapshotLoading.value = false
	}
}

async function customRequest({ file, onFinish, onError, onProgress }: UploadCustomRequestOptions) {
	if (!file.file) {
		onError()
		return
	}

	const formData = new FormData()
	formData.append("file", file.file)
	// Put the file into /opt/gopanel/upload (or a generic upload dir)
	formData.append("path", "/opt/gopanel/upload")

	try {
		await UploadFileData(formData, {
			onUploadProgress: ({ loaded, total }) => {
				onProgress({ percent: Math.round((loaded / (total || 1)) * 100) })
			}
		})
		
		// If upload succeeds, trigger deployment
		await WebsiteDeployTriggerAPI({
			websiteId: website.value.id,
			zipPath: `/opt/gopanel/upload/${file.file.name}`
		})

		showUploadModal.value = false
		message.success("上传成功，开始后台部署...")
		fetchData()
		emit("confirm")
		onFinish()
	} catch (err: any) {
		message.error(err.message || "上传或部署失败")
		onError()
	}
}

async function openImageDeployModal() {
	showImageDeployModal.value = true
	targetImageTag.value = website.value.engineEnv || ""
	try {
		const res = await listAllImage()
		if (res.data) {
			localImageOptions.value = res.data.map((item: any) => ({
				label: item.tags && item.tags.length > 0 ? item.tags[0] : item.name,
				value: item.tags && item.tags.length > 0 ? item.tags[0] : item.name
			}))
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
		await WebsiteDeployTriggerAPI({
			websiteId: website.value.id,
			zipPath: "",
			imageTag: targetImageTag.value
		})
		message.success("新镜像部署任务已提交")
		showImageDeployModal.value = false
		fetchData()
		emit("confirm")
	} catch (error: any) {
		message.error(error.message || "提交部署失败")
	} finally {
		imageDeployLoading.value = false
	}
}

function open(row: any) {
	website.value = row
	visible.value = true
	selectedDeploy.value = null
	fetchData()
}

async function fetchData() {
	if (!website.value) return
	loading.value = true
	try {
		const res = await WebsiteDeployListAPI({ websiteId: website.value.id })
		deployments.value = res.data || []
	} catch (error) {
		console.error(error)
	} finally {
		loading.value = false
	}
}

function viewLogs(row: any) {
	selectedDeploy.value = row
}

function handleRollback(row: any) {
	dialog.warning({
		title: "切换版本确认",
		content: `确认将应用流量切换到版本 [${row.version}] 吗？切换过程通常在 1 秒内完成。`,
		positiveText: "确认切换",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await WebsiteDeploySwitchAPI({ deployId: row.id })
				message.success("版本切换成功")
				fetchData()
				emit("confirm")
			} catch (error: any) {
				message.error(error.message || "版本切换失败")
			}
		}
	})
}

function handleDelete(row: any) {
	dialog.warning({
		title: "删除版本确认",
		content: `确认删除版本 [${row.version}] 吗？删除后将无法恢复该条部署历史。`,
		positiveText: "确认删除",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await WebsiteDeployDeleteAPI({ deployId: row.id })
				message.success("版本删除成功")
				fetchData()
			} catch (error: any) {
				message.error(error.message || "版本删除失败")
			}
		}
	})
}

const columns = [
	{
		title: "版本标识",
		key: "version",
		render(row: any) {
			return h("div", { class: "flex flex-col gap-1" }, [
				h("div", { class: "flex items-center space-x-2" }, [
					h("span", { class: "font-mono" }, row.version),
					row.isActive ? h(NTag, { type: "success", size: "small", round: true }, { default: () => "线上运行中" }) : null
				]),
				row.imageTag ? h("div", { class: "font-mono text-xs text-gray-500" }, row.imageTag) : null
			])
		}
	},
	{
		title: "部署时间",
		key: "createdAt",
		render(row: any) {
			return formatTime(row.createdAt)
		}
	},
	{
		title: "状态",
		key: "status",
		render(row: any) {
			const type = row.status === "Running" ? "success" : (row.status === "Failed" ? "error" : "info")
			return h(NTag, { type: type, size: "small" }, { default: () => row.status })
		}
	},
	{
		title: "操作",
		key: "actions",
		width: 150,
		render(row: any) {
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
						{ default: () => "切换至此版本" }
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
