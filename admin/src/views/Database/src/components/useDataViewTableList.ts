import { computed, h, ref } from 'vue'
import { NButton } from 'naive-ui'
import { getDBManagerTableListAPI } from '@/api/modules/database'
import type { DataTableSorter, DataViewEmit, DataViewProps, MessageLike } from './dataViewTypes'

export const useDataViewTableList = (
  props: DataViewProps,
  emit: DataViewEmit,
  message: MessageLike
) => {
  const loadingTables = ref(false)
  const tableList = ref<any[]>([])
  const tableListPagination = ref({ page: 1, limit: 20, itemCount: 0 })
  const tableSearchInput = ref('')
  const tableKeyword = ref('')
  const tableSortState = ref<{ columnKey: string | null; order: 'ascend' | 'descend' | false }>({
    columnKey: null,
    order: false
  })

  const formatCell = (value: any) => value === null || value === undefined || value === '' ? '-' : String(value)
  const formatCount = (value: any) => {
    if (value === null || value === undefined || value === '') return '-'
    const count = Number(value)
    return Number.isNaN(count) ? String(value) : count.toLocaleString('zh-CN')
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
    return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString('zh-CN', { hour12: false })
  }

  const selectedServerLabel = computed(() => props.selectedServerId
    ? props.serverOptions.find(server => server.value === props.selectedServerId)?.label || ''
    : '')
  const tableListSummary = computed(() => ({
    total: tableListPagination.value.itemCount,
    currentPage: tableListPagination.value.page,
    keyword: tableKeyword.value
  }))

  const tableListColumns = [
    {
      title: '表名', key: 'name', minWidth: 180, ellipsis: { tooltip: true as const }, sorter: true,
      sortOrder: () => tableSortState.value.columnKey === 'name' ? tableSortState.value.order : false,
      render: (row: any) => h(NButton, { text: true, type: 'primary', onClick: () => emit.selectTable(row.name) }, { default: () => row.name || '-' })
    },
    { title: '类型', key: 'tableType', width: 150, render: (row: any) => formatCell(row.tableType) },
    { title: '引擎', key: 'engine', width: 110, render: (row: any) => formatCell(row.engine) },
    {
      title: '行数', key: 'rowCount', width: 120, sorter: true,
      sortOrder: () => tableSortState.value.columnKey === 'rowCount' ? tableSortState.value.order : false,
      render: (row: any) => formatCount(row.rowCount)
    },
    {
      title: '大小', key: 'sizeBytes', width: 120, sorter: true,
      sortOrder: () => tableSortState.value.columnKey === 'sizeBytes' ? tableSortState.value.order : false,
      render: (row: any) => formatBytes(row.sizeBytes)
    },
    { title: '排序规则', key: 'collation', width: 160, ellipsis: { tooltip: true as const }, render: (row: any) => formatCell(row.collation) },
    {
      title: '更新时间', key: 'updateTime', width: 180, sorter: true,
      sortOrder: () => tableSortState.value.columnKey === 'updateTime' ? tableSortState.value.order : false,
      render: (row: any) => formatDateTime(row.updateTime)
    },
    { title: '备注', key: 'comment', minWidth: 180, ellipsis: { tooltip: true as const }, render: (row: any) => formatCell(row.comment) }
  ]

  const resetTableListState = () => {
    tableList.value = []
    tableListPagination.value = { page: 1, limit: tableListPagination.value.limit, itemCount: 0 }
    tableSearchInput.value = ''
    tableKeyword.value = ''
    tableSortState.value = { columnKey: null, order: false }
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
  const handleTableListSorterChange = (sorter: DataTableSorter | DataTableSorter[] | null) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : sorter
    tableSortState.value = {
      columnKey: currentSorter?.order ? String(currentSorter.columnKey || '') || null : null,
      order: currentSorter?.order || false
    }
    tableListPagination.value.page = 1
    fetchTableList()
  }

  return {
    applyTableSearch, fetchTableList, handleTableListPageChange, handleTableListPageSizeChange,
    handleTableListSorterChange, loadingTables, resetTableListState, resetTableSearch,
    selectedServerLabel, tableKeyword, tableList, tableListColumns, tableListPagination,
    tableListSummary, tableSearchInput
  }
}
