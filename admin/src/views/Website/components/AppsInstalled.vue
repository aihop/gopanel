<template>
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
          :type="item.status === '已启动' ? 'success' : 'error'"
        >
          {{ item.status }}
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
            <!-- <n-button size="small" round>参数</n-button> -->
          </n-button-group>
        </div>
      </template>
    </n-card>
  </div>

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
import { ref, watch } from "vue"
import { appsInstalledSearch, appsUninstall } from "@/api/modules/apps"
import type { AppsInstalledSearchParams } from "@/api/modules/apps"
import { useMessage, useDialog } from "naive-ui"
import { containerOperator } from "@/api/modules/container"

const showLog = ref(false)
const isReading = ref(false)
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
