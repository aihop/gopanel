<template>
  <AppsGrid
    :loading="loading"
    :apps="apps"
    @detail="openDetailDrawer"
    @install="handleInstallApp"
  />

  <AppInstallModal
    :show="showInstallModal"
    :current-app="currentApp"
    :install-loading="installLoading"
    :form-model="formModel"
    :version-options="versionOptions"
    :form-fields="formFields"
    @update:show="showInstallModal = $event"
    @version-change="handleVersionChange"
    @submit="submitInstall"
  />

  <AppDetailDrawer
    :show="showDetailDrawer"
    :detail-app="detailApp"
    :detail-loading="detailLoading"
    @update:show="showDetailDrawer = $event"
    @install="handleInstallFromDetail"
  />

  <AppInstallLogModal
    :show="logModalVisible"
    :logs-data="logsData"
    :is-install-finished="isInstallFinished"
    :repair-tip-visible="repairTipVisible"
    :repair-tip-title="repairTipTitle"
    :repair-tip-message="repairTipMessage"
    :repair-tip-commands="repairTipCommands"
    :repair-tip-output="repairTipOutput"
    :repair-tip-action="repairTipAction"
    :repairing-compose="repairingCompose"
    :retrying-install="retryingInstall"
    @update:show="handleLogModalClose"
    @repair="handleRepairCompose"
    @rebuild="handleRebuild"
    @retry="retryInstall"
    @copy="copyRepairCommands"
  />
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue"
import { appsListAPI, GetApp } from "@/api/modules/apps"
import type { AppsSearchParams } from "@/api/modules/apps"
import { useDialog, useMessage } from "naive-ui"
import { useAuthStore } from "@/store/auth"
import AppDetailDrawer from "./AppDetailDrawer.vue"
import AppsGrid from "./AppsGrid.vue"
import AppInstallLogModal from "./AppInstallLogModal.vue"
import AppInstallModal from "./AppInstallModal.vue"
import { useAppInstallFlow } from "./useAppInstallFlow"

const props = defineProps<{
  searchName: string
  page: number
  limit: number
  refreshKey?: number
}>()

const emits = defineEmits(["update:total"])

const message = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()
const apps = ref<any[]>([])
const loading = ref(false)

const showDetailDrawer = ref(false)
const detailApp = ref<any>(null)
const detailLoading = ref(false)

const fetchData = async () => {
  loading.value = true
  try {
    const params: AppsSearchParams = {
      page: props.page || 1,
      limit: props.limit || 20,
      name: props.searchName.trim() || undefined
    }
    const res = await appsListAPI(params)
    const data = res.data as any
    if (res.code === 0 && data && Array.isArray(data.items)) {
      apps.value = data.items
      emits("update:total", data.total)
    }
  } catch (error) {
  } finally {
    loading.value = false
  }
}

const {
  showInstallModal,
  installLoading,
  currentApp,
  versionOptions,
  formFields,
  formModel,
  logModalVisible,
  logsData,
  isInstallFinished,
  repairTipVisible,
  repairTipTitle,
  repairTipMessage,
  repairTipCommands,
  repairTipOutput,
  repairTipAction,
  repairingCompose,
  retryingInstall,
  handleInstallApp,
  handleVersionChange,
  submitInstall,
  retryInstall,
  handleRepairCompose,
  handleRebuild,
  copyRepairCommands,
  handleLogModalClose,
  cleanup
} = useAppInstallFlow({
  apps,
  authStore,
  message,
  dialog,
  fetchData
})

watch([() => props.searchName, () => props.page, () => props.limit, () => props.refreshKey], fetchData, { immediate: true })

const openDetailDrawer = async (item: any) => {
  detailApp.value = item
  showDetailDrawer.value = true
  detailLoading.value = true
  try {
    const res = await GetApp(item.key)
    if (res.code === 0 && res.data) {
      detailApp.value = res.data
    }
  } finally {
    detailLoading.value = false
  }
}

const handleInstallFromDetail = () => {
  showDetailDrawer.value = false
  if (detailApp.value) {
    void handleInstallApp(detailApp.value)
  }
}

onBeforeUnmount(() => {
  cleanup()
})
</script>
