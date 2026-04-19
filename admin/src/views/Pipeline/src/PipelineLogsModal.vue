<script setup lang="ts">
import { ref, watch, nextTick } from "vue"
import { NModal, NButton, NPopconfirm, useMessage } from "naive-ui"
import { useAuthStore } from "@/store/auth"
import { stopPipeline } from "@/api/modules/pipeline"

const props = defineProps<{ show: boolean; recordId: number }>()
const emit = defineEmits(["update:show", "finished"])

const logs = ref<string[]>([])
const terminalRef = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null
const authStore = useAuthStore()
const message = useMessage()

const isRunning = ref(false)
const isStopping = ref(false)

const scrollToBottom = () => {
  nextTick(() => {
    if (terminalRef.value) {
      terminalRef.value.scrollTop = terminalRef.value.scrollHeight
    }
  })
}

const startLogs = () => {
  if (eventSource) {
    eventSource.close()
  }
  logs.value = []
  isRunning.value = true
  isStopping.value = false
  
  const apiUrl = (window as any).__VITE_API_URL__ || "/api"
  const safeToken = encodeURIComponent(authStore.auth)
  eventSource = new EventSource(`${apiUrl}/pipeline/logs?recordId=${props.recordId}&token=${safeToken}`)
  
  eventSource.onmessage = (event) => {
    if (event.data === "ping" || event.data === ":") return
    if (event.data === "connection closed" || event.data === "EOF" || event.data === '["EOF"]') {
      eventSource?.close()
      eventSource = null
      isRunning.value = false
      logs.value.push("====== 流水线执行结束 ======")
      emit("finished")
      scrollToBottom()
      return
    }
    if (event.data) {
      // 限制最大日志行数，防止内存溢出和页面卡顿
      if (logs.value.length > 3000) {
        logs.value.splice(0, logs.value.length - 2000) // 截断保留最新的 2000 行
        logs.value.unshift("... 之前的日志已折叠，请在后台查看完整日志文件 ...")
      }
      logs.value.push(event.data)
      scrollToBottom()
    }
  }

  eventSource.onerror = (err) => {
    console.error("SSE Error:", err)
    isRunning.value = false
    logs.value.push("连接已断开或发生错误")
    eventSource?.close()
    eventSource = null
    emit("finished")
  }
}

const stopLogs = () => {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  isRunning.value = false
}

const handleStopPipeline = async () => {
  isStopping.value = true
  try {
    await stopPipeline({ id: props.recordId })
    message.success("已发送停止指令")
  } catch (error: any) {
    message.error(error.message || "停止流水线失败")
    isStopping.value = false
  }
}

watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      startLogs()
    } else {
      stopLogs()
    }
  },
  { immediate: true }
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="流水线日志"
    style="width: 800px;"
    class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(0,0,0,0.5)] sm:w-[90%]"
    @update:show="(val) => emit('update:show', val)"
  >
    <div
      ref="terminalRef"
      class="h-[500px] overflow-y-auto rounded-lg bg-[#0F0F0F] p-4 font-mono text-sm leading-relaxed text-gray-300 shadow-inner"
    >
      <div
        v-for="(log, idx) in logs"
        :key="idx"
        class="whitespace-pre-wrap break-words"
      >
        {{ log }}
      </div>
      <div
        v-if="logs.length === 0"
        class="text-gray-500 italic"
      >
        正在连接日志流...
      </div>
    </div>

    <div class="mt-4 flex justify-between items-center">
      <div>
        <n-popconfirm
          v-if="isRunning"
          @positive-click="handleStopPipeline"
          negative-text="取消"
          positive-text="确定停止"
        >
          <template #trigger>
            <n-button
              type="error"
              :loading="isStopping"
              ghost
            >
              强制停止执行
            </n-button>
          </template>
          确定要强制终止正在执行的流水线吗？
        </n-popconfirm>
      </div>
      <n-button
        type="primary"
        ghost
        @click="emit('update:show', false)"
      >关闭</n-button>
    </div>
  </n-modal>
</template>

<style scoped>
/* Custom scrollbar for terminal */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
::-webkit-scrollbar-track {
  background: #1e1e1e;
  border-radius: 4px;
}
::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
  background: #555555;
}
</style>
