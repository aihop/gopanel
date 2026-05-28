<script setup lang="ts">
import { NSelect, NInput, NSpin, NEmpty, NMenu, NIcon, NPagination } from 'naive-ui'
import { renderIcon } from '@/utils'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: Array<{ label: string; value: number; type: string }>
  databaseOptions: Array<{ label: string; value: string }>
  tableKeywordInput: string
  loadingTables: boolean
  tables: string[]
  filteredTableCount: number
  sidebarTablePage: number
  sidebarTablePageSize: number
  menuOptions: any[]
}>()

const emit = defineEmits<{
  (e: 'update:selectedServerId', value: number | null): void
  (e: 'update:selectedDatabase', value: string | null): void
  (e: 'update:tableKeywordInput', value: string): void
  (e: 'update:sidebarTablePage', value: number): void
  (e: 'update:sidebarTablePageSize', value: number): void
  (e: 'selectTable', tableName: string): void
  (e: 'searchTables'): void
  (e: 'resetTableSearch'): void
}>()
</script>

<template>
  <div class="w-72 border border-slate-300 bg-[#f8f9fa] flex flex-col overflow-hidden text-sm rounded-md">
    <div class="p-3 border-b border-slate-300 bg-[#e5e5e5] flex flex-col gap-3">
      <div class="flex items-center gap-1 font-semibold text-slate-700">
        <n-icon :component="renderIcon('mdi:server')" />
        <span>{{ $t('database.server') }}</span>
      </div>
      <n-select
        :value="selectedServerId"
        :options="serverOptions"
        size="small"
        clearable
        @update:value="emit('update:selectedServerId', $event)"
      />

      <div class="flex items-center gap-1 font-semibold text-slate-700">
        <n-icon :component="renderIcon('mdi:database')" />
        <span>{{ $t('database.database') }}</span>
      </div>
      <n-select
        :value="selectedDatabase"
        :options="databaseOptions"
        size="small"
        clearable
        :disabled="!selectedServerId"
        @update:value="emit('update:selectedDatabase', $event)"
      />

      <n-input
        :value="tableKeywordInput"
        size="small"
        clearable
        :disabled="!selectedDatabase || loadingTables || tables.length === 0"
        placeholder="输入表名后回车搜索"
        @update:value="emit('update:tableKeywordInput', $event)"
        @keyup.enter="emit('searchTables')"
        @clear="emit('resetTableSearch')"
      />
    </div>

    <div class="px-3 py-2 text-xs text-slate-500 border-b border-slate-200 bg-white">
      <span>表</span>
      <span class="mx-1">·</span>
      <span>{{ tables.length }}</span>
      <span>/</span>
      <span>{{ filteredTableCount }}</span>
    </div>

    <div class="flex-1 overflow-y-auto p-1 bg-white">
      <div
        v-if="loadingTables"
        class="flex justify-center p-4"
      >
        <n-spin size="small" />
      </div>
      <n-empty
        v-else-if="tables.length === 0"
        :description="$t('database.noTable')"
        class="mt-10"
      />
      <n-empty
        v-else-if="menuOptions.length === 0"
        description="未找到匹配的数据表"
        class="mt-10"
      />
      <n-menu
        v-else
        :options="menuOptions"
        :value="selectedTable"
        @update:value="emit('selectTable', $event)"
        :root-indent="12"
        :indent="12"
        class="text-xs"
      />
    </div>
    <div
      v-if="filteredTableCount > 0"
      class="px-2 py-2 border-t border-slate-200 bg-white"
    >
      <n-pagination
        :page="sidebarTablePage"
        :page-size="sidebarTablePageSize"
        :item-count="filteredTableCount"
        :page-sizes="[20, 30, 50, 100]"
        size="small"
        show-size-picker
        @update:page="emit('update:sidebarTablePage', $event)"
        @update:page-size="emit('update:sidebarTablePageSize', $event)"
      />
    </div>
  </div>
</template>
