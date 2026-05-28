<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { renderIcon } from '@/utils'

defineProps<{
  serverLabel?: string
  databaseName?: string | null
  tableName?: string | null
  title: string
  icon?: string
}>()
</script>

<template>
  <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs gap-3">
    <div class="min-w-0 flex items-center gap-1 text-slate-700">
      <n-icon :component="renderIcon('mdi:server')" />
      <span class="mr-2 truncate">{{ serverLabel || '-' }}</span>
      <span>»</span>
      <n-icon :component="renderIcon('mdi:database')" class="ml-2" />
      <span class="mr-2 truncate">{{ databaseName || '-' }}</span>
      <template v-if="tableName">
        <span>»</span>
        <n-icon :component="renderIcon(icon || 'mdi:table')" class="ml-2" />
        <span class="font-bold truncate">{{ title }}</span>
      </template>
      <template v-else>
        <span class="font-bold truncate ml-2">{{ title }}</span>
      </template>
    </div>
    <div class="shrink-0 flex items-center gap-2">
      <slot name="summary" />
      <slot name="actions" />
    </div>
  </div>
</template>
