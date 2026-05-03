<script setup lang="ts">
import { nextTick, ref, watch } from "vue"

const props = defineProps<{
  show: boolean
  title: string
  logType: "install" | "runtime"
  logsData: string[]
  isInstallFinished: boolean
  canClose: boolean
  repairTipVisible: boolean
  repairTipTitle: string
  repairTipMessage: string
  repairTipCommands: string
  repairTipOutput: string
  repairTipAction: string
  repairingCompose: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "repair"): void
  (e: "rebuild"): void
  (e: "copy"): void
  (e: "close"): void
}>()

const terminalRef = ref<HTMLElement | null>(null)

watch(() => props.logsData.length, () => {
  nextTick(() => {
    if (terminalRef.value) {
      terminalRef.value.scrollTop = terminalRef.value.scrollHeight
    }
  })
})
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="title"
    class="w-[800px]"
    :mask-closable="false"
    @update:show="emit('update:show', $event)"
    @after-leave="emit('close')"
  >
    <div class="mb-3 flex items-center justify-between text-xs text-slate-500">
      <span>实时输出，关闭窗口将断开连接</span>
    </div>

    <n-alert
      v-if="logType === 'install' && repairTipVisible"
      class="mb-3"
      type="warning"
      :title="repairTipTitle"
      :show-icon="true"
    >
      <div class="text-sm leading-6">
        <div v-if="repairTipMessage">{{ repairTipMessage }}</div>
        <div class="mt-2 whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700">{{ repairTipCommands }}</div>
        <div
          v-if="repairTipOutput"
          class="mt-2 whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700"
        >{{ repairTipOutput }}</div>
        <n-space class="mt-3">
          <n-button
            size="small"
            type="primary"
            :loading="repairingCompose"
            @click="emit('repair')"
          >一键修复</n-button>
          <n-button
            v-if="isInstallFinished && repairTipAction"
            size="small"
            secondary
            @click="emit('rebuild')"
          >重新重建</n-button>
          <n-button
            v-if="repairTipCommands"
            size="small"
            secondary
            @click="emit('copy')"
          >复制命令</n-button>
        </n-space>
      </div>
    </n-alert>

    <div
      ref="terminalRef"
      data-installed-log-terminal
      class="max-h-[60vh] overflow-auto whitespace-pre-wrap rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-100"
    >
      <div
        v-if="logsData.length === 0"
        class="text-slate-400"
      >暂无日志输出</div>
      <div
        v-for="(line, idx) in logsData"
        :key="idx"
      >
        <span
          v-if="line.includes('ERROR')"
          class="text-red-400"
        >{{ line }}</span>
        <span
          v-else-if="line.includes('INFO')"
          class="text-blue-300"
        >{{ line }}</span>
        <span v-else>{{ line }}</span>
      </div>
    </div>

    <template #action>
      <n-button
        :disabled="!canClose"
        type="primary"
        @click="emit('close')"
      >关闭</n-button>
    </template>
  </n-modal>
</template>
