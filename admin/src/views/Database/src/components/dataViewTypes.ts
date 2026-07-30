export type MessageLike = {
  error: (content: string) => void
  success: (content: string) => void
}

export type DataViewProps = {
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
}

export type DataViewEmit = {
  editRecord: (row: any) => void
  copyRecord: (row: any) => void
  selectTable: (tableName: string) => void
}

export type DataTableSorter = {
  columnKey?: string | number
  order?: 'ascend' | 'descend' | false
}
