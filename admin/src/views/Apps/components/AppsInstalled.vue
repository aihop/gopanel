<template>
  <div>
    <div class="apps-card-list">
      <n-card
        v-for="item in apps"
        :key="item.id"
        class="app-card"
      >
        <template #header>
          <img
            v-if="item.app?.icon"
            :src="item.app.icon"
            alt="icon"
            class="mr-2 h-8 w-8 align-middle"
          />
          <span
            class="item-name cursor-pointer text-primary hover:underline"
            @click="showDrawer(item)"
            style="margin-right: 8px"
          >
            {{ item.name }}
          </span>
          <n-tag
            v-if="item.status"
            :type="statusType(item.status)"
          >
            {{ statusLabel(item.status) }}
          </n-tag>
        </template>

        <template #header-extra>
          <n-button-group>
            <!-- <n-button secondary size="small" disabled>导入备份</n-button>
					<n-button secondary size="small" disabled>备份</n-button> -->
          </n-button-group>
        </template>

        <div class="flex flex-wrap gap-4">
          <n-tag size="small">版本：{{ item.version }}</n-tag>
          <n-tag
            v-if="item.httpPort"
            size="small"
          >服务端口：{{ item.httpPort }}</n-tag>
          <n-tag
            v-if="item.httpsPort"
            size="small"
          >服务端口(https)：{{ item.httpsPort }}</n-tag>
        </div>

        <div style="margin-top: 15px">
          <p>容器名：{{ item.containerName }}</p>
          <p>安装日期：{{ item.createdAt }}</p>
          <p v-if="item.description">描述：{{ item.description }}</p>
        </div>

        <template #footer>
          <div class="item-footer">
            <n-button-group>
              <n-button
                size="small"
                round
                @click="handleOperate(item, 'start')"
                :disabled="disableStart(item)"
              >启动</n-button>
              <n-button
                size="small"
                @click="handleOperate(item, 'stop')"
                :disabled="disableStop(item)"
              >停止</n-button>
              <n-button
                size="small"
                @click="handleOperate(item, 'restart')"
                :disabled="disableRestart(item)"
              >重启</n-button>
              <n-button
                size="small"
                @click="handleRebuild(item)"
                :disabled="disableRebuild(item)"
              >重建</n-button>
              <n-button
                size="small"
                @click="openLog(item)"
              >日志</n-button>
              <n-button
                size="small"
                round
                @click="openDeleteModal(item)"
                :disabled="disableUninstall(item)"
              >卸载</n-button>
              <!-- <n-button size="small" round>参数</n-button> -->
            </n-button-group>
          </div>
        </template>
      </n-card>
    </div>

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
      style="width: 80vw; max-width: 900px"
      @after-leave="handleLogModalClose"
    >
      <div
        ref="terminalRef"
        style="max-height: 60vh; overflow: auto; white-space: pre-wrap; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;"
      >
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
	return false
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
	const token = (authStore as any).getAuth?.() || authStore.auth || ""
	const apiUrl = "/api"
	if (logEventSource) {
		logEventSource.close()
		logEventSource = null
	}
	if (!logConfig.name || !token) {
		logsData.value.push("[系统提示] 缺少日志名称或登录状态无效")
		return
	}
	logEventSource = new EventSource(`${apiUrl}/apps/install/${encodeURIComponent(logConfig.name)}/logs?token=${encodeURIComponent(token)}&tail=true`)
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

<style scoped>
.apps-card-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(500px, 1fr));
	gap: 16px;
}
.app-card {
	min-width: 450px;
	margin-bottom: 16px;
}
.item-name {
	cursor: pointer;
	color: #18a058;
}
.item-footer {
	border-top: 1px solid rgb(240 240 240);
	padding-top: 15px;
	display: flex;
	justify-content: space-between;
	flex-direction: row;
}
</style>
