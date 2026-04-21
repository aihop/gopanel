<template>
  <div class="w-full">
    <n-spin :show="loading">
      <div
        v-if="apps.length"
        class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3"
      >
        <div
          v-for="item in apps"
          :key="item.id"
          class="relative overflow-hidden rounded-2xl border border-slate-200/80 bg-gradient-to-b from-white to-slate-50/80 p-5 shadow-sm transition hover:-translate-y-1 hover:border-blue-200/60 hover:shadow-md"
        >
          <div class="pointer-events-none absolute -top-10 -right-8 h-28 w-28 rounded-full bg-blue-500/10 blur-2xl"></div>

          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 flex-1 items-start gap-3">
              <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50/70">
                <img
                  v-if="item.app?.icon"
                  :src="item.app.icon"
                  alt="icon"
                  class="h-8 w-8 object-contain"
                />
                <span
                  v-else
                  class="text-base font-bold text-blue-600"
                >{{ item.name?.slice(0, 1)?.toUpperCase() }}</span>
              </div>

              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <div
                    class="truncate text-base font-semibold text-slate-900 hover:underline"
                    @click="showDrawer(item)"
                  >{{ item.name }}</div>
                  <n-tag
                    v-if="item.status"
                    :type="statusType(item.status)"
                    size="small"
                    round
                  >{{ statusLabel(item.status) }}</n-tag>
                  <n-button
                    v-if="canCancelInstall(item)"
                    tertiary
                    type="error"
                    size="tiny"
                    @click="cancelInstall(item)"
                  >取消安装</n-button>
                </div>
                <div class="mt-1 truncate text-sm text-slate-500">容器名：{{ item.containerName || "-" }}</div>
              </div>
            </div>

          </div>

          <p
            class="mt-4 line-clamp-3 min-h-[3.5rem] text-sm leading-6 text-slate-600"
            v-if="item.description"
          >
            {{ item.description   }}
          </p>

          <div class="mt-4 grid grid-cols-2 gap-2">
            <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
              <div class="text-xs text-slate-400">版本</div>
              <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.version || "-" }}</div>
            </div>
            <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
              <div class="text-xs text-slate-400">安装时间</div>
              <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.createdAt || "-" }}</div>
            </div>
            <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
              <div class="text-xs text-slate-400">HTTP 端口</div>
              <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.httpPort || "-" }}</div>
            </div>
            <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
              <div class="text-xs text-slate-400">HTTPS 端口</div>
              <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.httpsPort || "-" }}</div>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center justify-end gap-2">
            <n-button
              secondary
              size="small"
              @click="openLog(item)"
            >日志</n-button>
            <n-button
              secondary
              size="small"
              :disabled="disableStart(item)"
              @click="handleOperate(item, 'start')"
            >启动</n-button>
            <n-button
              secondary
              size="small"
              :disabled="disableStop(item)"
              @click="handleOperate(item, 'stop')"
            >停止</n-button>
            <n-button
              secondary
              size="small"
              :disabled="disableRestart(item)"
              @click="handleOperate(item, 'restart')"
            >重启</n-button>
            <n-button
              secondary
              type="warning"
              size="small"
              :disabled="disableRebuild(item)"
              @click="handleRebuild(item)"
            >重建</n-button>
            <n-button
              secondary
              type="error"
              size="small"
              :disabled="disableUninstall(item)"
              @click="openDeleteModal(item)"
            >卸载</n-button>
          </div>
        </div>
      </div>

      <div
        v-else
        class="py-16 text-center text-sm text-slate-400"
      >
        暂无已安装应用
      </div>
    </n-spin>

    <n-drawer
      :show="drawerVisible"
      @update:show="drawerVisible = $event"
      placement="right"
      width="400"
    >
      <n-drawer-content
        title="应用详情"
        v-if="drawerItem"
      >
        <pre class="whitespace-pre-wrap">{{ JSON.stringify(drawerItem.app, null, 2) }}</pre>
      </n-drawer-content>
    </n-drawer>

    <n-modal
      :show="showDeleteModal"
      @update:show="showDeleteModal = $event"
      preset="dialog"
      :title="'删除 - ' + deleteRow?.containerName"
    >
      <template #default>
        <n-checkbox
          :checked="deleteWithFile"
          @update:checked="deleteWithFile = $event"
        >删除文件</n-checkbox>
        <div style="color: #888; margin: 8px 0 16px 0">
          删除容器的所有文件，包括配置文件和持久化文件，请谨慎操作！
        </div>
        <div style="color: #d03050; margin-bottom: 8px">
          删除操作无法回滚，请输入
          <b>"{{ deleteRow?.containerName }}"</b>
          删除此应用
        </div>
        <n-input
          :value="deleteConfirmInput"
          @update:value="deleteConfirmInput = $event"
          placeholder="请输入名称"
        />
        <div
          v-if="deleteError"
          style="color: #d03050; margin-top: 8px"
        >{{ deleteError }}</div>
      </template>
      <template #action>
        <n-button @click="showDeleteModal = false">取消</n-button>
        <n-button
          type="error"
          @click="handleDeleteCompose"
          :disabled="deleteConfirmInput !== deleteRow?.containerName"
        >
          确认
        </n-button>
      </template>
    </n-modal>

    <n-modal
      :show="logModalVisible"
      @update:show="logModalVisible = $event"
      preset="card"
      :title="logTitle"
      style="width: 800px"
      :mask-closable="false"
      @after-leave="handleLogModalClose"
    >
      <div class="mb-3 flex items-center justify-between text-xs text-slate-500">
        <span>实时输出，关闭窗口将断开连接</span>
      </div>
      <n-alert
        v-if="repairTipVisible"
        class="mb-3"
        type="warning"
        :title="repairTipTitle"
        :show-icon="true"
      >
        <div class="text-sm leading-6">
          <div v-if="repairTipMessage">{{ repairTipMessage }}</div>
          <div class="mt-2 whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700">{{ repairTipCommands }}</div>
          <div
            v-if="repairTipOutput"
            class="mt-2 whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700"
          >{{ repairTipOutput }}</div>
          <n-space class="mt-3">
            <n-button
              size="small"
              type="primary"
              :loading="repairingCompose"
              @click="handleRepairCompose"
            >一键修复</n-button>
            <n-button
              size="small"
              secondary
              @click="handleRebuild({id: logConfig.id, name: logConfig.name, status: 'Error'})"
              v-if="isInstallFinished && repairTipAction"
            >重新重建</n-button>
            <n-button
              size="small"
              secondary
              @click="copyRepairCommands"
              v-if="repairTipCommands"
            >复制命令</n-button>
          </n-space>
        </div>
      </n-alert>
      <div
        ref="terminalRef"
        class="max-h-[60vh] overflow-auto whitespace-pre-wrap rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-100"
      >
        <div
          v-if="logsData.length === 0"
          class="text-slate-400"
        >暂无日志输出</div>
        <div
          v-for="(line, idx) in logsData"
          :key="idx"
        >
          <span
            v-if="line.includes('ERROR')"
            class="text-red-400"
          >{{ line }}</span>
          <span
            v-else-if="line.includes('INFO')"
            class="text-blue-300"
          >{{ line }}</span>
          <span v-else>{{ line }}</span>
        </div>
      </div>
      <template #action>
        <n-button
          :disabled="!isInstallFinished"
          type="primary"
          @click="handleLogModalClose"
        >关闭</n-button>
      </template>
    </n-modal>
  </div>
</template>
<script setup lang="ts">
import { ref, watch, reactive, nextTick, computed } from "vue"
import { useMessage, useDialog } from "naive-ui"
// @ts-ignore
import { appsInstalledListAPI, appsUninstall, InstalledOp, appsRepairComposeAPI, appsRepairPodmanShortNameAPI, appsRepairPortConflictAPI } from "../../../api/modules/apps"
// @ts-ignore
import { repairSystemdLingerAPI } from "@/api/modules/container"
import type { AppsInstalledSearchParams } from "../../../api/modules/apps"
import { useAuthStore } from "../../../store/auth"

const logConfig = reactive({
	id: 0,
	type: "install",
	name: "",
	tail: true
})
 

const props = defineProps<{
	searchName: string
	page: number
	limit: number
	refreshKey?: number
}>()
const emits = defineEmits(["update:total"])

const message = useMessage()
const dialog = useDialog()
const apps = ref<any[]>([])
const loading = ref(false)
const drawerVisible = ref(false)
const drawerItem = ref<any>(null)
const authStore = useAuthStore()
 
const showDeleteModal = ref(false)
const deleteRow = ref<any>(null)
const deleteWithFile = ref(false)
const deleteConfirmInput = ref("")
const deleteError = ref("")

const logModalVisible = ref(false)
const logsData = ref<string[]>([])
const isInstallFinished = ref(false)
const terminalRef = ref<HTMLElement | null>(null)
let logEventSource: EventSource | null = null
const logTitle = computed(() => (logConfig.name ? `安装日志 - ${logConfig.name}` : "安装日志"))

const repairTipVisible = ref(false)
const repairTipTitle = ref("")
const repairTipMessage = ref("")
const repairTipCommands = ref("")
const repairTipOutput = ref("")
const repairTipAction = ref("")
const currentInstallId = ref<number>(0)
const repairingCompose = ref(false)

const busyStatuses = new Set(["Installing", "Upgrading", "Rebuilding", "Syncing"])
const errorStatuses = new Set(["UpErr", "DownloadErr", "SyncFailed"])

const fetchData = async () => {
	loading.value = true
	try {
		const params: AppsInstalledSearchParams = {
			page: props.page || 1,
			limit: props.limit || 20,
			name: props.searchName.trim() || undefined
		}
		const res = await appsInstalledListAPI(params)
		const data = res.data as any
		if (res.code === 0 && data && Array.isArray(data.items)) {
			apps.value = data.items
			emits("update:total", data.total)
		} else {
			message.error(res.msg || "获取应用列表失败")
		}
	} catch (e) {
	} finally {
		loading.value = false
	}
}

watch([() => props.searchName, () => props.page, () => props.limit, () => props.refreshKey], fetchData, { immediate: true })

function showDrawer(item: any) {
	drawerItem.value = item
	drawerVisible.value = true
}

function statusLabel(status: string) {
	switch (status) {
		case "Running":
			return "已启动"
		case "Stopped":
			return "已停止"
		case "Installing":
			return "安装中"
		case "Upgrading":
			return "升级中"
		case "Rebuilding":
			return "重建中"
		case "Syncing":
			return "同步中"
		case "SyncFailed":
			return "同步失败"
		case "DownloadErr":
			return "下载失败"
		case "UpErr":
			return "启动失败"
		default:
			return status || "-"
	}
}

function statusType(status: string) {
	if (status === "Running") return "success"
	if (busyStatuses.has(status)) return "warning"
	if (errorStatuses.has(status)) return "error"
	return "default"
}

function isBusy(item: any) {
	return busyStatuses.has(item?.status)
}

function disableStart(item: any) {
	return isBusy(item) || item?.status === "Running"
}

function disableStop(item: any) {
	return isBusy(item) || item?.status === "Stopped"
}

function disableRestart(item: any) {
	return isBusy(item)
}

function disableRebuild(item: any) {
	return isBusy(item)
}

function disableUninstall(_item: any) {
	return isBusy(_item)
}

function canCancelInstall(item: any) {
	return item?.status === "Installing"
}

async function cancelInstall(item: any) {
	if (!item?.id) return
	dialog.warning({
		title: "取消安装确认",
		content: `确定要取消安装应用 ${item.name} 吗？这将删除安装记录，并清理安装目录。`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			const loadingMsg = message.loading("取消安装中...", { duration: 0 })
			try {
				const res = await InstalledOp({ installId: item.id, operate: "delete", forceDelete: true } as any)
				if (res.code === 0) {
					message.success("已取消安装")
					fetchData()
				} else {
					message.error(res.msg || "取消安装失败")
				}
			} catch (e) {
				message.error("取消安装异常")
			} finally {
				loadingMsg.destroy()
			}
		}
	})
}

async function handleOperate(item: any, operation: string) {
	if (isBusy(item)) {
		message.warning("当前任务进行中，暂不可操作")
		return
	}
	dialog.warning({
		title: "操作确认",
		content: `确定要${operation === "start" ? "启动" : operation === "stop" ? "停止" : operation === "restart" ? "重启" : operation}应用 ${item.name} 吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			const loadingMsg = message.loading(`${operation}中...`, { duration: 0 })
			try {
				const res = await InstalledOp({ installId: item.id, operate: operation } as any)
				if (res.code === 0) {
					message.success(`${operation} 操作成功`)
					fetchData()
				} else {
					message.error(res.msg || `${operation} 操作失败`)
				}
			} catch (e) {
				message.error(`${operation} 操作异常`)
			} finally {
				loadingMsg.destroy()
			}
		}
	})
}

async function handleRebuild(item: any) {
	if (isBusy(item)) {
		message.warning("当前任务进行中，暂不可重建")
		return
	}
	dialog.warning({
		title: "重建确认",
		content: `确定要重建应用 ${item.name} 吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			const loadingMsg = message.loading("重建中...", { duration: 0 })
			try {
				const res = await InstalledOp({ installId: item.id, operate: "rebuild" } as any)
				if (res.code === 0) {
					message.success("已开始重建")
					fetchData()
					// 调用 openLog 并重置完成状态与提示
					isInstallFinished.value = false
					repairTipVisible.value = false
					repairTipTitle.value = ""
					repairTipMessage.value = ""
					repairTipCommands.value = ""
					repairTipOutput.value = ""
					repairTipAction.value = ""
					openLog(item)
				} else {
					message.error(res.msg || "重建失败")
				}
			} catch (e) {
				message.error("重建异常")
			} finally {
				loadingMsg.destroy()
			}
		}
	})
}

function scrollToBottom() {
	nextTick(() => {
		if (terminalRef.value) {
			terminalRef.value.scrollTop = terminalRef.value.scrollHeight
		}
	})
}

function openLog(item: any) {
	logConfig.name = item?.name || ""
	logConfig.id = item?.id || 0
	currentInstallId.value = item?.id || 0
	logModalVisible.value = true
	logsData.value = []
	const token = authStore.auth || ""
	const apiUrl = "/api"
	if (logEventSource) {
		logEventSource.close()
		logEventSource = null
	}
	if (!logConfig.name || !token) {
		logsData.value.push("[系统提示] 缺少日志名称或登录状态无效")
		return
	}
	logEventSource = new EventSource(`${apiUrl}/apps/install/${encodeURIComponent(logConfig.name)}/logs?token=${encodeURIComponent(token)}`)
	logEventSource.onmessage = (event) => {
		if (event.data === "ping") return
		if (event.data === "EOF" || event.data === '["EOF"]') {
			logEventSource?.close()
			logEventSource = null
			isInstallFinished.value = true
			logsData.value.push("\n====== 日志结束 ======")
			scrollToBottom()
			checkInstallResult(logConfig.name)
			fetchData()
			return
		}
		logsData.value.push(event.data)
				if (event.data.includes("no compose command found")) {
					repairTipVisible.value = true
					repairTipTitle.value = "检测到 Compose 环境缺失"
					repairTipMessage.value = "当前主机未检测到 docker compose / podman compose / podman-compose，导致无法 pull/up。可以先一键修复（需要 root 权限），或复制命令手动执行。"
					repairTipCommands.value = "sudo apt-get update\nsudo apt-get install -y podman-compose"
					repairTipAction.value = "compose"
				} else if (event.data.includes("short-name") && event.data.includes("did not resolve")) {
					repairTipVisible.value = true
					repairTipTitle.value = "检测到 Podman 短名解析失败"
					repairTipMessage.value = "当前容器运行时配置不允许直接拉取简写镜像名。可以先一键修复（自动向 /etc/containers/registries.conf 追加 docker.io 源）。"
					repairTipCommands.value = ""
					repairTipAction.value = "short-name"
				} else if (event.data.includes("cgroup-manager") || event.data.includes("enable-linger")) {
					repairTipVisible.value = true
					repairTipTitle.value = "建议开启用户 Linger (保活) 支持"
					repairTipMessage.value = "检测到当前 Podman 用户会话未开启 Linger，可能导致 cgroup 限制降级或容器异常退出。可以先一键修复开启该支持。"
					repairTipCommands.value = ""
					repairTipAction.value = "linger"
				} else if (event.data.includes("port is already allocated") || event.data.includes("address already in use") || event.data.includes("bind: address already in use")) {
					repairTipVisible.value = true
					repairTipTitle.value = "检测到端口冲突"
					repairTipMessage.value = "当前应用所需的端口已被其他服务占用。可以点击一键修复，系统将自动寻找可用端口并换绑。"
					repairTipCommands.value = ""
					repairTipAction.value = "port-conflict"
				}
				if (logsData.value.length > 2000) {
			logsData.value = logsData.value.slice(-2000)
		}
		scrollToBottom()
	}
	logEventSource.onerror = () => {
		logsData.value.push("\n[系统提示] 与日志服务器的连接已断开或发生错误。")
		isInstallFinished.value = true
		logEventSource?.close()
		logEventSource = null
		scrollToBottom()
	}
}

const checkInstallResult = async (name: string) => {
	try {
		const res = await appsInstalledListAPI({ page: 1, limit: 1, name })
		const data = res.data as any
		const item = data?.items?.[0]
		if (!item) return
		if (item.status === "UpErr" || item.status === "DownloadErr" || item.status === "SyncFailed" || item.status === "Error") {
			if (!repairTipVisible.value && typeof item.message === "string" && item.message.includes("no compose command found")) {
				repairTipVisible.value = true
				repairTipTitle.value = "检测到 Compose 环境缺失"
				repairTipMessage.value = item.message
				repairTipCommands.value = "sudo apt-get update\nsudo apt-get install -y podman-compose"
				repairTipAction.value = "compose"
			} else if (!repairTipVisible.value && typeof item.message === "string" && item.message.includes("short-name") && item.message.includes("did not resolve")) {
				repairTipVisible.value = true
				repairTipTitle.value = "检测到 Podman 短名解析失败"
				repairTipMessage.value = item.message
				repairTipCommands.value = ""
				repairTipAction.value = "short-name"
			} else if (!repairTipVisible.value && typeof item.message === "string" && (item.message.includes("cgroup-manager") || item.message.includes("enable-linger"))) {
				repairTipVisible.value = true
				repairTipTitle.value = "建议开启用户 Linger (保活) 支持"
				repairTipMessage.value = item.message
				repairTipCommands.value = ""
				repairTipAction.value = "linger"
			} else if (!repairTipVisible.value && typeof item.message === "string" && (item.message.includes("port is already allocated") || item.message.includes("address already in use") || item.message.includes("bind: address already in use"))) {
				repairTipVisible.value = true
				repairTipTitle.value = "检测到端口冲突"
				repairTipMessage.value = item.message
				repairTipCommands.value = ""
				repairTipAction.value = "port-conflict"
				currentInstallId.value = item.id || 0
			}
		}
	} catch (e) {
	}
}

const handleRepairCompose = async () => {
	if (repairingCompose.value) return
	repairingCompose.value = true
	repairTipOutput.value = ""
	try {
		let res: any
		if (repairTipAction.value === "short-name") {
			res = await appsRepairPodmanShortNameAPI()
		} else if (repairTipAction.value === "linger") {
			res = await repairSystemdLingerAPI()
		} else if (repairTipAction.value === "port-conflict") {
			if (!currentInstallId.value) {
				throw new Error("无法获取应用安装ID，请重试")
			}
			res = await appsRepairPortConflictAPI(currentInstallId.value)
		} else {
			res = await appsRepairComposeAPI()
		}
		const r = res as any
		if (r.code === 0) {
			repairTipOutput.value = r.data?.output || "已执行修复，请重新发起操作。"
			message.success("修复已执行，请重新发起操作")
		} else {
			message.error(r.msg || "修复失败")
		}
	} catch (e: any) {
		message.error(e?.message || "修复失败")
	} finally {
		repairingCompose.value = false
	}
}

const copyRepairCommands = async () => {
	try {
		const text = repairTipCommands.value || ""
		if (!text) return
		if (navigator?.clipboard?.writeText) {
			await navigator.clipboard.writeText(text)
			message.success("已复制命令")
			return
		}
		message.warning("当前环境不支持一键复制，请手动选择复制")
	} catch (e) {
		message.warning("复制失败，请手动选择复制")
	}
}

function handleLogModalClose() {
	if (logEventSource) {
		logEventSource.close()
		logEventSource = null
	}
}

 

function openDeleteModal(row: any) {
	deleteRow.value = row
	deleteWithFile.value = false
	deleteConfirmInput.value = ""
	deleteError.value = ""
	showDeleteModal.value = true
}

async function handleDeleteCompose() {
	if (!deleteRow.value) return
	if (deleteConfirmInput.value !== deleteRow.value.containerName) {
		deleteError.value = "请输入正确的名称以确认删除"
		return
	}
	deleteError.value = ""
	const loadingMsg = message.loading("正在卸载...", { duration: 0 })
	try {
		const res = await appsUninstall({
			containerName: deleteRow.value.containerName,
			deleteDir: deleteWithFile.value
		})
		if (res.code === 0) {
			message.success("卸载成功")
			showDeleteModal.value = false
			fetchData()
		} else {
			message.error(res.msg || "卸载失败")
		}
	} catch (e) {
		message.error("卸载异常")
	} finally {
		loadingMsg.destroy()
	}
}
</script>
