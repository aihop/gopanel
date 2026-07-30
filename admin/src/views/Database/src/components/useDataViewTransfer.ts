import { ref } from 'vue'
import { execDBManagerSqlAPI, exportDBManagerTableAPI } from '@/api/modules/database'
import type { DataViewProps, MessageLike } from './dataViewTypes'

export const useDataViewTransfer = (props: DataViewProps, message: MessageLike) => {
  const showExportModal = ref(false)
  const exportFormat = ref('csv')
  const exportColumns = ref<string[]>([])
  const exportAllColumns = ref(true)
  const exportColumnOptions = ref<{ label: string; value: string }[]>([])
  const exportWhere = ref('')
  const exportIncludeDropTable = ref(false)
  const exportIncludeCreateTable = ref(false)
  const exporting = ref(false)

  const downloadFile = (content: string, filename: string) => {
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
  }

  const exportTable = async (format: string) => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    try {
      const res = await exportDBManagerTableAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format
      })
      downloadFile((res as any).data || res, `${props.selectedDatabase}_${props.selectedTable}.${format}`)
      message.success(`${format.toUpperCase()} 已导出`)
    } catch {
      message.error(`导出 ${format.toUpperCase()} 失败`)
    }
  }

  const handleOpenExportModal = async (format: string) => {
    exportFormat.value = format
    exportWhere.value = ''
    exportAllColumns.value = true
    exportColumns.value = []
    exportIncludeDropTable.value = false
    exportIncludeCreateTable.value = false
    showExportModal.value = true
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return

    const server = props.serverOptions.find(serverOption => serverOption.value === props.selectedServerId)
    const quote = server?.type === 'postgresql' ? '"' : '`'
    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        sql: `SELECT * FROM ${quote}${props.selectedTable}${quote} LIMIT 0`
      })
      if (res.code === 0 && res.data?.type === 'query' && res.data.columns) {
        exportColumnOptions.value = res.data.columns.map((column: string) => ({ label: column, value: column }))
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
      downloadFile((res as any).data || res, `${props.selectedDatabase}_${props.selectedTable}.${exportFormat.value}`)
      message.success('导出成功')
      showExportModal.value = false
    } catch {
      message.error('导出失败')
    } finally {
      exporting.value = false
    }
  }

  return {
    exportAllColumns, exportColumnOptions, exportColumns, exportFormat, exportIncludeCreateTable,
    exportIncludeDropTable, exporting, exportWhere, handleExportCSV: () => exportTable('csv'),
    handleExportSQL: () => exportTable('sql'), handleExportWithOptions, handleOpenExportModal,
    showExportModal
  }
}
