import { computed, nextTick, reactive, ref } from "vue"
import {
  appsInstalledListAPI,
  appsRepairComposeAPI,
  appsRepairPodmanShortNameAPI,
  appsRepairPodmanSubuidAPI,
  appsRepairPortConflictAPI
} from "@/api/modules/apps"
import { repairSystemdLingerAPI } from "@/api/modules/container"
import { applyRepairHintFromText } from "./appInstallHelpers"

interface UseInstalledAppLogOptions {
  authStore: { auth?: string }
  message: any
  fetchData: () => Promise<void>
}

export const useInstalledAppLog = (options: UseInstalledAppLogOptions) => {
  const logConfig = reactive({
    id: 0,
    type: "runtime" as "install" | "runtime",
    name: "",
    tail: true
  })

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
  const repairingCompose = ref(false)
  let logEventSource: EventSource | null = null

  const logTitle = computed(() => {
    const prefix = logConfig.type === "install" ? "安装日志" : "运行日志"
    return logConfig.name ? `${prefix} - ${logConfig.name}` : prefix
  })

  const canCloseLogModal = computed(() => logConfig.type !== "install" || isInstallFinished.value)

  const resetLogTips = () => {
    repairTipVisible.value = false
    repairTipTitle.value = ""
    repairTipMessage.value = ""
    repairTipCommands.value = ""
    repairTipOutput.value = ""
    repairTipAction.value = ""
  }

  const scrollToBottom = () => {
    nextTick(() => {
      const elements = document.querySelectorAll("[data-installed-log-terminal]")
      const element = elements.item(elements.length - 1) as HTMLElement | null
      if (element) {
        element.scrollTop = element.scrollHeight
      }
    })
  }

  const closeLogStream = () => {
    if (logEventSource) {
      logEventSource.close()
      logEventSource = null
    }
  }

  const checkInstallResult = async (name: string) => {
    try {
      const res = await appsInstalledListAPI({ page: 1, limit: 1, name })
      const data = res.data as any
      const item = data?.items?.[0]
      if (!item) return
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

  const openLog = (item: any, type: "install" | "runtime" = "runtime") => {
    logConfig.name = item?.name || ""
    logConfig.id = item?.id || 0
    logConfig.type = type
    currentInstallId.value = item?.id || 0
    logModalVisible.value = true
    logsData.value = []
    isInstallFinished.value = type !== "install"
    resetLogTips()

    const token = options.authStore.auth || ""
    const apiUrl = "/api"
    closeLogStream()
    if (!logConfig.name || !token) {
      logsData.value.push("[系统提示] 缺少日志名称或登录状态无效")
      return
    }

    const endpoint = logConfig.type === "install"
      ? `/apps/install/${encodeURIComponent(logConfig.name)}/logs`
      : `/apps/installed/${encodeURIComponent(logConfig.name)}/runtime/logs`
    logEventSource = new EventSource(`${apiUrl}${endpoint}?token=${encodeURIComponent(token)}`)
    logEventSource.onmessage = event => {
      if (event.data === "ping") return
      if (event.data === "EOF" || event.data === '["EOF"]') {
        closeLogStream()
        isInstallFinished.value = logConfig.type === "install"
        logsData.value.push("\n====== 日志结束 ======")
        scrollToBottom()
        if (logConfig.type === "install") {
          void checkInstallResult(logConfig.name)
          void options.fetchData()
        }
        return
      }
      logsData.value.push(event.data)
      if (logConfig.type === "install") {
        applyRepairHintFromText(event.data, false, {
          repairTipVisible,
          repairTipTitle,
          repairTipMessage,
          repairTipCommands,
          repairTipAction
        })
      }
      if (logsData.value.length > 2000) {
        logsData.value = logsData.value.slice(-2000)
      }
      scrollToBottom()
    }
    logEventSource.onerror = () => {
      logsData.value.push(
        logConfig.type === "install"
          ? "\n[系统提示] 与安装日志服务器的连接已断开或发生错误。"
          : "\n[系统提示] 与运行日志服务器的连接已断开或发生错误。"
      )
      isInstallFinished.value = logConfig.type === "install"
      closeLogStream()
      scrollToBottom()
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
        repairTipOutput.value = res.data?.output || "已执行修复，请重新发起操作。"
        options.message.success("修复已执行，请重新发起操作")
      } else {
        options.message.error(res.msg || "修复失败")
      }
    } catch (error: any) {
      // 错误提示由请求拦截器统一处理
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
        options.message.success("已复制命令")
        return
      }
      options.message.warning("当前环境不支持一键复制，请手动选择复制")
    } catch (error) {
      options.message.warning("复制失败，请手动选择复制")
    }
  }

  const handleLogModalClose = () => {
    closeLogStream()
    logModalVisible.value = false
  }

  return {
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
  }
}
