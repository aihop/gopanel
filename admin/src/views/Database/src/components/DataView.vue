<script setup lang="ts">
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NPagination, NInput, NSelect, NInputGroup } from 'naive-ui'
import { renderIcon } from '@/utils'
import { useDataView } from './useDataView'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
}>()

const emit = defineEmits<{
  (e: 'editRecord', row: any): void
  (e: 'copyRecord', row: any): void
  (e: 'selectTable', tableName: string): void
}>()

const message = useMessage()

const {
  applyTableSearch,
  fetchTableData,
  fetchTableList,
  handleReset,
  handleSearch,
  handleTableListPageChange,
  handleTableListPageSizeChange,
  handleTableListSorterChange,
  loadingData,
  loadingTables,
  pagination,
  recordSummary,
  searchColumn,
  searchOptions,
  searchValue,
  selectedServerLabel,
  setAdvancedSearch,
  tableColumns,
  tableData,
  tableKeyword,
  tableList,
  tableListColumns,
  tableListPagination,
  tableListSummary,
  tableSearchInput,
  resetTableSearch
} = useDataView(props, {
  editRecord: (row: any) => emit('editRecord', row),
  copyRecord: (row: any) => emit('copyRecord', row),
  selectTable: (tableName: string) => emit('selectTable', tableName)
}, message)

defineExpose({ fetchTableData, setAdvancedSearch })
</script>

<template>
  <div class="flex-1 flex flex-col relative overflow-hidden">
    <div
      v-if="!selectedDatabase"
      class="flex-1 flex items-center justify-center text-slate-400"
    >
      <n-empty description="请选择数据库" />
    </div>
    <template v-else-if="!selectedTable">
      <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs">
        <div class="text-slate-700 flex items-center gap-1">
          <n-icon :component="renderIcon('mdi:server')" />
          <span class="mr-2">{{ selectedServerLabel }}</span>
          <span>»</span>
          <n-icon
            :component="renderIcon('mdi:database')"
            class="ml-2"
          />
          <span class="font-bold">{{ selectedDatabase }}</span>
        </div>
        <div class="flex items-center gap-2">
          <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500">
            <span class="px-2 py-1 rounded bg-slate-100">共 {{ tableListSummary.total }} 张表</span>
            <span
              v-if="tableKeyword"
              class="px-2 py-1 rounded bg-blue-50 text-blue-600"
            >筛选：{{ tableKeyword }}</span>
          </div>
          <n-input-group>
            <n-input
              v-model:value="tableSearchInput"
              placeholder="输入表名后回车搜索"
              size="tiny"
              style="width: 220px"
              clearable
              @keyup.enter="applyTableSearch"
              @clear="resetTableSearch"
            />
            <n-button
              size="tiny"
              type="primary"
              @click="applyTableSearch"
            >
              <template #icon>
                <n-icon :component="renderIcon('mdi:magnify')" />
              </template>
            </n-button>
            <n-button
              size="tiny"
              @click="resetTableSearch"
            >
              <template #icon>
                <n-icon :component="renderIcon('mdi:refresh')" />
              </template>
            </n-button>
          </n-input-group>
          <n-button
            size="tiny"
            @click="fetchTableList"
            :loading="loadingTables"
          >刷新</n-button>
        </div>
      </div>
      <div class="flex-1 overflow-auto p-2 bg-white">
        <n-data-table
          :columns="tableListColumns as any"
          :data="tableList"
          :loading="loadingTables"
          :pagination="false"
          :bordered="true"
          size="small"
          :scroll-x="1200"
          class="h-full text-xs"
          flex-height
          @update:sorter="handleTableListSorterChange"
        />
      </div>
      <div class="p-2 border-t border-slate-200 flex justify-between items-center bg-[#f8f9fa] z-10">
        <div class="text-xs text-slate-500">
          共 {{ tableListPagination.itemCount }} 张表
        </div>
        <n-pagination
          v-model:page="tableListPagination.page"
          v-model:page-size="tableListPagination.limit"
          :item-count="tableListPagination.itemCount"
          show-size-picker
          :page-sizes="[10, 20, 50, 100]"
          @update:page="handleTableListPageChange"
          @update:page-size="handleTableListPageSizeChange"
          size="small"
        />
      </div>
    </template>
    <template v-else>
      <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs">
        <div class="text-slate-700 flex items-center gap-1">
          <n-icon :component="renderIcon('mdi:server')" />
          <span class="mr-2">{{ selectedServerLabel }}</span>
          <span>»</span>
          <n-icon
            :component="renderIcon('mdi:database')"
            class="ml-2"
          />
          <span class="mr-2">{{ selectedDatabase }}</span>
          <span>»</span>
          <n-icon
            :component="renderIcon('mdi:table')"
            class="ml-2"
          />
          <span class="font-bold">{{ selectedTable }}</span>
        </div>
        <div class="flex items-center gap-2">
          <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500">
            <span class="px-2 py-1 rounded bg-slate-100">{{ recordSummary.total }} 条记录</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ recordSummary.columnCount }} 列</span>
            <span
              v-if="recordSummary.hasFilters"
              class="px-2 py-1 rounded bg-blue-50 text-blue-600"
            >已启用筛选</span>
          </div>
          <n-input-group>
            <n-select
              v-model:value="searchColumn"
              :options="searchOptions"
              placeholder="选择字段"
              size="tiny"
              style="width: 120px"
              clearable
            />
            <n-input
              v-model:value="searchValue"
              placeholder="搜索值..."
              size="tiny"
              style="width: 150px"
              @keyup.enter="handleSearch"
              clearable
            />
            <n-button
              size="tiny"
              type="primary"
              @click="handleSearch"
            >
              <template #icon>
                <n-icon :component="renderIcon('mdi:magnify')" />
              </template>
            </n-button>
            <n-button
              size="tiny"
              @click="handleReset"
            >
              <template #icon>
                <n-icon :component="renderIcon('mdi:refresh')" />
              </template>
            </n-button>
          </n-input-group>
          <n-button
            size="tiny"
            @click="fetchTableData"
            :loading="loadingData"
          >刷新</n-button>
        </div>
      </div>
      <div class="flex-1 overflow-auto p-2 bg-white">
        <n-data-table
          :columns="tableColumns"
          :data="tableData"
          :loading="loadingData"
          :pagination="false"
          :bordered="true"
          size="small"
          :scroll-x="tableColumns.length * 120"
          class="h-full text-xs"
          flex-height
        />
      </div>
      <div class="p-2 border-t border-slate-200 flex justify-end bg-[#f8f9fa] z-10">
        <n-pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.limit"
          :item-count="pagination.itemCount"
          show-size-picker
          :page-sizes="[10, 20, 50, 100]"
          @update:page="fetchTableData"
          @update:page-size="fetchTableData"
          size="small"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
:deep(.db-primary-col) {
  background: #f8fafc;
  font-weight: 600;
}
</style>
