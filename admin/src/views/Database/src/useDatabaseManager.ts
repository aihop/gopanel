import { ref, onMounted, computed, watch, nextTick, type Ref } from 'vue'
import { databaseServerListAPI, databaseListAPI, getDBManagerTableListAPI, execDBManagerSqlAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

type MessageLike = {
  error: (content: string) => void
}

type ManagerProps = {
  defaultServerId?: number | null
  defaultDatabaseName?: string | null
}

type ServerOption = { label: string; value: number; type: string }
type DatabaseOption = { label: string; value: string }

export const databaseManagerTabLabels: Record<string, string> = {
  data: '浏览',
  structure: '结构',
  sql: 'SQL',
  search: '搜索',
  insert: '插入',
  operations: '操作'
}

export const useDatabaseManager = (
  props: ManagerProps,
  message: MessageLike,
  dataViewRef: Ref<any>
) => {
  const selectedServerId = ref<number | null>(null)
  const selectedDatabase = ref<string | null>(null)
  const selectedTable = ref<string | null>(null)
  const activeTab = ref('data')
  const tableKeywordInput = ref('')
  const tableKeyword = ref('')
  const tableListRows = ref<any[]>([])
  const tableListPagination = ref({
    page: 1,
    limit: 20,
    itemCount: 0
  })
  const tableListSortState = ref<{ columnKey: string | null; order: 'ascend' | 'descend' | false }>({
    columnKey: null,
    order: false
  })

  const serverOptions = ref<ServerOption[]>([])
  const databaseOptions = ref<DatabaseOption[]>([])
  const loadingTables = ref(false)

  const structureData = ref<any[]>([])
  const loadingStructure = ref(false)

  const isEditing = ref(false)
  const recordData = ref<Record<string, any>>({})
  const originalRecordData = ref<Record<string, any>>({})

  const viewContextKey = computed(() => {
    return `${selectedServerId.value || 0}:${selectedDatabase.value || ''}:${selectedTable.value || ''}`
  })

  const selectedServer = computed(() => {
    return serverOptions.value.find(item => item.value === selectedServerId.value) || null
  })

  const selectedServerLabel = computed(() => {
    return selectedServer.value?.label || '未选择服务器'
  })

  const activeTabLabel = computed(() => {
    return databaseManagerTabLabels[activeTab.value] || activeTab.value
  })

  const tables = computed(() => {
    return tableListRows.value.map((item: any) => item?.name).filter(Boolean)
  })

  const filteredTables = computed(() => {
    if (!tableKeywordInput.value.trim()) return tables.value
    const keyword = tableKeywordInput.value.toLowerCase()
    return tables.value.filter(t => t.toLowerCase().includes(keyword))
  })

  const sidebarTablePage = computed({
    get: () => tableListPagination.value.page,
    set: (val) => { tableListPagination.value.page = val }
  })

  const sidebarTablePageSize = computed({
    get: () => tableListPagination.value.limit,
    set: (val) => { tableListPagination.value.limit = val }
  })

  const handleSidebarTablePageChange = (page: number) => {
    handleTableListPageChange(page)
  }

  const handleSidebarTablePageSizeChange = (pageSize: number) => {
    handleTableListPageSizeChange(pageSize)
  }

  const sidebarTableTotal = computed(() => tableListPagination.value.itemCount)

  const menuOptions = computed(() => {
    return tables.value.map(t => ({
      label: t,
      key: t,
      icon: renderIcon('mdi:table')
    }))
  })

  const resetRecordEditorState = () => {
    isEditing.value = false
    recordData.value = {}
    originalRecordData.value = {}
  }

  const resetTableSearch = () => {
    tableKeywordInput.value = ''
    tableKeyword.value = ''
    tableListPagination.value.page = 1
  }

  const fetchTableList = async () => {
    if (!selectedServerId.value || !selectedDatabase.value) {
      tableListRows.value = []
      tableListPagination.value.itemCount = 0
      return
    }

    loadingTables.value = true
    try {
      const res = await getDBManagerTableListAPI({
        serverId: selectedServerId.value,
        databaseName: selectedDatabase.value,
        page: tableListPagination.value.page,
        limit: tableListPagination.value.limit,
        keyword: tableKeyword.value || undefined,
        sortField: tableListSortState.value.columnKey || undefined,
        sortOrder: tableListSortState.value.order || undefined
      })

      if (res.code === 0 && res.data) {
        const items = Array.isArray(res.data.items) ? res.data.items : []
        tableListRows.value = items
        tableListPagination.value.itemCount = Number(res.data.total || 0)
      } else {
        tableListRows.value = []
        tableListPagination.value.itemCount = 0
      }
    } catch (error) {
      tableListRows.value = []
      tableListPagination.value.itemCount = 0
    } finally {
      loadingTables.value = false
    }
  }

  const applyTableSearch = async () => {
    tableKeyword.value = tableKeywordInput.value.trim()
    tableListPagination.value.page = 1
    await fetchTableList()
  }

  const fetchServers = async () => {
    try {
      const res = await databaseServerListAPI({ page: 1, limit: 100 })
      const data = res.data as any
      if (data) {
        const items = Array.isArray(data) ? data : (data.items || [])
        serverOptions.value = items.map((s: any) => ({
          label: `${s.name} (${s.type})`,
          value: s.id,
          type: s.type
        }))
      }
    } catch (error) {
    }
  }

  const onServerChange = async (val: number, skipClear = false) => {
    if (!skipClear) {
      activeTab.value = 'data'
      selectedDatabase.value = null
      selectedTable.value = null
      resetTableSearch()
      tableListRows.value = []
      tableListPagination.value.itemCount = 0
      tableListPagination.value.page = 1
      structureData.value = []
      resetRecordEditorState()
    }
    databaseOptions.value = []

    try {
      const res = await databaseListAPI({ page: 1, limit: 100, wheres: [{ field: 'server_id', rule: 'eq', val: String(val) }] })
      const data = res.data as any
      if (data) {
        const items = Array.isArray(data) ? data : (data.items || [])
        databaseOptions.value = items.map((db: any) => ({
          label: db.name,
          value: db.name
        }))
      }
    } catch (error) {
    }
  }

  const onDatabaseChange = async (val: string) => {
    activeTab.value = 'data'
    selectedTable.value = null
    resetTableSearch()
    tableListPagination.value.page = 1
    tableListPagination.value.itemCount = 0
    structureData.value = []
    resetRecordEditorState()
    if (!selectedServerId.value || !val) return

    await fetchTableList()
  }

  const fetchTableStructure = async () => {
    if (!selectedServerId.value || !selectedDatabase.value || !selectedTable.value) return

    const server = serverOptions.value.find(s => s.value === selectedServerId.value)
    if (!server) return

    loadingStructure.value = true
    let sql = ''
    const escapedStringTable = selectedTable.value.replace(/'/g, "''")
    if (server.type === 'mysql' || server.type === 'mariadb') {
      const escapedIdentifierTable = selectedTable.value.replace(/`/g, '``')
      sql = `SHOW FULL COLUMNS FROM \`${escapedIdentifierTable}\``
    } else if (server.type === 'sqlite') {
      sql = `SELECT name AS "Field", type AS "Type", CASE WHEN "notnull" = 0 THEN 'YES' ELSE 'NO' END AS "Null", dflt_value AS "Default", CASE WHEN pk > 0 THEN 'PRI' ELSE '' END AS "Key" FROM pragma_table_info('${escapedStringTable}')`
    } else {
      sql = `SELECT col.column_name AS "Field", col.data_type AS "Type", col.is_nullable AS "Null", col.column_default AS "Default", CASE WHEN pk.column_name IS NOT NULL THEN 'PRI' ELSE '' END AS "Key", CASE WHEN col.is_identity = 'YES' OR col.column_default LIKE 'nextval(%' THEN 'auto_increment' ELSE '' END AS "Extra" FROM information_schema.columns col LEFT JOIN (SELECT kcu.table_schema, kcu.table_name, kcu.column_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON kcu.constraint_name = tc.constraint_name AND kcu.constraint_schema = tc.constraint_schema WHERE tc.constraint_type = 'PRIMARY KEY') pk ON pk.table_schema = col.table_schema AND pk.table_name = col.table_name AND pk.column_name = col.column_name WHERE col.table_schema = (SELECT n.nspname FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace WHERE c.relname = '${escapedStringTable}' AND c.relkind IN ('r', 'p') AND pg_catalog.pg_table_is_visible(c.oid) LIMIT 1) AND col.table_name = '${escapedStringTable}' ORDER BY col.ordinal_position`
    }

    try {
      const res = await execDBManagerSqlAPI({
        serverId: selectedServerId.value,
        databaseName: selectedDatabase.value,
        sql
      })

      if (res.code === 0 && res.data && res.data.type === 'query') {
        structureData.value = res.data.rows || []
      } else {
        structureData.value = []
      }
    } catch (error) {
      structureData.value = []
    } finally {
      loadingStructure.value = false
    }
  }

  const onTableSelect = (key: string) => {
    selectedTable.value = key
    structureData.value = []
    resetRecordEditorState()

    if (['structure', 'search', 'insert'].includes(activeTab.value)) {
      fetchTableStructure()
    }
  }

  const handleTableListPageChange = (page: number) => {
    tableListPagination.value.page = page
    void fetchTableList()
  }

  const handleTableListPageSizeChange = (pageSize: number) => {
    tableListPagination.value.limit = pageSize
    tableListPagination.value.page = 1
    void fetchTableList()
  }

  const handleTableListSorterChange = (sorter: { columnKey?: string; order?: 'ascend' | 'descend' | false } | { columnKey?: string; order?: 'ascend' | 'descend' | false }[] | null) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : sorter
    tableListSortState.value = {
      columnKey: currentSorter?.order ? (currentSorter.columnKey || null) : null,
      order: currentSorter?.order || false
    }
    tableListPagination.value.page = 1
    void fetchTableList()
  }

  const onTabChange = (tab: string) => {
    activeTab.value = tab
    if (['structure', 'search'].includes(tab) && selectedTable.value) {
      if (structureData.value.length === 0) fetchTableStructure()
    } else if (tab === 'insert' && selectedTable.value) {
      isEditing.value = false
      recordData.value = {}
      if (structureData.value.length === 0) {
        fetchTableStructure()
      }
    }
  }

  const handleAdvancedSearch = (conditions: any[]) => {
    activeTab.value = 'data'
    nextTick(() => {
      if (dataViewRef.value) {
        dataViewRef.value.setAdvancedSearch(conditions)
      }
    })
  }

  const handleTableDropped = () => {
    selectedTable.value = null
    activeTab.value = 'data'
    if (selectedDatabase.value) {
      onDatabaseChange(selectedDatabase.value)
    }
  }

  const handleTableRenamed = (newName: string) => {
    selectedTable.value = newName
    if (selectedDatabase.value) {
      onDatabaseChange(selectedDatabase.value)
    }
  }

  const handleTableTruncated = () => {
    nextTick(() => {
      if (dataViewRef.value) {
        dataViewRef.value.fetchTableData()
      }
    })
  }

  const handleEditRecord = async (row: any) => {
    await fetchTableStructure()
    isEditing.value = true
    recordData.value = { ...row }
    originalRecordData.value = { ...row }
    activeTab.value = 'insert'
  }

  const handleCopyRecord = async (row: any) => {
    await fetchTableStructure()
    isEditing.value = false
    recordData.value = { ...row }
    const pk = structureData.value.find(c => c.Key === 'PRI')
    if (pk) delete recordData.value[pk.Field]
    activeTab.value = 'insert'
  }

  const handleInsertSuccess = () => {
    activeTab.value = 'data'
    nextTick(() => {
      if (dataViewRef.value) {
        dataViewRef.value.fetchTableData()
      }
    })
  }

  const clearSelectedTable = () => {
    selectedTable.value = null
    structureData.value = []
    resetRecordEditorState()
  }

  onMounted(async () => {
    await fetchServers()
    if (props.defaultServerId) {
      selectedServerId.value = props.defaultServerId
      await onServerChange(props.defaultServerId, true)
      if (props.defaultDatabaseName) {
        selectedDatabase.value = props.defaultDatabaseName
        await onDatabaseChange(props.defaultDatabaseName)
      }
    }
  })

  watch(
    () => [props.defaultServerId, props.defaultDatabaseName],
    async ([newServerId, newDbName]) => {
      if (newServerId) {
        selectedServerId.value = newServerId as number
        await onServerChange(newServerId as number, true)
        if (newDbName) {
          selectedDatabase.value = newDbName as string
          await onDatabaseChange(newDbName as string)
        }
      }
    }
  )

  return {
    activeTab,
    activeTabLabel,
    applyTableSearch,
    clearSelectedTable,
    dataViewRef,
    databaseOptions,
    fetchTableStructure,
    filteredTables,
    handleAdvancedSearch,
    handleCopyRecord,
    handleEditRecord,
    handleInsertSuccess,
    fetchTableList,
    handleTableDropped,
    handleTableRenamed,
    handleTableTruncated,
    handleSidebarTablePageChange,
    handleSidebarTablePageSizeChange,
    isEditing,
    loadingStructure,
    loadingTables,
    menuOptions,
    onDatabaseChange,
    onServerChange,
    handleTableListPageChange,
    handleTableListPageSizeChange,
    handleTableListSorterChange,
    onTableSelect,
    onTabChange,
    originalRecordData,
    recordData,
    resetTableSearch,
    selectedDatabase,
    selectedServer,
    selectedServerId,
    selectedServerLabel,
    selectedTable,
    serverOptions,
    sidebarTablePage,
    sidebarTablePageSize,
    sidebarTableTotal,
    structureData,
    tableKeywordInput,
    tables,
    viewContextKey
  }
}
