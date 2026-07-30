import { computed, h, onMounted, ref, watch } from 'vue'
import { NButton, NPopconfirm, NInput } from 'naive-ui'
import { getDBManagerTableDataAPI, deleteDBManagerRecordAPI, updateDBManagerRecordAPI } from '@/api/modules/database'
import type { DataTableSorter, DataViewEmit, DataViewProps, MessageLike } from './dataViewTypes'
import { useDataViewTableList } from './useDataViewTableList'
import { useDataViewTransfer } from './useDataViewTransfer'
import { useDataViewColumnWidths } from './useDataViewColumnWidths'

export const useDataView = (
  props: DataViewProps,
  emit: DataViewEmit,
  message: MessageLike
) => {
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

  const onEditingInput = (v: string) => {
    editingValue.value = v
    editingIsNull.value = false
  }

  const setEditingNull = () => {
    editingIsNull.value = true
    editingValue.value = ''
  }

  const handleCellEditSave = async () => {
    if (!editingCell.value) return
    const { rowIndex, columnKey } = editingCell.value

    const newVal = editingIsNull.value ? null : editingValue.value
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

  const loadingData = ref(false)
  const {
    applyTableSearch, fetchTableList, handleTableListPageChange, handleTableListPageSizeChange,
    handleTableListSorterChange, loadingTables, resetTableListState, resetTableSearch,
    selectedServerLabel, tableKeyword, tableList, tableListColumns, tableListPagination,
    tableListSummary, tableSearchInput
  } = useDataViewTableList(props, emit, message)

  const {
    exportAllColumns, exportColumnOptions, exportColumns, exportFormat, exportIncludeCreateTable,
    exportIncludeDropTable, exporting, exportWhere, handleExportCSV, handleExportSQL,
    handleExportWithOptions, handleOpenExportModal, showExportModal
  } = useDataViewTransfer(props, message)

  const checkedRowKeys = ref<number[]>([])

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

  const { getStoredWidth, saveColumnWidth } = useDataViewColumnWidths()
  const pagination = ref({
    page: 1,
    limit: 20,
    itemCount: 0
  })

  const searchColumn = ref<string | null>(null)
  const searchValue = ref('')
  const searchOptions = ref<{ label: string; value: string }[]>([])
  const advancedSearch = ref<any[]>([])
  const dataSortState = ref<{ columnKey: string | null; order: 'ascend' | 'descend' | false }>({
    columnKey: null,
    order: false
  })

  const formatCell = (value: any) => {
    if (value === null || value === undefined || value === '') return '-'
    return String(value)
  }

  const recordSummary = computed(() => {
    return {
      total: pagination.value.itemCount,
      currentPage: pagination.value.page,
      pageSize: pagination.value.limit,
      columnCount: searchOptions.value.length,
      hasFilters: Boolean(searchColumn.value || searchValue.value || advancedSearch.value.length)
    }
  })

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
    dataSortState.value = { columnKey: null, order: false }
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
        sorter: true,
        sortOrder: () => dataSortState.value.columnKey === col ? dataSortState.value.order : false,
        width: getStoredWidth(props.selectedTable ?? '', col) || 150,
        resizable: true,
        ellipsis: { tooltip: true as const },
        className: index === 0 ? 'db-primary-col' : undefined,
        onColumnResize(width: number) {
          if (props.selectedTable) {
            saveColumnWidth(props.selectedTable, col, width)
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
                  setTimeout(() => {
                    if (editingCell.value) handleCellEditSave()
                  }, 100)
                }
              }),
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
        advancedSearch: advancedSearch.value.length > 0 ? advancedSearch.value : undefined,
        sortField: dataSortState.value.columnKey || undefined,
        sortOrder: dataSortState.value.order || undefined
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

  const handleDataSorterChange = (sorter: DataTableSorter | DataTableSorter[] | null) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : sorter
    dataSortState.value = {
      columnKey: currentSorter?.order ? String(currentSorter.columnKey || '') || null : null,
      order: currentSorter?.order || false
    }
    pagination.value.page = 1
    checkedRowKeys.value = []
    fetchTableData()
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
    handleDataSorterChange,
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
