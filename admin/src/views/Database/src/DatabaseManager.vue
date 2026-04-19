<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useMessage, NSelect, NSpin, NEmpty, NMenu, NTabs, NTab, NIcon } from 'naive-ui'
import { databaseServerListAPI, databaseListAPI, getDBManagerTablesAPI, execDBManagerSqlAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

import DataView from './components/DataView.vue'
import StructureView from './components/StructureView.vue'
import InsertView from './components/InsertView.vue'
import SqlView from './components/SqlView.vue'
import SearchView from './components/SearchView.vue'
import OperationsView from './components/OperationsView.vue'

const props = defineProps<{
  defaultServerId?: number | null
  defaultDatabaseName?: string | null
}>()

const message = useMessage()

// 状态
const selectedServerId = ref<number | null>(null)
const selectedDatabase = ref<string | null>(null)
const selectedTable = ref<string | null>(null)
const activeTab = ref('data')

const serverOptions = ref<{label: string, value: number, type: string}[]>([])
const databaseOptions = ref<{label: string, value: string}[]>([])
const tables = ref<string[]>([])
const loadingTables = ref(false)

const structureData = ref<any[]>([])
const loadingStructure = ref(false)

const isEditing = ref(false)
const recordData = ref<Record<string, any>>({})
const originalRecordData = ref<Record<string, any>>({})

const dataViewRef = ref()

// 菜单数据
const menuOptions = computed(() => {
  return tables.value.map(t => ({
    label: t,
    key: t,
    icon: renderIcon('mdi:table')
  }))
})

// 初始化
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

// 当 props 变化时（比如在 Modal 中打开不同的数据库）
watch(() => [props.defaultServerId, props.defaultDatabaseName], async ([newServerId, newDbName]) => {
  if (newServerId) {
    selectedServerId.value = newServerId as number
    await onServerChange(newServerId as number, true)
    if (newDbName) {
      selectedDatabase.value = newDbName as string
      await onDatabaseChange(newDbName as string)
    }
  }
})

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
    message.error("获取服务器列表失败")
  }
}

const onServerChange = async (val: number, skipClear = false) => {
  if (!skipClear) {
    selectedDatabase.value = null
    selectedTable.value = null
    tables.value = []
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
    message.error("获取数据库列表失败")
  }
}

const onDatabaseChange = async (val: string) => {
  selectedTable.value = null
  if (!selectedServerId.value || !val) return
  
  loadingTables.value = true
  try {
    const res = await getDBManagerTablesAPI({
      serverId: selectedServerId.value,
      databaseName: val
    })
    if (res.code === 0) {
      tables.value = res.data || []
    }
  } catch (error) {
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
  } else if (server.type === 'sqlite'){
    sql = `SELECT name AS "Field", type AS "Type", "notnull" AS "Null", dflt_value AS "Default", CASE WHEN pk > 0 THEN 'PRI' ELSE '' END AS "Key" FROM pragma_table_info('${selectedTable.value}')`
  } else {
    sql = `SELECT column_name as "Field", data_type as "Type", is_nullable as "Null", column_default as "Default" FROM information_schema.columns WHERE table_name = '${selectedTable.value}'`
  }

  try {
    const res = await execDBManagerSqlAPI({
      serverId: selectedServerId.value,
      databaseName: selectedDatabase.value,
      sql: sql
    })
    
    if (res.code === 0 && res.data && res.data.type === 'query') {
      structureData.value = res.data.rows || []
    }
  } catch (error) {
  } finally {
    loadingStructure.value = false
  }
}

const onTableSelect = (key: string) => {
  selectedTable.value = key
  if (activeTab.value !== 'data' && activeTab.value !== 'structure') {
    activeTab.value = 'data'
  }
  if (activeTab.value === 'structure') {
    fetchTableStructure()
  }
}

const onTabChange = (tab: string) => {
  activeTab.value = tab
  if ((tab === 'structure' || tab === 'search') && selectedTable.value) {
    if (structureData.value.length === 0) fetchTableStructure()
  } else if (tab === 'insert' && selectedTable.value) {
    isEditing.value = false
    recordData.value = {}
    if (structureData.value.length === 0) {
      fetchTableStructure()
    }
  }
}

// 搜索回调
const handleAdvancedSearch = (conditions: any[]) => {
  activeTab.value = 'data'
  if (dataViewRef.value) {
    dataViewRef.value.setAdvancedSearch(conditions)
  }
}

// 操作回调
const handleTableDropped = () => {
  selectedTable.value = null
  activeTab.value = 'data'
  onDatabaseChange(selectedDatabase.value as string)
}

const handleTableRenamed = (newName: string) => {
  selectedTable.value = newName
  activeTab.value = 'data'
  onDatabaseChange(selectedDatabase.value as string)
}

const handleTableTruncated = () => {
  activeTab.value = 'data'
  if (dataViewRef.value) {
    dataViewRef.value.fetchTableData()
  }
}

// 记录操作回调
const handleEditRecord = async (row: any) => {
  if (structureData.value.length === 0) await fetchTableStructure()
  isEditing.value = true
  recordData.value = { ...row }
  originalRecordData.value = { ...row }
  activeTab.value = 'insert'
}

const handleCopyRecord = async (row: any) => {
  if (structureData.value.length === 0) await fetchTableStructure()
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
</script>

<template>
  <div class="db-manager flex h-[650px] gap-2 p-2 bg-[#f0f0f0] rounded-lg">
    <!-- 左侧树形结构 (数据库/表) -->
    <div class="w-64 border border-slate-300 bg-[#f8f9fa] flex flex-col overflow-hidden text-sm">
      <div class="p-2 border-b border-slate-300 bg-[#e5e5e5] flex flex-col gap-2">
        <div class="flex items-center gap-1 font-semibold text-slate-700">
          <n-icon :component="renderIcon('mdi:server')" />
          <span>{{ $t('database.server') }}</span>
        </div>
        <n-select
          v-model:value="selectedServerId"
          :options="serverOptions"
          size="small"
          @update:value="onServerChange"
        />
        <div class="flex items-center gap-1 font-semibold text-slate-700 mt-1">
          <n-icon :component="renderIcon('mdi:database')" />
          <span>{{ $t('database.database') }}</span>
        </div>
        <n-select
          v-model:value="selectedDatabase"
          :options="databaseOptions"
          size="small"
          :disabled="!selectedServerId"
          @update:value="onDatabaseChange"
        />
      </div>
      <div class="flex-1 overflow-y-auto p-1 bg-white">
        <div
          v-if="loadingTables"
          class="flex justify-center p-4"
        >
          <n-spin size="small" />
        </div>
        <n-empty
          v-else-if="tables.length === 0"
          :description="$t('database.noTable')"
          class="mt-10"
        />
        <n-menu
          v-else
          :options="menuOptions"
          :value="selectedTable"
          @update:value="onTableSelect"
          :root-indent="12"
          :indent="12"
          class="text-xs"
        />
      </div>
    </div>

    <!-- 右侧内容区 -->
    <div class="flex-1 border border-slate-200 bg-white flex flex-col overflow-hidden shadow-sm text-sm">
      <!-- 顶部标签页：模拟 phpMyAdmin 的样式 -->
      <div class="border-b border-slate-200 bg-[#f5f5f5] pt-1 px-2">
        <n-tabs
          v-model:value="activeTab"
          type="card"
          size="small"
          class="phpmyadmin-tabs"
          @update:value="onTabChange"
        >
          <n-tab name="data">
            <template #default>
              <div class="flex items-center gap-1"><n-icon :component="renderIcon('mdi:table-search')" /><span>{{ $t('database.preview') }}</span></div>
            </template>
          </n-tab>
          <n-tab name="structure">
            <template #default>
              <div class="flex items-center gap-1"><n-icon :component="renderIcon('mdi:table-cog')" /><span>{{ $t('database.structure') }}</span></div>
            </template>
          </n-tab>
          <n-tab name="sql">
            <template #default>
              <div class="flex items-center gap-1"><n-icon :component="renderIcon('mdi:console')" /><span>SQL</span></div>
            </template>
          </n-tab>
          <n-tab name="search">
            <template #default>
              <div class="flex items-center gap-1"><n-icon :component="renderIcon('mdi:magnify')" /><span>{{ $t('database.search') }}</span></div>
            </template>
          </n-tab>
          <n-tab name="insert">
            <template #default>
              <div class="flex items-center gap-1"><n-icon :component="renderIcon('mdi:plus-box-outline')" /><span>{{ $t('database.insert') }}</span></div>
            </template>
          </n-tab>
          <n-tab name="operations">
            <template #default>
              <div class="flex items-center gap-1"><n-icon :component="renderIcon('mdi:cog-outline')" /><span>{{ $t('database.action') }}</span></div>
            </template>
          </n-tab>
        </n-tabs>
      </div>

      <!-- 子视图组件 -->
      <DataView
        v-show="activeTab === 'data'"
        ref="dataViewRef"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        @editRecord="handleEditRecord"
        @copyRecord="handleCopyRecord"
      />
      <StructureView
        v-if="activeTab === 'structure'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        :structureData="structureData"
        :loadingStructure="loadingStructure"
        @refresh="fetchTableStructure"
      />
      <InsertView
        v-if="activeTab === 'insert'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        :structureData="structureData"
        :loadingStructure="loadingStructure"
        :isEditing="isEditing"
        :recordData="recordData"
        :originalRecordData="originalRecordData"
        @cancel="activeTab = 'data'"
        @success="handleInsertSuccess"
      />
      <SqlView
        v-if="activeTab === 'sql'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :serverOptions="serverOptions"
      />
      <SearchView
        v-if="activeTab === 'search'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        :structureData="structureData"
        :loadingStructure="loadingStructure"
        @search="handleAdvancedSearch"
        @cancel="activeTab = 'data'"
      />
      <OperationsView
        v-if="activeTab === 'operations'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        @tableDropped="handleTableDropped"
        @tableRenamed="handleTableRenamed"
        @tableTruncated="handleTableTruncated"
      />
    </div>
  </div>
</template>

<style scoped>
.phpmyadmin-tabs {
  --n-tab-padding: 4px 12px !important;
  --n-tab-font-size: 12px !important;
  --n-tab-border-radius: 4px 4px 0 0 !important;
  --n-tab-text-color: #333 !important;
  --n-tab-text-color-active: #000 !important;
  --n-pane-padding: 0 !important;
}

:deep(.n-tabs-nav) {
  border-bottom: none !important;
}

:deep(.n-tabs-tab) {
  background-color: #e9ecef !important;
  border: 1px solid #ced4da !important;
  border-bottom: none !important;
  margin-right: 4px !important;
}

:deep(.n-tabs-tab--active) {
  background-color: #ffffff !important;
  border-color: #ced4da !important;
  border-bottom: 1px solid #ffffff !important;
  margin-bottom: -1px !important;
  font-weight: bold;
}
</style>