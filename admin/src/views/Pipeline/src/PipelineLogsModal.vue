<script setup lang="ts">
import { ref, watch, nextTick, computed } from "vue"
import { NModal, NButton, NPopconfirm, useMessage, NAlert, NSpace } from "naive-ui"
import { useAuthStore } from "@/store/auth"
import { getPipelineRecords, stopPipeline } from "@/api/modules/pipeline"
import { appsRepairPodmanSubuidAPI } from "@/api/modules/apps"
import type { Pipeline } from "@/api/interface/pipeline"
import { buildRuntimeDetailText } from "@/utils/runtime"

const props = defineProps<{ show: boolean; recordId: number; pipelineId?: number | null }>()
const emit = defineEmits(["update:show", "finished", "retry"])

const logs = ref<string[]>([])
const terminalRef = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null
const authStore = useAuthStore()
const message = useMessage()

const isRunning = ref(false)
const isStopping = ref(false)
const runnerResult = ref<{ hostPort: number; containerId: string } | null>(null)
const currentRecord = ref<Pipeline.ResRecord | null>(null)

const repairTipVisible = ref(false)
const repairTipTitle = ref("")
const repairTipMessage = ref("")
const repairTipAction = ref("")
const repairTipOutput = ref("")
const repairing = ref(false)
const repairSuccess = ref(false)

const scrollToBottom = () => {
  nextTick(() => {
    if (terminalRef.value) {
      terminalRef.value.scrollTop = terminalRef.value.scrollHeight
    }
  })
}

const handleRepair = async () => {
  if (repairing.value) return
  repairing.value = true
  repairTipOutput.value = ""
  try {
    let res: any
    if (repairTipAction.value === "subuid") {
      res = await appsRepairPodmanSubuidAPI()
    }
    
    if (res && res.code === 0) {
      repairTipOutput.value = res.data?.output || "已执行修复，请点击继续执行。"
      repairSuccess.value = true
      message.success("修复已执行")
    } else {
      message.error(res?.msg || "修复失败")
    }
  } catch (e: any) {
    void 0
  } finally {
    repairing.value = false
  }
}

const handleRetry = () => {
  emit("retry")
}

const copyRunnerAddress = async () => {
  if (!runnerResult.value) return
  try {
    await navigator.clipboard.writeText(`127.0.0.1:${runnerResult.value.hostPort}`)
    message.success("运行地址已复制")
  } catch (error: any) {
    message.error(error?.message || "复制失败")
  }
}

const runnerRuntimeText = computed(() => {
  const row = currentRecord.value
  if (!row?.runnerContainerId) return ""
  return buildRuntimeDetailText(row, {
    kindFallback: "Runtime",
    userFallback: "镜像默认",
    runtimePrefix: "",
    runUserPrefix: "运行用户："
  })
})

const fetchCurrentRecord = async () => {
  if (!props.pipelineId || !props.recordId) {
    currentRecord.value = null
    return
  }
  try {
    const res = await getPipelineRecords({
      pipelineId: props.pipelineId,
      page: 1,
      limit: 100
    })
    const items = Array.isArray(res.data?.items) ? res.data.items : []
    currentRecord.value = items.find((item: Pipeline.ResRecord) => Number(item.id) === Number(props.recordId)) || null
  } catch (error) {
    currentRecord.value = null
  }
}

const startLogs = () => {
  if (eventSource) {
    eventSource.close()
  }
  logs.value = []
  isRunning.value = true
  isStopping.value = false
  runnerResult.value = null
  fetchCurrentRecord()
  
  repairTipVisible.value = false
  repairSuccess.value = false
  repairTipTitle.value = ""
  repairTipMessage.value = ""
  repairTipAction.value = ""
  repairTipOutput.value = ""
  
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

      const runnerMatch = event.data.match(/Runner 容器已启动：containerId=([^,\s]+), hostPort=(\d+)/)
      if (runnerMatch) {
        runnerResult.value = {
          containerId: runnerMatch[1],
          hostPort: Number(runnerMatch[2])
        }
      }

      if (event.data.includes("insufficient UIDs or GIDs")) {
        repairTipVisible.value = true
        repairTipTitle.value = "检测到 UID/GID 映射不足"
        repairTipMessage.value = "当前用户缺乏足够的子 UID/GID 映射，导致无法创建容器命名空间。可以点击一键修复，系统将自动配置并重置命名空间。"
        repairTipAction.value = "subuid"
      }

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
    <n-alert
      v-if="runnerResult"
      type="success"
      title="运行实例已启动"
      class="mb-4"
    >
      <div class="text-sm">
        当前流水线已产出代码运行实例，桥接地址：
        <span class="font-mono text-emerald-700">127.0.0.1:{{ runnerResult.hostPort }}</span>
        <span class="ml-2 text-slate-500">容器 ID：{{ runnerResult.containerId.slice(0, 12) }}</span>
        <span
          v-if="runnerRuntimeText"
          class="ml-2 text-slate-500"
        >{{ runnerRuntimeText }}</span>
        <n-button
          size="tiny"
          class="ml-3"
          type="primary"
          quaternary
          @click="copyRunnerAddress"
        >
          复制地址
        </n-button>
      </div>
    </n-alert>

    <n-alert
      v-if="repairTipVisible"
      type="warning"
      :title="repairTipTitle"
      closable
      class="mb-4"
      @close="repairTipVisible = false"
    >
      <div class="text-sm">
        <div v-if="repairTipMessage">{{ repairTipMessage }}</div>
        <div
          v-if="repairTipOutput"
          class="mt-2 whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700"
        >
          {{ repairTipOutput }}
        </div>
        <n-space class="mt-3">
          <n-button
            v-if="!repairSuccess"
            size="small"
            type="primary"
            :loading="repairing"
            @click="handleRepair"
          >
            一键修复
          </n-button>
          <n-button
            v-else
            size="small"
            type="success"
            @click="handleRetry"
          >
            继续执行
          </n-button>
        </n-space>
      </div>
    </n-alert>

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
