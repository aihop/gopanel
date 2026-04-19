<template>
  <n-spin :show="loading">
    <div
      v-if="apps.length"
      class="apps-grid"
    >
      <div
        v-for="item in apps"
        :key="item.id"
        class="app-card"
      >
        <div class="app-card__glow"></div>
        <div class="app-card__body">
          <div class="app-card__header">
            <div class="app-card__identity">
              <div class="app-card__icon">
                <img
                  v-if="item.app?.icon"
                  :src="item.app.icon"
                  alt="icon"
                  class="h-10 w-10 object-contain"
                />
                <span
                  v-else
                  class="text-base font-bold text-emerald-600"
                >
                  {{ item.name?.slice(0, 1)?.toUpperCase() }}
                </span>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span
                    class="app-card__name truncate"
                    @click="showDrawer(item)"
                  >
                    {{ item.name }}
                  </span>
                  <n-tag
                    v-if="item.status"
                    :type="item.status === '已启动' ? 'success' : 'error'"
                    size="small"
                    round
                  >
                    {{ item.status }}
                  </n-tag>
                </div>
                <div class="mt-1 text-sm text-slate-500">{{ item.app?.name || "已安装应用" }}</div>
              </div>
            </div>
          </div>

          <div class="app-card__meta">
            <div class="app-chip">
              <span class="app-chip__label">版本</span>
              <span class="app-chip__value">{{ item.version || "-" }}</span>
            </div>
            <div
              v-if="item.httpPort || item.httpsPort"
              class="app-chip"
            >
              <span class="app-chip__label">服务端口</span>
              <span class="app-chip__value">
                {{ item.httpPort || "-" }}<template v-if="item.httpsPort"> / {{ item.httpsPort }}</template>
              </span>
            </div>
          </div>

          <div class="app-card__info">
            <p><span class="app-info__label">容器名</span>{{ item.containerName || "-" }}</p>
            <p><span class="app-info__label">安装时间</span>{{ item.createdAt || "-" }}</p>
            <p v-if="item.description"><span class="app-info__label">描述</span>{{ item.description }}</p>
          </div>

          <div class="app-card__footer">
            <n-button
              size="small"
              round
              @click="handleOperate(item, 'start')"
            >启动</n-button>
            <n-button
              size="small"
              @click="handleOperate(item, 'stop')"
            >停止</n-button>
            <n-button
              size="small"
              @click="handleOperate(item, 'restart')"
            >重启</n-button>
            <n-button
              size="small"
              round
              @click="openDeleteModal(item)"
            >卸载</n-button>
          </div>
        </div>
      </div>
    </div>
    <div
      v-else
      class="app-empty"
    >
      暂无已安装应用
    </div>
  </n-spin>

  <n-drawer
    v-model:show="drawerVisible"
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
    v-model:show="showDeleteModal"
    preset="dialog"
    :title="'删除 - ' + deleteRow?.containerName"
  >
    <template #default>
      <n-checkbox v-model:checked="deleteWithFile">删除文件</n-checkbox>
      <div style="color: #888; margin: 8px 0 16px 0">
        删除容器的所有文件，包括配置文件和持久化文件，请谨慎操作！
      </div>
      <div style="color: #d03050; margin-bottom: 8px">
        删除操作无法回滚，请输入
        <b>"{{ deleteRow?.containerName }}"</b>
        删除此应用
      </div>
      <n-input
        v-model:value="deleteConfirmInput"
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
</template>

<script setup lang="ts">
import { ref, watch, onMounted, h } from "vue"
import { appsInstalledSearch, appsUninstall } from "@/api/modules/apps"
import type { AppsInstalledSearchParams } from "@/api/modules/apps"
import { useMessage, useDialog } from "naive-ui"
import { containerOperator } from "@/api/modules/container"
 
const isReading = ref(false)
 

const handleLogReading = (reading: boolean) => {
	isReading.value = reading
	// 更新完成之后禁用更新按钮
	if (reading == false) {
		updateDrawerInfo.value.canUpdate = false
	}
}

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

interface UpdateDrawerInfo {
	currentVersionInfo: null | {
		appName: string
		appVersion: string
		appWebsite: string
		appVersionCode?: number
	}
	latestVersionInfo: null | {
		appVersion: string
		appVersionCode?: number
	}
	logName: string
	canUpdate: boolean
	item: any
}

const updateDrawerInfo = ref<UpdateDrawerInfo>({
	currentVersionInfo: null,
	latestVersionInfo: null,
	logName: "",
	canUpdate: false,
	item: null
})
const updateLoading = ref(false)

const showDeleteModal = ref(false)
const deleteRow = ref<any>(null)
const deleteWithFile = ref(false)
const deleteConfirmInput = ref("")
const deleteError = ref("")

const fetchData = async () => {
	loading.value = true
	try {
		const params: AppsInstalledSearchParams = {
			page: props.page,
			limit: props.limit,
			name: props.searchName.trim() || undefined
		}
		const res = await appsInstalledSearch(params)
		const data = res.data as any
		if (res.code === 0 && data && Array.isArray(data.items)) {
			apps.value = data.items
			emits("update:total", data.total)
		} else {
			message.error(res.msg || "获取应用列表失败")
		}
	} catch (e) {
		message.error("获取应用列表失败")
	} finally {
		loading.value = false
	}
}

watch([() => props.searchName, () => props.page, () => props.limit], fetchData, { immediate: true })

function showDrawer(item: any) {
	drawerItem.value = item
	drawerVisible.value = true
}

async function handleOperate(item: any, operation: string) {
	dialog.warning({
		title: "操作确认",
		content: `确定要${operation === "start" ? "启动" : operation === "stop" ? "停止" : operation === "restart" ? "重启" : operation}容器${item.containerName}吗？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			const loadingMsg = message.loading(`${operation}中...`, { duration: 0 })
			try {
				const res = await containerOperator({ names: [item.containerName], operation })
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
.app-card {
	position: relative;
	overflow: hidden;
	border-radius: 24px;
	border: 1px solid rgba(226, 232, 240, 0.88);
	background:
		radial-gradient(circle at top right, rgba(16, 185, 129, 0.1), transparent 28%),
		linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.92));
	box-shadow: 0 14px 36px rgba(15, 23, 42, 0.06);
	transition: transform 0.26s ease, box-shadow 0.26s ease, border-color 0.26s ease;
}

.app-card:hover {
	transform: translateY(-4px);
	border-color: rgba(16, 185, 129, 0.2);
	box-shadow: 0 22px 44px rgba(15, 23, 42, 0.1);
}

.apps-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
	gap: 18px;
}

.app-card__glow {
	position: absolute;
	top: -40px;
	right: -32px;
	width: 120px;
	height: 120px;
	border-radius: 9999px;
	background: rgba(16, 185, 129, 0.12);
	filter: blur(20px);
	pointer-events: none;
}

.app-card__body {
	position: relative;
	z-index: 1;
	display: flex;
	flex-direction: column;
	gap: 18px;
	padding: 22px;
	height: 100%;
}

.app-card__header {
	display: flex;
	justify-content: space-between;
	gap: 14px;
}

.app-card__identity {
	display: flex;
	align-items: flex-start;
	gap: 12px;
	min-width: 0;
	flex: 1;
}

.app-card__icon {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 54px;
	height: 54px;
	flex-shrink: 0;
	border-radius: 18px;
	border: 1px solid rgba(209, 250, 229, 0.95);
	background: linear-gradient(135deg, rgba(236, 253, 245, 0.98), rgba(255, 255, 255, 0.72));
	box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.75);
}

.app-card__name {
	cursor: pointer;
	color: #0f766e;
	font-weight: 700;
}

.app-card__name:hover {
	text-decoration: underline;
}

.app-card__meta {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 10px;
}

.app-chip {
	padding: 10px 12px;
	border-radius: 16px;
	background: rgba(248, 250, 252, 0.95);
	border: 1px solid rgba(226, 232, 240, 0.9);
}

.app-chip__label {
	display: block;
	font-size: 0.72rem;
	color: rgb(148 163 184);
	margin-bottom: 4px;
}

.app-chip__value {
	display: block;
	font-size: 0.88rem;
	font-weight: 600;
	color: rgb(30 41 59);
	word-break: break-word;
}

.app-card__info {
	display: flex;
	flex-direction: column;
	gap: 8px;
	font-size: 0.92rem;
	color: rgb(71 85 105);
	line-height: 1.65;
	min-height: 96px;
}

.app-card__info p {
	margin: 0;
}

.app-info__label {
	display: inline-block;
	min-width: 64px;
	margin-right: 8px;
	color: rgb(148 163 184);
}

.app-card__footer {
	display: flex;
	flex-wrap: wrap;
	gap: 10px;
	padding-top: 16px;
	border-top: 1px solid rgba(226, 232, 240, 0.9);
}

.app-empty {
	padding: 64px 20px;
	text-align: center;
	font-size: 0.95rem;
	color: rgb(148 163 184);
}

@media (max-width: 640px) {
	.apps-grid {
		grid-template-columns: 1fr;
	}

	.app-card__meta {
		grid-template-columns: 1fr;
	}
}
</style>
