<script setup lang="ts">
import { nextTick, ref, watch } from "vue"

const props = defineProps<{
  show: boolean
  logsData: string[]
  isInstallFinished: boolean
  repairTipVisible: boolean
  repairTipTitle: string
  repairTipMessage: string
  repairTipCommands: string
  repairTipOutput: string
  repairTipAction: string
  repairingCompose: boolean
  retryingInstall: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "repair"): void
  (e: "rebuild"): void
  (e: "retry"): void
  (e: "copy"): void
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
    title="应用安装进度"
    style="width: 700px"
    :mask-closable="false"
    :closable="true"
    @update:show="emit('update:show', $event)"
  >
    <n-alert
      v-if="repairTipVisible"
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
            v-if="isInstallFinished && repairTipAction === 'port-conflict'"
            size="small"
            secondary
            @click="emit('rebuild')"
          >重新重建</n-button>
          <n-button
            v-if="isInstallFinished && repairTipAction && repairTipAction !== 'port-conflict'"
            size="small"
            secondary
            :loading="retryingInstall"
            @click="emit('retry')"
          >重新安装</n-button>
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
      class="h-[400px] overflow-y-auto rounded-md bg-[#1e1e1e] p-4 font-mono text-sm leading-relaxed text-[#d4d4d4]"
    >
      <div
        v-for="(log, index) in logsData"
        :key="index"
        class="whitespace-pre-wrap break-all"
      >
        <span
          v-if="log.includes('ERROR')"
          class="text-red-400"
        >{{ log }}</span>
        <span
          v-else-if="log.includes('INFO')"
          class="text-blue-300"
        >{{ log }}</span>
        <span v-else>{{ log }}</span>
      </div>
    </div>
    <template #action>
      <n-button
        :disabled="!isInstallFinished"
        type="primary"
        @click="emit('update:show', false)"
      >关闭</n-button>
    </template>
  </n-modal>
</template>
