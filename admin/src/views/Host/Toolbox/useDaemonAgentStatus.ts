import { ref } from "vue"
import { AgentEnsureAPI, AgentStatusAPI } from "@/api/modules/agent"
import { useAuthStore } from "@/store/auth"

export const useDaemonAgentStatus = (
  opDialogRef: { value: { acceptParams: (params: any) => void } | null },
  onFinished: () => void
) => {
  const authStore = useAuthStore()
  const ensuringAgent = ref(false)
  const agentStatus = ref<{ online: boolean; error?: string }>({ online: true })

  const fetchAgentStatus = async () => {
    try {
      const res = await AgentStatusAPI()
      agentStatus.value = {
        online: !!res?.data?.online,
        error: res?.data?.error
      }
    } catch (e: any) {
      agentStatus.value = { online: false, error: e?.message || "获取 Agent 状态失败" }
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
    fetchAgentStatus,
    ensureAgent,
    handleEnsureFinished
  }
}
