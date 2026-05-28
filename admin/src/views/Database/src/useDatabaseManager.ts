import { ref, onMounted, computed, watch, type Ref } from 'vue'
import { databaseServerListAPI, databaseListAPI, getDBManagerTablesAPI, execDBManagerSqlAPI } from '@/api/modules/database'
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

  const serverOptions = ref<ServerOption[]>([])
  const databaseOptions = ref<DatabaseOption[]>([])
  const tables = ref<string[]>([])
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

  const filteredTables = computed(() => {
    const keyword = tableKeyword.value.trim().toLowerCase()
    if (!keyword) return tables.value
    return tables.value.filter(t => t.toLowerCase().includes(keyword))
  })

  const menuOptions = computed(() => {
    return filteredTables.value.map(t => ({
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
  }

  const applyTableSearch = () => {
    tableKeyword.value = tableKeywordInput.value.trim()
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
      message.error('获取服务器列表失败')
    }
  }

  const onServerChange = async (val: number, skipClear = false) => {
    if (!skipClear) {
      activeTab.value = 'data'
      selectedDatabase.value = null
      selectedTable.value = null
      resetTableSearch()
      tables.value = []
      structureData.value = []
      resetRecordEditorState()
    }
    databaseOptions.value = []

    try {
      const res = await databaseListAPI({ page: 1, limit: 100, server_id: val })
      const data = res.data as any
      if (data) {
        const items = Array.isArray(data) ? data : (data.items || [])
        databaseOptions.value = items.map((db: any) => ({
          label: db.name,
          value: db.name
        }))
      }
    } catch (error) {
      message.error('获取数据库列表失败')
    }
  }

  const onDatabaseChange = async (val: string) => {
    activeTab.value = 'data'
    selectedTable.value = null
    resetTableSearch()
    structureData.value = []
    resetRecordEditorState()
    if (!selectedServerId.value || !val) return

    loadingTables.value = true
    try {
      const res = await getDBManagerTablesAPI({
        serverId: selectedServerId.value,
        databaseName: val
      })
      if (res.code === 0) {
        tables.value = res.data || []
      } else {
        tables.value = []
      }
    } catch (error) {
      tables.value = []
    } finally {
      loadingTables.value = false
    }
  }

  const fetchTableStructure = async () => {
    if (!selectedServerId.value || !selectedDatabase.value || !selectedTable.value) return

    const server = serverOptions.value.find(s => s.value === selectedServerId.value)
    if (!server) return

    loadingStructure.value = true
    let sql = ''
    if (server.type === 'mysql') {
      sql = `SHOW FULL COLUMNS FROM \`${selectedTable.value}\``
    } else if (server.type === 'sqlite') {
      sql = `SELECT name AS "Field", type AS "Type", "notnull" AS "Null", dflt_value AS "Default", CASE WHEN pk > 0 THEN 'PRI' ELSE '' END AS "Key" FROM pragma_table_info('${selectedTable.value}')`
    } else {
      sql = `SELECT column_name as "Field", data_type as "Type", is_nullable as "Null", column_default as "Default" FROM information_schema.columns WHERE table_name = '${selectedTable.value}'`
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
    if (dataViewRef.value) {
      dataViewRef.value.setAdvancedSearch(conditions)
    }
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
    if (dataViewRef.value) {
      dataViewRef.value.fetchTableData()
    }
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
    if (dataViewRef.value) {
      dataViewRef.value.fetchTableData()
    }
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
    handleTableDropped,
    handleTableRenamed,
    handleTableTruncated,
    isEditing,
    loadingStructure,
    loadingTables,
    menuOptions,
    onDatabaseChange,
    onServerChange,
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
    structureData,
    tableKeywordInput,
    tables,
    viewContextKey
  }
}
