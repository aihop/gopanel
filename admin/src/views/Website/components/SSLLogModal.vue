<script setup lang="ts">
defineProps<{
  show: boolean
  logsData: string[]
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="SSL自动签证日志"
    style="width: 900px"
    @update:show="emit('update:show', $event)"
  >
    <div class="h-[400px] overflow-y-auto rounded-xl bg-slate-900 p-4 font-mono text-sm text-green-400">
      <div v-if="logsData.length === 0" class="text-slate-500">正在建立连接...</div>
      <div v-for="(log, index) in logsData" :key="index" class="mb-1 break-words">
        {{ log }}
      </div>
    </div>
  </n-modal>
</template>
