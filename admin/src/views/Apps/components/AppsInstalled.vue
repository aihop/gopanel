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
      @after-leave="handleLogModalClose"
    >
      <div class="mb-3 text-xs text-slate-500">实时输出，关闭窗口将断开连接</div>
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
        >{{ line }}</div>
      </div>
    </n-modal>
  </div>
</template>
<script setup lang="ts">
import { ref, watch, reactive, nextTick, computed } from "vue"
import { useMessage, useDialog } from "naive-ui"
import { appsInstalledListAPI, appsUninstall, InstalledOp } from "../../../api/modules/apps"
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
const terminalRef = ref<HTMLElement | null>(null)
let logEventSource: EventSource | null = null
const logTitle = computed(() => (logConfig.name ? `安装日志 - ${logConfig.name}` : "安装日志"))

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

watch([() => props.searchName, () => props.page, () => props.limit], fetchData, { immediate: true })

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
			logsData.value.push("\n====== 日志结束 ======")
			scrollToBottom()
			return
		}
		logsData.value.push(event.data)
		if (logsData.value.length > 2000) {
			logsData.value = logsData.value.slice(-2000)
		}
		scrollToBottom()
	}
	logEventSource.onerror = () => {
		logsData.value.push("\n[系统提示] 与日志服务器的连接已断开或发生错误。")
		logEventSource?.close()
		logEventSource = null
		scrollToBottom()
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
