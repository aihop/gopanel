<template>
  <div class="w-full">
    <InstalledAppsGrid
      :apps="apps"
      :loading="loading"
      @detail="showDrawer"
      @cancel="cancelInstall"
      @log="openLog"
      @operate="handleOperate"
      @rebuild="handleRebuild"
      @delete="openDeleteModal"
    />

    <InstalledAppDetailDrawer
      v-model:show="drawerVisible"
      :item="drawerItem"
    />

    <InstalledAppDeleteModal
      v-model:show="showDeleteModal"
      :row="deleteRow"
      :delete-with-file="deleteWithFile"
      :delete-confirm-input="deleteConfirmInput"
      :delete-error="deleteError"
      @update:delete-with-file="deleteWithFile = $event"
      @update:delete-confirm-input="deleteConfirmInput = $event"
      @confirm="handleDeleteCompose"
    />

    <InstalledAppLogModal
      :show="logModalVisible"
      :title="logTitle"
      :log-type="logConfig.type"
      :logs-data="logsData"
      :is-install-finished="isInstallFinished"
      :can-close="canCloseLogModal"
      :repair-tip-visible="repairTipVisible"
      :repair-tip-title="repairTipTitle"
      :repair-tip-message="repairTipMessage"
      :repair-tip-commands="repairTipCommands"
      :repair-tip-output="repairTipOutput"
      :repair-tip-action="repairTipAction"
      :repairing-compose="repairingCompose"
      @update:show="logModalVisible = $event"
      @repair="handleRepairCompose"
      @rebuild="handleLogRebuild"
      @copy="copyRepairCommands"
      @close="handleLogModalClose"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { appsInstalledListAPI, appsUninstall, InstalledOp } from "@/api/modules/apps"
import type { AppsInstalledSearchParams } from "@/api/modules/apps"
import { useAuthStore } from "@/store/auth"
import InstalledAppDeleteModal from "./InstalledAppDeleteModal.vue"
import InstalledAppDetailDrawer from "./InstalledAppDetailDrawer.vue"
import InstalledAppLogModal from "./InstalledAppLogModal.vue"
import InstalledAppsGrid from "./InstalledAppsGrid.vue"
import { isBusy } from "./installedAppHelpers"
import { useInstalledAppLog } from "./useInstalledAppLog"

const props = defineProps<{
  searchName: string
  page: number
  limit: number
  refreshKey?: number
}>()

const emits = defineEmits<{
  (e: "update:total", total: number): void
}>()

const message = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()

const apps = ref<any[]>([])
const loading = ref(false)
const drawerVisible = ref(false)
const drawerItem = ref<any>(null)
const showDeleteModal = ref(false)
const deleteRow = ref<any>(null)
const deleteWithFile = ref(false)
const deleteConfirmInput = ref("")
const deleteError = ref("")

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
  } finally {
    loading.value = false
  }
}

watch([() => props.searchName, () => props.page, () => props.limit, () => props.refreshKey], fetchData, { immediate: true })

const {
  logConfig,
  logModalVisible,
  logsData,
  logTitle,
  isInstallFinished,
  canCloseLogModal,
  repairTipVisible,
  repairTipTitle,
  repairTipMessage,
  repairTipCommands,
  repairTipOutput,
  repairTipAction,
  repairingCompose,
  openLog,
  handleRepairCompose,
  copyRepairCommands,
  handleLogModalClose
} = useInstalledAppLog({
  authStore,
  message,
  fetchData
})

function showDrawer(item: any) {
  drawerItem.value = item
  drawerVisible.value = true
}

function openDeleteModal(row: any) {
  deleteRow.value = row
  deleteWithFile.value = false
  deleteConfirmInput.value = ""
  deleteError.value = ""
  showDeleteModal.value = true
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
          await fetchData()
        } else {
          message.error(res.msg || "取消安装失败")
        }
      } catch (error) {
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
  const actionText = operation === "start" ? "启动" : operation === "stop" ? "停止" : operation === "restart" ? "重启" : operation
  dialog.warning({
    title: "操作确认",
    content: `确定要${actionText}应用 ${item.name} 吗？`,
    positiveText: "确定",
    negativeText: "取消",
    onPositiveClick: async () => {
      const loadingMsg = message.loading(`${actionText}中...`, { duration: 0 })
      try {
        const res = await InstalledOp({ installId: item.id, operate: operation } as any)
        if (res.code === 0) {
          message.success(`${actionText}操作成功`)
          await fetchData()
        } else {
          message.error(res.msg || `${actionText}操作失败`)
        }
      } catch (error) {
        message.error(`${actionText}操作异常`)
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
          await fetchData()
          openLog(item, "install")
        } else {
          message.error(res.msg || "重建失败")
        }
      } catch (error) {
        message.error("重建异常")
      } finally {
        loadingMsg.destroy()
      }
    }
  })
}

function handleLogRebuild() {
  handleRebuild({ id: logConfig.id, name: logConfig.name, status: "Error" })
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
      await fetchData()
    } else {
      message.error(res.msg || "卸载失败")
    }
  } catch (error) {
    message.error("卸载异常")
  } finally {
    loadingMsg.destroy()
  }
}
</script>
