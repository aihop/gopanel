import { ref } from 'vue'
import { importDBManagerTableAPI, uploadDBManagerImportAPI } from '@/api/modules/database'
import type { DataViewProps } from './dataViewTypes'

type ImportMessage = {
  error: (content: string) => void
  success: (content: string) => void
  warning: (content: string) => void
}

export const useDataViewImport = (
  props: DataViewProps,
  message: ImportMessage,
  refreshData: () => void
) => {
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

  const handleFileSelected = (event: Event) => {
    const target = event.target as HTMLInputElement
    if (!target.files?.length) return
    const file = target.files[0]
    importFilename.value = file.name
    selectedFile.value = file
    importFormat.value = file.name.endsWith('.sql') ? 'sql' : 'csv'
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
      const request = {
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        format: importFormat.value
      }
      const res = selectedFile.value
        ? await uploadDBManagerImportAPI({ ...request, file: selectedFile.value })
        : await importDBManagerTableAPI({ ...request, content: importContent.value })

      if (res.code !== 0) {
        message.error(res.message || '导入失败')
        return
      }
      message.success(`导入成功，共 ${(res.data as any)?.imported || 0} 条记录`)
      showImportModal.value = false
      importContent.value = ''
      clearFileSelection()
      refreshData()
    } catch (error: any) {
      // 错误提示由请求拦截器统一处理
    } finally {
      importing.value = false
    }
  }

  return {
    clearFileSelection, formatFileSize, handleFileSelected, handleImport, importContent,
    importFilename, importFormat, importing, selectedFile, showImportModal
  }
}
