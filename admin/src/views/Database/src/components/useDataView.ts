import { computed, h, onMounted, ref, watch } from 'vue'
import { NButton, NPopconfirm, NInput } from 'naive-ui'
import { getDBManagerTableListAPI, getDBManagerTableDataAPI, deleteDBManagerRecordAPI, updateDBManagerRecordAPI, exportDBManagerTableAPI, execDBManagerSqlAPI } from '@/api/modules/database'

type MessageLike = {
  error: (content: string) => void
  success: (content: string) => void
}

type DataViewProps = {
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
}

type EmitLike = {
  editRecord: (row: any) => void
  copyRecord: (row: any) => void
  selectTable: (tableName: string) => void
}

export const useDataView = (
  props: DataViewProps,
  emit: EmitLike,
  message: MessageLike
) => {
  const loadingTables = ref(false)

  // Inline editing state
  const editingCell = ref<{ rowIndex: number; columnKey: string } | null>(null)
  const editingValue = ref('')
  const editingIsNull = ref(false)
  const editingConditions = ref<Record<string, any>>({})
  const editingOriginalValue = ref<any>(null)

  const isNullish = (v: any) => v === null || v === undefined

  const handleCellDblClick = (row: any, rowIndex: number, columnKey: string) => {
    editingConditions.value = { ...row }
    editingOriginalValue.value = row[columnKey]
    editingIsNull.value = isNullish(row[columnKey])
    editingValue.value = editingIsNull.value ? '' : String(row[columnKey])
    editingCell.value = { rowIndex, columnKey }
  }

  // 编辑框输入时更新值，并取消 NULL 标记
  const onEditingInput = (v: string) => {
    editingValue.value = v
    editingIsNull.value = false
  }

  // 点击「NULL」把当前单元格置空为 NULL
  const setEditingNull = () => {
    editingIsNull.value = true
    editingValue.value = ''
  }

  const handleCellEditSave = async () => {
    if (!editingCell.value) return
    const { rowIndex, columnKey } = editingCell.value

    const newVal = editingIsNull.value ? null : editingValue.value
    // 无变化则不提交（同时避免双击后直接失焦把 NULL 误写成空串）
    const origIsNull = isNullish(editingOriginalValue.value)
    const unchanged = editingIsNull.value
      ? origIsNull
      : !origIsNull && editingValue.value === String(editingOriginalValue.value)
    if (unchanged) {
      editingCell.value = null
      return
    }

    try {
      const res = await updateDBManagerRecordAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        data: { [columnKey]: newVal },
        conditions: { ...editingConditions.value }
      })
      if (res.code === 0) {
        message.success('保存成功')
        if (tableData.value[rowIndex]) {
          tableData.value[rowIndex][columnKey] = newVal
        }
        editingCell.value = null
      } else {
        message.error(res.message || '保存失败')
        revertCellEdit(rowIndex, columnKey)
      }
    } catch (error) {
      message.error('保存失败')
      revertCellEdit(rowIndex, columnKey)
    }
  }

  const revertCellEdit = (rowIndex: number, columnKey: string) => {
    if (tableData.value[rowIndex]) {
      tableData.value[rowIndex][columnKey] = editingOriginalValue.value
    }
    editingCell.value = null
  }

  const handleCellEditCancel = () => {
    if (editingCell.value) {
      revertCellEdit(editingCell.value.rowIndex, editingCell.value.columnKey)
    }
  }

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

  // Batch selection state
  const checkedRowKeys = ref<number[]>([])
  const rowKey = (row: any, index: number) => index

  const downloadFile = (content: string, filename: string) => {
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleExportCSV = async () => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    try {
      const res = await exportDBManagerTableAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format: 'csv'
      })
      const data = (res as any).data || res
      downloadFile(data, `${props.selectedDatabase}_${props.selectedTable}.csv`)
      message.success('CSV 已导出')
    } catch (error) {
      message.error('导出 CSV 失败')
    }
  }

  const handleExportSQL = async () => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    try {
      const res = await exportDBManagerTableAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format: 'sql'
      })
      const data = (res as any).data || res
      downloadFile(data, `${props.selectedDatabase}_${props.selectedTable}.sql`)
      message.success('SQL 已导出')
    } catch (error) {
      message.error('导出 SQL 失败')
    }
  }

  // ═══════════════ 导出选项 Modal ═══════════════
  const showExportModal = ref(false)
  const exportFormat = ref('csv')
  const exportColumns = ref<string[]>([])
  const exportAllColumns = ref(true)
  const exportColumnOptions = ref<{ label: string; value: string }[]>([])
  const exportWhere = ref('')
  const exportIncludeDropTable = ref(false)
  const exportIncludeCreateTable = ref(false)
  const exporting = ref(false)

  const handleOpenExportModal = async (format: string) => {
    exportFormat.value = format
    exportWhere.value = ''
    exportAllColumns.value = true
    exportColumns.value = []
    exportIncludeDropTable.value = false
    exportIncludeCreateTable.value = false
    showExportModal.value = true

    // 获取列列表
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return

    const server = props.serverOptions.find((s: any) => s.value === props.selectedServerId)
    const isPg = server?.type === 'postgresql'
    const q = isPg ? '"' : '`'

    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        sql: `SELECT * FROM ${q}${props.selectedTable}${q} LIMIT 0`
      })
      if (res.code === 0 && res.data && res.data.type === 'query' && res.data.columns) {
        exportColumnOptions.value = res.data.columns.map((c: string) => ({ label: c, value: c }))
      }
    } catch {
      exportColumnOptions.value = []
    }
  }

  const handleExportWithOptions = async () => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    exporting.value = true
    try {
      const res = await exportDBManagerTableAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format: exportFormat.value,
        columns: exportAllColumns.value ? undefined : exportColumns.value,
        where: exportWhere.value || undefined,
        includeDropTable: exportFormat.value === 'sql' ? exportIncludeDropTable.value : undefined,
        includeCreateTable: exportFormat.value === 'sql' ? exportIncludeCreateTable.value : undefined
      })
      const data = (res as any).data || res
      const ext = exportFormat.value
      downloadFile(data, `${props.selectedDatabase}_${props.selectedTable}.${ext}`)
      message.success('导出成功')
      showExportModal.value = false
    } catch (error) {
      message.error('导出失败')
    } finally {
      exporting.value = false
    }
  }

  const handleBatchDelete = async () => {
    if (checkedRowKeys.value.length === 0) return
    if (!confirm(`确定要删除选中的 ${checkedRowKeys.value.length} 条记录吗？`)) return

    const selectedRows = checkedRowKeys.value.map(idx => tableData.value[idx]).filter(Boolean)
    let successCount = 0
    let failCount = 0

    for (const row of selectedRows) {
      try {
        const res = await deleteDBManagerRecordAPI({
          serverId: props.selectedServerId,
          databaseName: props.selectedDatabase,
          tableName: props.selectedTable,
          conditions: row
        })
        if (res.code === 0) {
          successCount++
        } else {
          failCount++
        }
      } catch {
        failCount++
      }
    }

    if (successCount > 0) {
      message.success(`成功删除 ${successCount} 条记录${failCount > 0 ? `，${failCount} 条失败` : ''}`)
    } else {
      message.error('删除失败')
    }

    checkedRowKeys.value = []
    fetchTableData()
  }

  const tableData = ref<any[]>([])
  const tableColumns = ref<any[]>([])

  // 列宽持久化存储，key 为表名
  const colWidths = ref<Record<string, Record<string, number>>>({})
  const getStoredWidth = (tableName: string, col: string): number | undefined => {
    return colWidths.value[tableName]?.[col]
  }
  const saveColWidth = (tableName: string, col: string, width: number) => {
    if (!colWidths.value[tableName]) {
      colWidths.value[tableName] = {}
    }
    colWidths.value[tableName][col] = width
    try {
      localStorage.setItem('db_manager_col_widths', JSON.stringify(colWidths.value))
    } catch { /* quota exceeded */ }
  }
  // 初始化时从 localStorage 恢复
  try {
    const saved = localStorage.getItem('db_manager_col_widths')
    if (saved) colWidths.value = JSON.parse(saved)
  } catch { /* ignore */ }
  const pagination = ref({
    page: 1,
    limit: 20,
    itemCount: 0
  })

  const searchColumn = ref<string | null>(null)
  const searchValue = ref('')
  const searchOptions = ref<{ label: string; value: string }[]>([])
  const advancedSearch = ref<any[]>([])

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

  const selectedServerLabel = computed(() => {
    return props.selectedServerId
      ? props.serverOptions.find(s => s.value === props.selectedServerId)?.label || ''
      : ''
  })

  const tableListSummary = computed(() => {
    return {
      total: tableListPagination.value.itemCount,
      currentPage: tableListPagination.value.page,
      keyword: tableKeyword.value
    }
  })

  const recordSummary = computed(() => {
    return {
      total: pagination.value.itemCount,
      currentPage: pagination.value.page,
      pageSize: pagination.value.limit,
      columnCount: searchOptions.value.length,
      hasFilters: Boolean(searchColumn.value || searchValue.value || advancedSearch.value.length)
    }
  })

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
          onClick: () => emit.selectTable(row.name)
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

  // 行级详情
  const showDetailModal = ref(false)
  const detailRow = ref<Record<string, any>>({})
  const detailColumns = ref<{ key: string; label: string }[]>([])

  const openDetail = (row: any, orderColumns: string[]) => {
    detailRow.value = row
    detailColumns.value = orderColumns.map((col: string) => ({
      key: col,
      label: col
    }))
    showDetailModal.value = true
  }

  const buildRowColumns = (cols: string[]) => {
    const visibleCols = cols.filter((col: string) => col !== '__rowid__')
    searchOptions.value = visibleCols.map((col: string) => ({ label: col, value: col }))

    const actionCol = {
      title: '操作',
      key: 'actions',
      fixed: 'left',
      width: 200,
      render(row: any) {
        return h('div', { class: 'flex gap-2' }, [
          h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => emit.editRecord(row) }, { default: () => '编辑' }),
          h(NButton, { size: 'tiny', ghost: true, onClick: () => openDetail(row, visibleCols) }, { default: () => '详情' }),
          h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => emit.copyRecord(row) }, { default: () => '复制' }),
          h(NPopconfirm, { onPositiveClick: () => deleteRecord(row) }, {
            trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
            default: () => '确定要删除这条记录吗？'
          })
        ])
      }
    }

    const isEditingThisCell = (rowIndex: number, key: string) => {
      return editingCell.value?.rowIndex === rowIndex && editingCell.value?.columnKey === key
    }

    const selectionCol = {
      title: '',
      key: '__selection',
      width: 40,
      render(row: any, rowIndex: number) {
        const isChecked = checkedRowKeys.value.includes(rowIndex)
        return h('input', {
          type: 'checkbox',
          checked: isChecked,
          onChange: (e: Event) => {
            const target = e.target as HTMLInputElement
            if (target.checked) {
              checkedRowKeys.value = [...checkedRowKeys.value, rowIndex]
            } else {
              checkedRowKeys.value = checkedRowKeys.value.filter(k => k !== rowIndex)
            }
          }
        })
      }
    }

    tableColumns.value = [
      selectionCol,
      actionCol,
      ...visibleCols.map((col: string, index: number) => ({
        title: col,
        key: col,
        width: getStoredWidth(props.selectedTable ?? '', col) || 150,
        resizable: true,
        ellipsis: { tooltip: true as const },
        className: index === 0 ? 'db-primary-col' : undefined,
        onColumnResize(width: number) {
          if (props.selectedTable) {
            saveColWidth(props.selectedTable, col, width)
          }
        },
        render(row: any, rowIndex: number) {
          if (isEditingThisCell(rowIndex, col)) {
            return h('div', { class: 'db-inline-edit-cell flex items-center gap-1' }, [
              h(NInput, {
                value: editingValue.value,
                placeholder: editingIsNull.value ? 'NULL' : '',
                'onUpdate:value': onEditingInput,
                size: 'small',
                autofocus: true,
                onKeyup: (e: KeyboardEvent) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    handleCellEditSave()
                  }
                  if (e.key === 'Escape') {
                    e.preventDefault()
                    handleCellEditCancel()
                  }
                },
                onBlur: () => {
                  // Guard: only save if still in editing mode (Enter may have cleared it)
                  setTimeout(() => {
                    if (editingCell.value) handleCellEditSave()
                  }, 100)
                }
              }),
              // 置为 NULL：用 mousedown+preventDefault 避免先触发输入框失焦保存
              h(NButton, {
                size: 'tiny',
                quaternary: true,
                type: editingIsNull.value ? 'primary' : 'default',
                title: '设为 NULL',
                onMousedown: (e: MouseEvent) => { e.preventDefault(); setEditingNull() }
              }, { default: () => 'NULL' })
            ])
          }
          const raw = row[col]
          if (isNullish(raw)) {
            // NULL 显示为灰色斜体，和空串区分开
            return h('span', {
              class: 'db-cell-inline db-cell-null',
              onDblclick: (e: MouseEvent) => { e.stopPropagation(); handleCellDblClick(row, rowIndex, col) },
              style: { cursor: 'pointer', minHeight: '22px', display: 'inline-block', width: '100%', color: '#bbb', fontStyle: 'italic' }
            }, 'NULL')
          }
          return h('span', {
            class: 'db-cell-inline',
            title: '双击编辑',
            onDblclick: (e: MouseEvent) => {
              e.stopPropagation()
              handleCellDblClick(row, rowIndex, col)
            },
            style: { cursor: 'pointer', minHeight: '22px', display: 'inline-block', width: '100%' }
          }, raw === '' ? '' : formatCell(raw))
        }
      }))
    ]
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
          buildRowColumns(res.data.columns as string[])
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

  onMounted(() => {
    if (props.selectedTable) {
      fetchTableData()
    }
  })

  return {
    advancedSearch,
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
    detailColumns,
    detailRow,
    handleExportCSV,
    handleExportSQL,
    handleExportWithOptions,
    handleOpenExportModal,
    handleReset,
    openDetail,
    showDetailModal,
    exportAllColumns,
    exportColumnOptions,
    exportColumns,
    exportFormat,
    exportIncludeCreateTable,
    exportIncludeDropTable,
    exportWhere,
    exporting,
    showExportModal,
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
  }
}
