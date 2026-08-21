<script setup lang="ts">
import { NButton, NIcon } from 'naive-ui'
import { renderIcon } from '@/utils'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  selectedServerLabel: string
  selectedDatabase: string | null
  selectedTable: string | null
  activeTabLabel: string
  tableCount: number
  databaseSqlMode: boolean
}>()

const emit = defineEmits<{
  (e: 'backToTables'): void
}>()
</script>

<template>
  <div class="border-b border-slate-200 bg-[#f8f9fa] px-4 py-3 flex items-center justify-between gap-4">
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2 text-xs text-slate-500 mb-1">
        <n-icon :component="renderIcon('mdi:server')" />
        <span class="truncate">{{ selectedServerLabel }}</span>
        <template v-if="selectedDatabase">
          <span>»</span>
          <n-icon :component="renderIcon('mdi:database')" />
          <span class="truncate">{{ selectedDatabase }}</span>
        </template>
        <template v-if="selectedTable">
          <span>»</span>
          <n-icon :component="renderIcon('mdi:table')" />
          <span class="truncate font-semibold text-slate-700">{{ selectedTable }}</span>
        </template>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <div class="text-sm font-semibold text-slate-800">
          {{
            databaseSqlMode
              ? t('databaseManager.context.selectedDatabaseSqlFmt', [selectedDatabase])
              : selectedTable
                ? t('databaseManager.context.selectedTableFmt', [selectedTable, activeTabLabel])
                : selectedDatabase
                  ? t('databaseManager.context.selectedDatabaseFmt', [selectedDatabase])
                  : t('databaseManager.context.managed')
          }}
        </div>
        <div class="px-2 py-0.5 rounded-full bg-slate-100 text-[11px] text-slate-600">
          {{ tableCount }} {{ t('databaseManager.context.tables') }}
        </div>
        <div
          v-if="selectedTable"
          class="px-2 py-0.5 rounded-full bg-blue-50 text-[11px] text-blue-600"
        >
          {{ t('databaseManager.context.contextSynced') }}
        </div>
      </div>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <n-button
        v-if="selectedTable || databaseSqlMode"
        size="tiny"
        @click="emit('backToTables')"
      >
        {{ databaseSqlMode ? $t('database.backToObjects') : $t('database.backToTables') }}
      </n-button>
    </div>
  </div>
</template>
