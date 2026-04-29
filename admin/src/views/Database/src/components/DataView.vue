<script setup lang="ts">
import { ref, watch, h } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NPagination, NPopconfirm, NInput, NSelect, NInputGroup } from 'naive-ui'
import { getDBManagerTableListAPI, getDBManagerTableDataAPI, deleteDBManagerRecordAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

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
const loadingTables = ref(false)
const tableList = ref<any[]>([])
const tableListPagination = ref({
  page: 1,
  limit: 20,
  itemCount: 0
})
const tableSearchInput = ref('')
const tableKeyword = ref('')
const tableSortState = ref<{ columnKey: string | null; order: 'ascend' | 'descend' | false }>({
  columnKey: null,
  order: false
})

const loadingData = ref(false)
const tableData = ref<any[]>([])
const tableColumns = ref<any[]>([])
const pagination = ref({
  page: 1,
  limit: 20,
  itemCount: 0
})

const searchColumn = ref<string | null>(null)
const searchValue = ref('')
const searchOptions = ref<{label: string, value: string}[]>([])

const advancedSearch = ref<any[]>([])

const tableListColumns = [
  {
    title: '表名',
    key: 'name',
    minWidth: 180,
    ellipsis: { tooltip: true as const },
    sorter: true,
    sortOrder: () => tableSortState.value.columnKey === 'name' ? tableSortState.value.order : false,
    render(row: any) {
      return h(NButton, {
        text: true,
        type: 'primary',
        onClick: () => emit('selectTable', row.name)
      }, { default: () => row.name || '-' })
    }
  },
  {
    title: '类型',
    key: 'tableType',
    width: 150,
    render(row: any) {
      return formatCell(row.tableType)
    }
  },
  {
    title: '引擎',
    key: 'engine',
    width: 110,
    render(row: any) {
      return formatCell(row.engine)
    }
  },
  {
    title: '行数',
    key: 'rowCount',
    width: 120,
    sorter: true,
    sortOrder: () => tableSortState.value.columnKey === 'rowCount' ? tableSortState.value.order : false,
    render(row: any) {
      return formatCount(row.rowCount)
    }
  },
  {
    title: '大小',
    key: 'sizeBytes',
    width: 120,
    sorter: true,
    sortOrder: () => tableSortState.value.columnKey === 'sizeBytes' ? tableSortState.value.order : false,
    render(row: any) {
      return formatBytes(row.sizeBytes)
    }
  },
  {
    title: '排序规则',
    key: 'collation',
    width: 160,
    ellipsis: { tooltip: true as const },
    render(row: any) {
      return formatCell(row.collation)
    }
  },
  {
    title: '更新时间',
    key: 'updateTime',
    width: 180,
    sorter: true,
    sortOrder: () => tableSortState.value.columnKey === 'updateTime' ? tableSortState.value.order : false,
    render(row: any) {
      return formatDateTime(row.updateTime)
    }
  },
  {
    title: '备注',
    key: 'comment',
    minWidth: 180,
    ellipsis: { tooltip: true as const },
    render(row: any) {
      return formatCell(row.comment)
    }
  }
]

const formatCell = (value: any) => {
  if (value === null || value === undefined || value === '') return '-'
  return String(value)
}

const formatCount = (value: any) => {
  if (value === null || value === undefined || value === '') return '-'
  const count = Number(value)
  if (Number.isNaN(count)) return String(value)
  return count.toLocaleString('zh-CN')
}

const formatBytes = (value: any) => {
  if (value === null || value === undefined || value === '') return '-'
  const size = Number(value)
  if (Number.isNaN(size) || size < 0) return String(value)
  if (size === 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let current = size
  let unitIndex = 0
  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024
    unitIndex++
  }
  const digits = current >= 10 || unitIndex === 0 ? 0 : 1
  return `${current.toFixed(digits)} ${units[unitIndex]}`
}

const formatDateTime = (value: any) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString('zh-CN', { hour12: false })
}

const resetTableListState = () => {
  tableList.value = []
  tableListPagination.value.itemCount = 0
  tableListPagination.value.page = 1
  tableSearchInput.value = ''
  tableKeyword.value = ''
  tableSortState.value = {
    columnKey: null,
    order: false
  }
}

const resetTableDataState = () => {
  tableData.value = []
  tableColumns.value = []
  searchOptions.value = []
  pagination.value.itemCount = 0
}

const resetRecordSearch = () => {
  pagination.value.page = 1
  searchColumn.value = null
  searchValue.value = ''
  advancedSearch.value = []
}

const fetchTableList = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) return

  loadingTables.value = true
  try {
    const res = await getDBManagerTableListAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      page: tableListPagination.value.page,
      limit: tableListPagination.value.limit,
      keyword: tableKeyword.value || undefined,
      sortField: tableSortState.value.columnKey || undefined,
      sortOrder: tableSortState.value.order || undefined
    })

    if (res.code === 0 && res.data) {
      tableList.value = res.data.items || []
      tableListPagination.value.itemCount = res.data.total || 0
    } else {
      tableList.value = []
      tableListPagination.value.itemCount = 0
      message.error((res as any)?.message || (res as any)?.msg || '获取数据表列表失败')
    }
  } catch (error) {
    tableList.value = []
    tableListPagination.value.itemCount = 0
    message.error((error as any)?.message || '获取数据表列表失败')
  } finally {
    loadingTables.value = false
  }
}

const applyTableSearch = () => {
  tableKeyword.value = tableSearchInput.value.trim()
  tableListPagination.value.page = 1
  fetchTableList()
}

const resetTableSearch = () => {
  tableSearchInput.value = ''
  tableKeyword.value = ''
  tableListPagination.value.page = 1
  fetchTableList()
}

const handleTableListPageChange = (page: number) => {
  tableListPagination.value.page = page
  fetchTableList()
}

const handleTableListPageSizeChange = (pageSize: number) => {
  tableListPagination.value.limit = pageSize
  tableListPagination.value.page = 1
  fetchTableList()
}

const handleTableListSorterChange = (sorter: { columnKey?: string; order?: 'ascend' | 'descend' | false } | { columnKey?: string; order?: 'ascend' | 'descend' | false }[] | null) => {
  const currentSorter = Array.isArray(sorter) ? sorter[0] : sorter
  tableSortState.value = {
    columnKey: currentSorter?.order ? (currentSorter.columnKey || null) : null,
    order: currentSorter?.order || false
  }
  tableListPagination.value.page = 1
  fetchTableList()
}

const setAdvancedSearch = (conditions: any[]) => {
  advancedSearch.value = conditions
  pagination.value.page = 1
  fetchTableData()
}

const handleSearch = () => {
  pagination.value.page = 1
  fetchTableData()
}

const handleReset = () => {
  searchColumn.value = null
  searchValue.value = ''
  advancedSearch.value = []
  handleSearch()
}

const fetchTableData = async () => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  loadingData.value = true
  try {
    const res = await getDBManagerTableDataAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      tableName: props.selectedTable,
      page: pagination.value.page,
      limit: pagination.value.limit,
      searchColumn: searchColumn.value || undefined,
      searchValue: searchValue.value || undefined,
      advancedSearch: advancedSearch.value.length > 0 ? advancedSearch.value : undefined
    })
    
    if (res.code === 0 && res.data) {
      tableData.value = res.data.rows || []
      pagination.value.itemCount = res.data.total || 0
      
      if (res.data.columns && res.data.columns.length > 0) {
        const visibleCols = (res.data.columns as string[]).filter((col: string) => col !== "__rowid__")
        searchOptions.value = visibleCols.map((col: string) => ({ label: col, value: col }))
        
        const actionCol = {
          title: '操作',
          key: 'actions',
          fixed: 'left',
          width: 140,
          render(row: any) {
            return h('div', { class: 'flex gap-2' }, [
              h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => emit('editRecord', row) }, { default: () => '编辑' }),
              h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => emit('copyRecord', row) }, { default: () => '复制' }),
              h(NPopconfirm, { onPositiveClick: () => deleteRecord(row) }, {
                trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
                default: () => '确定要删除这条记录吗？'
              })
            ])
          }
        }
        tableColumns.value = [
          actionCol,
          ...visibleCols.map((col: string) => ({ title: col, key: col, ellipsis: { tooltip: true as const } }))
        ]
      } else {
        tableColumns.value = []
        searchOptions.value = []
      }
    } else {
      tableData.value = []
      tableColumns.value = []
      pagination.value.itemCount = 0
      message.error((res as any)?.message || (res as any)?.msg || '获取表数据失败')
    }
  } catch (error) {
    tableData.value = []
    tableColumns.value = []
    pagination.value.itemCount = 0
    message.error((error as any)?.message || '获取表数据失败')
  } finally {
    loadingData.value = false
  }
}

const deleteRecord = async (row: any) => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  try {
    const res = await deleteDBManagerRecordAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      tableName: props.selectedTable,
      conditions: row
    })
    if (res.code === 0) {
      message.success('删除成功')
      fetchTableData()
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (error) {
    message.error('删除请求失败')
  }
}

watch(() => props.selectedTable, () => {
  resetRecordSearch()
  if (props.selectedTable) {
    fetchTableData()
    return
  }
  if (props.selectedDatabase) {
    fetchTableList()
  } else {
    resetTableDataState()
  }
})

watch(() => [props.selectedServerId, props.selectedDatabase], ([selectedServerId, selectedDatabase]) => {
  resetTableListState()
  resetRecordSearch()
  resetTableDataState()

  if (!selectedServerId || !selectedDatabase) return
  if (!props.selectedTable) {
    fetchTableList()
  }
}, { immediate: true })

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
          <span class="mr-2">{{ selectedServerId ? serverOptions.find(s => s.value === selectedServerId)?.label : '' }}</span>
          <span>»</span>
          <n-icon
            :component="renderIcon('mdi:database')"
            class="ml-2"
          />
          <span class="font-bold">{{ selectedDatabase }}</span>
        </div>
        <div class="flex items-center gap-2">
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
          :columns="tableListColumns"
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
          <span class="mr-2">{{ selectedServerId ? serverOptions.find(s => s.value === selectedServerId)?.label : '' }}</span>
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
