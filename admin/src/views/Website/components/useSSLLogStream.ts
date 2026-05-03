import { ref } from "vue"
import { useAuthStore } from "@/store/auth"

interface UseSSLLogStreamOptions {
  onFinished: () => Promise<void> | void
}

export const useSSLLogStream = (options: UseSSLLogStreamOptions) => {
  const logsData = ref<string[]>([])
  const logModalVisible = ref(false)
  let logEventSource: EventSource | null = null

  function openLogModal(id: number) {
    logsData.value = []
    logModalVisible.value = true
    logEventSource?.close()

    const authStore = useAuthStore()
    const apiUrl = (window as { __VITE_API_URL__?: string }).__VITE_API_URL__ || "/api"
    logEventSource = new EventSource(`${apiUrl}/ssl/${id}/logs?token=${authStore.auth}`)

    logEventSource.onmessage = event => {
      if (event.data === "ping") return
      if (event.data === "EOF" || event.data === '["EOF"]') {
        logEventSource?.close()
        logEventSource = null
        void options.onFinished()
        return
      }
      logsData.value.push(event.data)
    }

    logEventSource.onerror = () => {
      logsData.value.push("连接已断开或发生错误")
      logEventSource?.close()
      logEventSource = null
    }
  }

  function handleLogModalChange(value: boolean) {
    logModalVisible.value = value
    if (!value) {
      logEventSource?.close()
      logEventSource = null
    }
  }

  return {
    logsData,
    logModalVisible,
    openLogModal,
    handleLogModalChange
  }
}
