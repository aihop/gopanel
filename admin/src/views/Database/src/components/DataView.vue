<script setup lang="ts">
import { ref, watch, h } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NPagination, NPopconfirm, NInput, NSelect, NInputGroup } from 'naive-ui'
import { getDBManagerTableDataAPI, deleteDBManagerRecordAPI } from '@/api/modules/database'
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
}>()

const message = useMessage()
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
  pagination.value.page = 1
  searchColumn.value = null
  searchValue.value = ''
  advancedSearch.value = []
  fetchTableData()
}, { immediate: true })

defineExpose({ fetchTableData, setAdvancedSearch })
</script>

<template>
  <div class="flex-1 flex flex-col relative overflow-hidden">
    <div
      v-if="!selectedTable"
      class="flex-1 flex items-center justify-center text-slate-400"
    >
      <n-empty :description="$t('database.selectTable')" />
    </div>
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
