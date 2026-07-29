<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from "vue"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import "xterm/css/xterm.css"
import { useAuthStore } from "@/store/auth"

const authStore = useAuthStore()

const props = defineProps<{
  taskId: number | null
  groupId?: number | null
  defaultAgent?: string
}>()

const emit = defineEmits<{
  (e: 'task-created', taskId: number): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
let term: Terminal
let fitAddon: FitAddon
let ws: WebSocket
let pingInterval: any
let resizeObserver: ResizeObserver | null = null
let intentionalClose = false
let serverErrorShown = false

const initTerminal = () => {
  if (!terminalRef.value) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4'
    }
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalRef.value)
  
  nextTick(() => {
    try {
      fitAddon.fit()
    } catch (e) {
      console.warn("Fit addon error on init", e)
    }
  })

  // 建立 WebSocket 连接
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  const token = authStore.auth || ""
  
  let wsUrl = `${protocol}//${window.location.host}/api/code/terminal?token=${token}&cols=${term.cols}&rows=${term.rows}`
  
  if (props.taskId) {
    wsUrl += `&task_id=${props.taskId}`
  } else {
    if (props.defaultAgent) {
      wsUrl += `&agent=${props.defaultAgent}`
    }
    if (props.groupId) {
      wsUrl += `&project_id=${props.groupId}`
    }
  }
  
  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    if (props.taskId) {
      term.writeln(`\x1b[32m[GoPanel] 正在恢复历史任务 #${props.taskId} 的上下文...\x1b[0m\r\n`)
    }
  }

  ws.onmessage = (event) => {
    if (typeof event.data === "string" && (event.data.includes("失败") || event.data.includes("错误"))) {
      serverErrorShown = true
    }
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === "cmd") {
        term.write(msg.data)
      } else if (msg.type === "meta" && msg.task_id) {
        // 后端通知前端：新任务已创建，请更新 URL 或左侧列表
        emit('task-created', msg.task_id)
      } else if (msg.type === "pong") {
        // do nothing
      } else {
        term.write(event.data)
      }
    } catch (e) {
      term.write(event.data)
    }
  }

  ws.onerror = () => {
    term.writeln("\r\n\x1b[31m[错误] WebSocket 连接失败。\x1b[0m")
  }

  ws.onclose = () => {
    if (intentionalClose) {
      return
    }
    if (!serverErrorShown) {
      term.writeln("\r\n\x1b[33m[系统] 终端连接已断开。\x1b[0m")
    }
  }

  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "cmd", data }))
    }
  })

  pingInterval = setInterval(() => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "ping" }))
    }
  }, 15000)
}

const handleResize = () => {
  if (fitAddon) {
    try {
      fitAddon.fit()
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", data: JSON.stringify({ cols: term.cols, rows: term.rows }) }))
      }
    } catch (e) {
      console.warn("Fit addon resize error", e)
    }
  }
}

onMounted(() => {
  initTerminal()
  window.addEventListener("resize", handleResize)
  if (terminalRef.value && typeof ResizeObserver !== "undefined") {
    resizeObserver = new ResizeObserver(() => {
      handleResize()
    })
    resizeObserver.observe(terminalRef.value)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener("resize", handleResize)
  resizeObserver?.disconnect()
  if (pingInterval) clearInterval(pingInterval)
  intentionalClose = true
  if (ws) ws.close()
  if (term) term.dispose()
})
</script>

<template>
  <div class="h-[calc(100vh-220px)] min-h-0 w-full">
    <div
      ref="terminalRef"
      class="h-[calc(100vh-220px)] min-h-0 w-full"
    ></div>
  </div>
</template>

<style scoped>
:deep(.xterm),
:deep(.xterm-screen),
:deep(.xterm-helpers),
:deep(.xterm-viewport) {
  height: 100%;
}

:deep(.xterm) {
  padding: 16px 18px;
}

:deep(.xterm-viewport) {
  overflow-y: auto !important;
}
:deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}
:deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: #1e1e1e;
}
:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #424242;
  border-radius: 4px;
}
</style>
