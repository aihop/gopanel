import { ref } from "vue"
import { AgentEnsureAPI, AgentStatusAPI, AgentUpdateCheckAPI } from "@/api/modules/agent"
import { useAuthStore } from "@/store/auth"

export const useDaemonAgentStatus = (
  opDialogRef: { value: { acceptParams: (params: any) => void } | null },
  onFinished: () => void
) => {
  const authStore = useAuthStore()
  const ensuringAgent = ref(false)
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
      agentStatus.value = { online: false, error: e?.message || "获取 Agent 状态失败" }
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
          title: "初始化 Agent",
          sseUrl: `/api/agent/ensure/logs?log=${encodeURIComponent(log)}&token=${encodeURIComponent(token)}`
        })
      }
    } finally {
      ensuringAgent.value = false
    }
  }

  const handleEnsureFinished = () => {
    fetchAgentStatus()
    onFinished()
  }

  return {
    ensuringAgent,
    agentStatus,
    agentUpdate,
    fetchAgentStatus,
    checkAgentUpdate,
    ensureAgent,
    handleEnsureFinished
  }
}
