import { ref } from 'vue'

export const useDataViewColumnWidths = () => {
  const columnWidths = ref<Record<string, Record<string, number>>>({})
  try {
    const saved = localStorage.getItem('db_manager_col_widths')
    if (saved) columnWidths.value = JSON.parse(saved)
  } catch { /* ignore */ }

  const getStoredWidth = (tableName: string, column: string) => columnWidths.value[tableName]?.[column]
  const saveColumnWidth = (tableName: string, column: string, width: number) => {
    if (!columnWidths.value[tableName]) columnWidths.value[tableName] = {}
    columnWidths.value[tableName][column] = width
    try {
      localStorage.setItem('db_manager_col_widths', JSON.stringify(columnWidths.value))
    } catch { /* quota exceeded */ }
  }

  return { getStoredWidth, saveColumnWidth }
}
