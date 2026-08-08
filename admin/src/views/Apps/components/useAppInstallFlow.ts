import { reactive, ref } from "vue"
import {
  appsInstalledListAPI,
  appsInstallAPI,
  appsRepairComposeAPI,
  appsRepairPodmanShortNameAPI,
  appsRepairPodmanSubuidAPI,
  appsRepairPortConflictAPI,
  AppsValidateAPI,
  GetApp,
  GetAppDetail,
  InstalledOp
} from "@/api/modules/apps"
import { repairSystemdLingerAPI } from "@/api/modules/container"
import { applyRepairHintFromText, createDefaultFormModel } from "./appInstallHelpers"

interface UseAppInstallFlowOptions {
  apps: { value: any[] }
  authStore: any
  message: any
  dialog: any
  fetchData: () => Promise<void>
}

export const useAppInstallFlow = (options: UseAppInstallFlowOptions) => {
  const { apps, authStore, message, dialog, fetchData } = options

  const showInstallModal = ref(false)
  const installLoading = ref(false)
  const currentApp = ref<any>(null)
  const appDetail = ref<any>(null)
  const versionOptions = ref<any[]>([])
  const formFields = ref<any[]>([])
  const formModel = reactive(createDefaultFormModel())

  const logModalVisible = ref(false)
  const logsData = ref<string[]>([])
  const isInstallFinished = ref(false)
  const repairTipVisible = ref(false)
  const repairTipTitle = ref("")
  const repairTipMessage = ref("")
  const repairTipCommands = ref("")
  const repairTipOutput = ref("")
  const repairTipAction = ref("")
  const currentInstallId = ref<number>(0)
  const currentInstallName = ref("")
  const repairingCompose = ref(false)
  const retryingInstall = ref(false)
  const lastInstallReq = ref<any | null>(null)

  let logEventSource: EventSource | null = null

  const resetFormModel = () => Object.assign(formModel, createDefaultFormModel())
  const resetRepairTip = () => {
    repairTipVisible.value = false; repairTipTitle.value = ""; repairTipMessage.value = ""
    repairTipCommands.value = ""; repairTipOutput.value = ""; repairTipAction.value = ""
  }
  const closeLogStream = () => {
    if (logEventSource) { logEventSource.close(); logEventSource = null }
  }

  const checkInstallResult = async (name: string) => {
    try {
      const res = await appsInstalledListAPI({ page: 1, limit: 1, name })
      const data = res.data as any
      const item = data?.items?.[0]
      if (!item) return
      if (typeof item.name === "string" && item.name) currentInstallName.value = item.name
      if (item.status === "UpErr" || item.status === "DownloadErr" || item.status === "SyncFailed" || item.status === "Error") {
        if (!repairTipVisible.value && typeof item.message === "string") {
          applyRepairHintFromText(item.message, true, {
            repairTipVisible,
            repairTipTitle,
            repairTipMessage,
            repairTipCommands,
            repairTipAction
          })
          if (repairTipAction.value === "port-conflict") {
            currentInstallId.value = item.id || 0
          }
        }
      }
    } catch (error) {
    }
  }

  const startLogStream = (installName: string, finishText: string, onFinish?: () => void) => {
    closeLogStream()
    const apiUrl = "/api"
    const token = authStore.getAuth() || authStore.auth || ""
    logEventSource = new EventSource(`${apiUrl}/apps/install/${installName}/logs?token=${token}`)
    logEventSource.onmessage = (event) => {
      if (event.data === "ping") return
      if (event.data === "EOF" || event.data === '["EOF"]') {
        closeLogStream()
        isInstallFinished.value = true
        logsData.value.push(`\n====== ${finishText} ======`)
        onFinish?.()
        void checkInstallResult(installName)
        void fetchData()
        return
      }
      logsData.value.push(event.data)
      applyRepairHintFromText(event.data, false, {
        repairTipVisible,
        repairTipTitle,
        repairTipMessage,
        repairTipCommands,
        repairTipAction
      })
    }
    logEventSource.onerror = (err) => {
      console.error("SSE Error:", err)
      if (!isInstallFinished.value) {
        logsData.value.push("\n[系统提示] 与日志服务器的连接已断开或发生错误。")
        isInstallFinished.value = true
      }
      closeLogStream()
    }
  }

  const handleVersionChange = async (version: string) => {
    if (!appDetail.value || !appDetail.value.id) return
    try {
      installLoading.value = true
      const res: any = await GetAppDetail(appDetail.value.id, version)
      if (res.code === 0) {
        const detail = res.data
        appDetail.value.appDetail = detail
        formFields.value = detail.params?.formFields || []
        formModel.params = {}
        formFields.value.forEach((field) => {
          formModel.params[field.envKey] = field.default !== undefined ? field.default : ""
        })
        appDetail.value.appDetailId = detail.id
        if (detail.hostMode !== undefined) {
          formModel.hostMode = detail.hostMode
        }
      }
    } catch (error) {
      void 0
    } finally {
      installLoading.value = false
    }
  }

  const handleInstallApp = async (item: any) => {
    currentApp.value = item
    const loadingMsg = message.loading("获取应用信息中...", { duration: 0 })
    try {
      const res = await GetApp(item.key)
      if (res.code === 0 && res.data) {
        appDetail.value = res.data
        versionOptions.value = (res.data.versions || []).map((v: string) => ({ label: v, value: v }))
        resetFormModel()
        formModel.name = `${item.key}-${Math.random().toString(36).substring(2, 6)}`
        formModel.version = versionOptions.value.length > 0 ? versionOptions.value[0].value : ""
        if (formModel.version) {
          await handleVersionChange(formModel.version)
        }
        showInstallModal.value = true
      } else {
        message.error(res.msg || "获取应用详情失败")
      }
    } catch (error) {
      void 0
    } finally {
      loadingMsg.destroy()
    }
  }

  const doSubmitInstall = async (reqData: any) => {
    try {
      const res = await appsInstallAPI(reqData as any)
      if (res.code !== 0) {
        message.error(res.msg || "安装请求失败")
        return
      }

      message.success("应用开始安装")
      showInstallModal.value = false
      currentInstallId.value = (res.data as any)?.installId || 0
      currentInstallName.value = reqData.name
      logModalVisible.value = true
      logsData.value = []
      isInstallFinished.value = false
      resetRepairTip()

      const appIndex = apps.value.findIndex((item) => item.key === currentApp.value?.key)
      if (currentApp.value) {
        currentApp.value.installed = false
        currentApp.value.installing = true
      }
      if (appIndex !== -1) {
        apps.value[appIndex].installed = false
        apps.value[appIndex].installing = true
      }

      startLogStream(reqData.name, "安装流程结束", () => {
        if (currentApp.value) currentApp.value.installing = false
        if (appIndex !== -1) apps.value[appIndex].installing = false
      })
    } catch (error: any) {
      void 0
    } finally {
      installLoading.value = false
    }
  }

  const submitInstall = async () => {
    installLoading.value = true
    try {
      formModel.advanced =
        formModel.allowPort ||
        !!formModel.containerName ||
        formModel.cpuQuota > 0 ||
        formModel.memoryLimit > 0

      const reqData = {
        name: formModel.name,
        appDetailId: appDetail.value.appDetailId,
        params: formModel.params,
        advanced: formModel.advanced,
        allowPort: formModel.allowPort,
        containerName: formModel.containerName,
        cpuQuota: formModel.cpuQuota,
        memoryLimit: formModel.memoryLimit,
        memoryUnit: formModel.memoryUnit,
        pullImage: true,
        editCompose: false
      }
      lastInstallReq.value = JSON.parse(JSON.stringify(reqData))
      currentInstallName.value = reqData.name

      try {
        const validateRes = await AppsValidateAPI()
        if (validateRes.code === 0 && validateRes.data?.isWarning) {
          installLoading.value = false
          dialog.warning({
            title: "资源预警",
            content: validateRes.data.message,
            positiveText: "强制继续",
            negativeText: "取消",
            onPositiveClick: async () => {
              installLoading.value = true
              await doSubmitInstall(reqData)
            }
          })
          return
        }
      } catch (error) {
        console.error("预检失败", error)
      }

      await doSubmitInstall(reqData)
    } catch (error: any) {
      installLoading.value = false
    }
  }

  const retryInstall = async () => {
    if (retryingInstall.value) return
    if (!lastInstallReq.value) {
      message.warning("未找到上一次安装参数，请重新打开安装表单提交一次")
      return
    }
    retryingInstall.value = true
    try {
      await doSubmitInstall(JSON.parse(JSON.stringify(lastInstallReq.value)))
    } catch {
      return
    } finally {
      retryingInstall.value = false
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
      } else if (repairTipAction.value === "subuid") {
        res = await appsRepairPodmanSubuidAPI()
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

      if (res.code === 0) {
        repairTipOutput.value = res.data?.output || "已执行修复，请重新发起安装。"
        message.success("修复已执行，请重新发起安装")
      } else {
        message.error(res.msg || "修复失败")
      }
    } catch (error: any) {
      void 0
    } finally {
      repairingCompose.value = false
    }
  }

  const handleRebuild = async () => {
    if (!currentInstallId.value) return
    const loadingMsg = message.loading("重建中...", { duration: 0 })
    try {
      const res = await InstalledOp({ installId: currentInstallId.value, operate: "rebuild" } as any)
      if (res.code !== 0) {
        message.error(res.msg || "重建失败")
        return
      }

      message.success("已开始重建")
      await fetchData()
      isInstallFinished.value = false
      resetRepairTip()
      logsData.value = []

      let installName = currentInstallName.value || lastInstallReq.value?.name || ""
      const resInfo = await appsInstalledListAPI({ page: 1, limit: 50, name: "" })
      if (resInfo.data && (resInfo.data as any).items) {
        const item = (resInfo.data as any).items.find((entry: any) => entry.id === currentInstallId.value)
        if (item?.name) installName = item.name
      }
      if (!installName) {
        installName = currentApp.value?.key || ""
      }
      currentInstallName.value = installName
      startLogStream(installName, "重建流程结束")
    } catch (error: any) {
      void 0
    } finally {
      loadingMsg.destroy()
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
    } catch (error) {
      message.warning("复制失败，请手动选择复制")
    }
  }

  const handleLogModalClose = (visible: boolean) => {
    logModalVisible.value = visible
    if (!visible) {
      closeLogStream()
    }
  }

  const cleanup = () => closeLogStream()

  return {
    showInstallModal, installLoading, currentApp, appDetail, versionOptions, formFields, formModel,
    logModalVisible, logsData, isInstallFinished, repairTipVisible, repairTipTitle, repairTipMessage,
    repairTipCommands, repairTipOutput, repairTipAction, repairingCompose, retryingInstall,
    handleInstallApp, handleVersionChange, submitInstall, retryInstall, handleRepairCompose,
    handleRebuild, copyRepairCommands, handleLogModalClose, cleanup
  }
}
