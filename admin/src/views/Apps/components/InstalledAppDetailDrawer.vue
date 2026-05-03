<script setup lang="ts">
import { getRuntimeKindLabel, getRuntimeModeLabel, getRunUserLabel } from "@/utils/runtime"

defineProps<{
  show: boolean
  item: any
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()
</script>

<template>
  <n-drawer
    :show="show"
    placement="right"
    width="400"
    @update:show="emit('update:show', $event)"
  >
    <n-drawer-content
      title="应用详情"
      closable
    >
      <div
        v-if="item"
        class="mb-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-600"
      >
        <div class="font-medium text-slate-800">运行时</div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <n-tag
            size="small"
            round
            :bordered="false"
            type="info"
          >{{ getRuntimeKindLabel(item) }}</n-tag>
          <n-tag
            size="small"
            round
            :bordered="false"
            :type="item.runtimeMode === 'rootless' ? 'success' : 'default'"
          >{{ getRuntimeModeLabel(item) }}</n-tag>
        </div>
        <div class="mt-2 text-xs text-slate-500">运行用户：{{ getRunUserLabel(item) }}</div>
        <div
          v-if="item.runtimeHost"
          class="mt-1 break-all text-xs text-slate-500"
        >Host：{{ item.runtimeHost }}</div>
      </div>

      <pre
        v-if="item"
        class="whitespace-pre-wrap"
      >{{ JSON.stringify(item.app, null, 2) }}</pre>
    </n-drawer-content>
  </n-drawer>
</template>
