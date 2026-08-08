import { ref } from "vue"
import { AgentEnsureAPI, AgentStatusAPI, AgentUpdateAPI, AgentUpdateCheckAPI } from "@/api/modules/agent"
import { useAuthStore } from "@/store/auth"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { controlPlaneMessages } from "@/i18n/locales/controlPlane"

export const useDaemonAgentStatus = (
  opDialogRef: { value: { acceptParams: (params: any) => void } | null },
  onFinished: () => void
) => {
  const authStore = useAuthStore()
  const { t } = useI18n({ messages: controlPlaneMessages })
  const message = useMessage()
  const ensuringAgent = ref(false)
  const updatingAgent = ref(false)
  const agentStatus = ref<{ online: boolean; error?: string; version?: string }>({ online: true })
  // gp-agent 版本 / 是否有更新
  const agentUpdate = ref<{ needUpdate: boolean; currentVersion?: string; latestVersion?: string }>({ needUpdate: false })

  const fetchAgentStatus = async () => {
    try {
      const res = await AgentStatusAPI()
      agentStatus.value = {
        online: !!res?.data?.online,
        error: res?.data?.error,
        version: res?.data?.agent?.version
      }
    } catch (e: any) {
      agentStatus.value = { online: false, error: e?.message || t("controlPlane.loadFailed") }
    }
  }

  // 检查 gp-agent 是否有新版本（会请求升级服务器，按需调用，不随状态轮询）
  const checkAgentUpdate = async () => {
    try {
      const res = await AgentUpdateCheckAPI()
      const d = res?.data || {}
      agentUpdate.value = {
        needUpdate: !!d.needUpdate,
        currentVersion: d.currentVersion,
        latestVersion: d.latestVersion
      }
      // 状态里没拿到版本时用这里的当前版本补上
      if (!agentStatus.value.version && d.currentVersion) {
        agentStatus.value = { ...agentStatus.value, version: d.currentVersion }
      }
    } catch {
      agentUpdate.value = { needUpdate: false }
    }
  }

  const ensureAgent = async () => {
    if (ensuringAgent.value) return
    ensuringAgent.value = true
    try {
      const res = await AgentEnsureAPI()
      const log = res?.data?.log
      const token = authStore.getAuth() || authStore.auth || ""
      if (log) {
        opDialogRef.value?.acceptParams({
          title: t("controlPlane.initializeTitle"),
          sseUrl: `/api/agent/ensure/logs?log=${encodeURIComponent(log)}&token=${encodeURIComponent(token)}`
        })
      } else {
        // 没拿到日志名说明后端没真正起任务，别把按钮一直锁着
        ensuringAgent.value = false
        message.error(t("controlPlane.operationFailed"))
      }
    } catch (error: any) {
      ensuringAgent.value = false
    }
    // 注意：成功发起后不在这里解锁 —— 任务还在后台跑，
    // 解锁交给日志弹窗结束时的 handleEnsureFinished，避免连点触发两次安装
  }

  // 手动更新 gp-agent。以前是进入面板时自动跑，现在必须用户点按钮才会执行，
  // 过程走和「一键初始化」相同的 SSE 日志弹窗。
  const updateAgent = async () => {
    if (updatingAgent.value) return
    updatingAgent.value = true
    try {
      const res = await AgentUpdateAPI()
      const log = res?.data?.log
      const token = authStore.getAuth() || authStore.auth || ""
      if (log) {
        opDialogRef.value?.acceptParams({
          title: t("controlPlane.updateTitle"),
          sseUrl: `/api/agent/ensure/logs?log=${encodeURIComponent(log)}&token=${encodeURIComponent(token)}`
        })
      } else {
        updatingAgent.value = false
        message.error(t("controlPlane.operationFailed"))
      }
    } catch (error: any) {
      updatingAgent.value = false
    }
    // 同 ensureAgent：解锁交给日志弹窗结束
  }

  const handleEnsureFinished = () => {
    // 任务结束（日志弹窗关闭/收到 EOF）才解锁按钮，防止连点触发两次安装/更新
    ensuringAgent.value = false
    updatingAgent.value = false
    fetchAgentStatus()
    // 更新完版本号会变，这里顺手把「有无新版」重新算一次
    checkAgentUpdate()
    onFinished()
  }

  return {
    ensuringAgent,
    updatingAgent,
    agentStatus,
    agentUpdate,
    fetchAgentStatus,
    checkAgentUpdate,
    ensureAgent,
    updateAgent,
    handleEnsureFinished
  }
}
