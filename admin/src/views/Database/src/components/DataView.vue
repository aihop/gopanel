<script setup lang="ts">
import { ref } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NPagination, NInput, NSelect, NInputGroup, NDropdown, NModal, NSwitch, NRadio, NRadioGroup } from 'naive-ui'
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

// Import state
const showImportModal = ref(false)
const importFormat = ref('csv')
const importContent = ref('')
const importing = ref(false)
const importFilename = ref('')

const handleFileSelected = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  importFilename.value = file.name
  const reader = new FileReader()
  reader.onload = () => {
    importContent.value = reader.result as string
    // Auto-detect format from extension
    if (file.name.endsWith('.sql')) {
      importFormat.value = 'sql'
    } else {
      importFormat.value = 'csv'
    }
  }
  reader.readAsText(file)
}

const handleImport = async () => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  if (!importContent.value.trim()) {
    message.warning('请选择文件或粘贴内容')
    return
  }

  importing.value = true
  try {
    const { importDBManagerTableAPI } = await import('@/api/modules/database')
    const res = await importDBManagerTableAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      tableName: props.selectedTable,
      format: importFormat.value,
      content: importContent.value
    })
    if (res.code === 0) {
      const imported = (res.data as any)?.imported || 0
      message.success(`导入成功，共 ${imported} 条记录`)
      showImportModal.value = false
      importContent.value = ''
      importFilename.value = ''
      fetchTableData()
    } else {
      message.error(res.message || '导入失败')
    }
  } catch (error: any) {
    message.error(error?.message || '导入请求失败')
  } finally {
    importing.value = false
  }
}

const {
  applyTableSearch,
  checkedRowKeys,
  editingCell,
  editingValue,
  fetchTableData,
  fetchTableList,
  handleBatchDelete,
  handleCellDblClick,
  handleCellEditSave,
  handleCellEditCancel,
  handleExportCSV,
  handleExportSQL,
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

defineExpose({ fetchTableData, setAdvancedSearch, handleCellEditCancel })
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
          <template v-if="checkedRowKeys.length > 0">
            <span class="text-[11px] text-blue-600 whitespace-nowrap">已选 {{ checkedRowKeys.length }} 项</span>
            <n-button
              size="tiny"
              type="error"
              @click="handleBatchDelete"
            >
              <template #icon>
                <n-icon :component="renderIcon('mdi:delete')" />
              </template>
              批量删除
            </n-button>
          </template>
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
          <n-dropdown
            trigger="click"
            :options="[
              { key: 'csv', label: '导出 CSV' },
              { key: 'sql', label: '导出 SQL' }
            ]"
            @select="(key: string) => key === 'csv' ? handleExportCSV() : handleExportSQL()"
          >
            <n-button
              size="tiny"
              :disabled="!selectedTable"
            >
              <template #icon>
                <n-icon :component="renderIcon('mdi:file-download-outline')" />
              </template>
              导出
            </n-button>
          </n-dropdown>
          <n-button
            size="tiny"
            :disabled="!selectedTable"
            @click="showImportModal = true"
          >
            <template #icon>
              <n-icon :component="renderIcon('mdi:file-upload-outline')" />
            </template>
            导入
          </n-button>
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

    <!-- Import Modal -->
    <n-modal
      v-model:show="showImportModal"
      preset="card"
      style="width: 560px"
      title="导入数据"
    >
      <div class="flex flex-col gap-4 text-sm">
        <div class="flex items-center gap-3">
          <span class="text-slate-700 w-16">格式:</span>
          <n-radio-group v-model:value="importFormat">
            <n-radio value="csv">CSV</n-radio>
            <n-radio value="sql">SQL</n-radio>
          </n-radio-group>
        </div>

        <div class="flex items-center gap-3">
          <span class="text-slate-700 w-16">文件:</span>
          <label class="cursor-pointer px-3 py-1.5 rounded bg-blue-50 text-blue-600 hover:bg-blue-100 border border-blue-200 text-xs">
            选择文件
            <input
              type="file"
              accept=".csv,.sql,.txt"
              class="hidden"
              @change="handleFileSelected"
            />
          </label>
          <span class="text-slate-400 text-xs">{{ importFilename || '未选择文件' }}</span>
        </div>

        <div>
          <div class="text-slate-500 text-xs mb-1">或直接粘贴 {{ importFormat === 'csv' ? 'CSV' : 'SQL' }} 内容:</div>
          <n-input
            v-model:value="importContent"
            type="textarea"
            :rows="10"
            placeholder="在此粘贴内容..."
            class="w-full"
          />
        </div>

        <div class="flex justify-end gap-2 mt-2">
          <n-button size="small" @click="showImportModal = false">取消</n-button>
          <n-button
            size="small"
            type="primary"
            :loading="importing"
            @click="handleImport"
          >开始导入</n-button>
        </div>
      </div>
    </n-modal>
</template>

<style scoped>
:deep(.db-primary-col) {
  background: #f8fafc;
  font-weight: 600;
}

:deep(.db-inline-edit-cell) {
  margin: -4px 0;
}

:deep(.db-inline-edit-cell .n-input) {
  min-height: 28px;
}

:deep(.db-cell-inline):hover {
  background: #f0f5ff;
  border-radius: 3px;
}
</style>
