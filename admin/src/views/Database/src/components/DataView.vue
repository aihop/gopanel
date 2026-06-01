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
const selectedFile = ref<File | null>(null)

const formatFileSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const handleFileSelected = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  importFilename.value = file.name
  selectedFile.value = file
  // Auto-detect format from extension
  if (file.name.endsWith('.sql')) {
    importFormat.value = 'sql'
  } else {
    importFormat.value = 'csv'
  }
  // Clear pasted content when a file is selected
  importContent.value = ''
}

const clearFileSelection = () => {
  selectedFile.value = null
  importFilename.value = ''
}

const handleImport = async () => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  if (!importContent.value.trim() && !selectedFile.value) {
    message.warning('请选择文件或粘贴内容')
    return
  }

  importing.value = true
  try {
    let res: any

    if (selectedFile.value) {
      // File upload path: multipart upload
      const { uploadDBManagerImportAPI } = await import('@/api/modules/database')
      res = await uploadDBManagerImportAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format: importFormat.value,
        file: selectedFile.value
      })
    } else {
      // Paste path: JSON with content string
      const { importDBManagerTableAPI } = await import('@/api/modules/database')
      res = await importDBManagerTableAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format: importFormat.value,
        content: importContent.value
      })
    }

    if (res.code === 0) {
      const imported = (res.data as any)?.imported || 0
      message.success(`导入成功，共 ${imported} 条记录`)
      showImportModal.value = false
      importContent.value = ''
      importFilename.value = ''
      selectedFile.value = null
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
  exportAllColumns,
  exportColumnOptions,
  exportColumns,
  exportFormat,
  exportIncludeCreateTable,
  exportIncludeDropTable,
  exportWhere,
  exporting,
  fetchTableData,
  fetchTableList,
  handleBatchDelete,
  handleCellDblClick,
  handleCellEditSave,
  handleCellEditCancel,
  handleExportCSV,
  handleExportSQL,
  handleExportWithOptions,
  handleOpenExportModal,
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
  detailColumns,
  detailRow,
  openDetail,
  showDetailModal,
  showExportModal,
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
            @select="(key: string) => handleOpenExportModal(key)"
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
          :scroll-x="'max-content'"
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
          <span v-if="selectedFile" class="text-slate-600 text-xs flex items-center gap-2">
            <n-icon :component="renderIcon('mdi:file-outline')" />
            {{ importFilename }}
            <span class="text-slate-400">({{ formatFileSize(selectedFile.size) }})</span>
            <n-button size="tiny" text type="error" @click="clearFileSelection">清除</n-button>
          </span>
          <span v-else class="text-slate-400 text-xs">未选择文件</span>
        </div>

        <div v-if="!selectedFile">
          <div class="text-slate-500 text-xs mb-1">或直接粘贴 {{ importFormat === 'csv' ? 'CSV' : 'SQL' }} 内容:</div>
          <n-input
            v-model:value="importContent"
            type="textarea"
            :rows="10"
            placeholder="在此粘贴内容..."
            class="w-full"
          />
        </div>
        <div v-else class="rounded bg-blue-50 border border-blue-200 p-3 text-xs text-blue-700">
          <n-icon :component="renderIcon('mdi:cloud-upload-outline')" class="mr-1" />
          文件已选择，点击「开始导入」将直接上传。大文件不会加载到页面中。
        </div>

        <div class="flex justify-end gap-2 mt-2">
          <n-button size="small" @click="showImportModal = false">取消</n-button>
          <n-button
            size="small"
            type="primary"
            :loading="importing"
            @click="handleImport"
          >{{ selectedFile ? '上传并导入' : '开始导入' }}</n-button>
        </div>
      </div>
    </n-modal>

    <!-- Export Modal -->
    <n-modal
      v-model:show="showExportModal"
      preset="card"
      style="width: 500px"
      :title="`导出表数据 - ${selectedTable} (${exportFormat.toUpperCase()})`"
    >
      <div class="flex flex-col gap-4 text-sm">
        <div>
          <div class="text-slate-600 mb-1">导出字段</div>
          <n-radio-group v-model:value="exportAllColumns" class="mb-2">
            <n-radio :value="true" size="small">全部字段</n-radio>
            <n-radio :value="false" size="small">选择字段</n-radio>
          </n-radio-group>
          <n-select
            v-if="!exportAllColumns"
            v-model:value="exportColumns"
            :options="exportColumnOptions"
            multiple
            placeholder="选择要导出的字段"
            class="w-full"
            size="small"
          />
        </div>

        <div>
          <div class="text-slate-600 mb-1">WHERE 条件（可选）</div>
          <n-input
            v-model:value="exportWhere"
            placeholder="例如: status = 1 AND created_at > '2025-01-01'"
            size="small"
            clearable
          />
        </div>

        <div v-if="exportFormat === 'sql'">
          <div class="text-slate-600 mb-2">SQL 选项</div>
          <div class="flex flex-col gap-2">
            <label class="flex items-center gap-2 cursor-pointer">
              <n-switch v-model:value="exportIncludeDropTable" size="small" />
              <span class="text-xs text-slate-600">包含 DROP TABLE IF EXISTS</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <n-switch v-model:value="exportIncludeCreateTable" size="small" />
              <span class="text-xs text-slate-600">包含 CREATE TABLE</span>
            </label>
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-2">
          <n-button size="small" @click="showExportModal = false">取消</n-button>
          <n-button
            size="small"
            type="primary"
            :loading="exporting"
            @click="handleExportWithOptions"
          >导出</n-button>
        </div>
      </div>
    </n-modal>

    <!-- Row Detail Modal -->
    <n-modal
      v-model:show="showDetailModal"
      preset="card"
      style="width: 600px; max-height: 80vh;"
      title="行详情"
      size="small"
    >
      <div class="flex flex-col gap-1 text-sm max-h-[60vh] overflow-y-auto">
        <div
          v-for="col in detailColumns"
          :key="col.key"
          class="flex border-b border-slate-100 last:border-b-0"
        >
          <div class="w-48 shrink-0 px-3 py-2 text-slate-500 font-mono text-xs bg-slate-50 truncate">
            {{ col.label }}
          </div>
          <div class="flex-1 px-3 py-2 text-slate-800 font-mono text-xs break-all">
            {{ detailRow[col.key] === null ? 'NULL' : detailRow[col.key] === undefined ? '' : String(detailRow[col.key]) }}
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <n-button size="small" @click="showDetailModal = false">关闭</n-button>
        </div>
      </template>
    </n-modal>
</template>

<style scoped>
:deep(.db-primary-col) {
  background: var(--bg-secondary-color);
  font-weight: 600;
}

:deep(.db-inline-edit-cell) {
  margin: -4px 0;
}

:deep(.db-inline-edit-cell .n-input) {
  min-height: 28px;
}

:deep(.db-cell-inline):hover {
  background: color-mix(in srgb, var(--primary-color) 10%, var(--bg-default-color));
  border-radius: 3px;
}
</style>
