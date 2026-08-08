<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage, NTabs, NTab, NIcon } from 'naive-ui'
import { renderIcon } from '@/utils'

import DataView from './components/DataView.vue'
import DatabaseManagerContextBar from './components/DatabaseManagerContextBar.vue'
import DatabaseManagerSidebar from './components/DatabaseManagerSidebar.vue'
import DatabaseTablesView from './components/DatabaseTablesView.vue'
import StructureView from './components/StructureView.vue'
import InsertView from './components/InsertView.vue'
import SqlView from './components/SqlView.vue'
import SearchView from './components/SearchView.vue'
import OperationsView from './components/OperationsView.vue'
import CreateDatabaseModal from './components/CreateDatabaseModal.vue'
import CreateTableModal from './components/CreateTableModal.vue'
import { useDatabaseManager } from './useDatabaseManager'

const props = defineProps<{
  defaultServerId?: number | null
  defaultDatabaseName?: string | null
  fillHeight?: boolean
}>()

const message = useMessage()
const dataViewRef = ref()
const databaseSqlMode = ref(false)
const {
  activeTab,
  activeTabLabel,
  applyTableSearch,
  clearSelectedTable,
  databaseOptions,
  fetchTableList,
  fetchTableStructure,
  handleAdvancedSearch,
  handleCopyRecord,
  handleEditRecord,
  handleInsertSuccess,
  handleSidebarTablePageChange,
  handleSidebarTablePageSizeChange,
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
  selectedServerLabel,
  selectedServerId,
  selectedTable,
  serverOptions,
  sidebarTablePage,
  sidebarTablePageSize,
  sidebarTableTotal,
  structureData,
  tableKeywordInput,
  tables,
  viewContextKey
} = useDatabaseManager(props, message, dataViewRef)

const handleServerModelUpdate = (value: number | null) => {
  databaseSqlMode.value = false
  selectedServerId.value = value
  if (value) {
    onServerChange(value)
  }
}

const handleDatabaseModelUpdate = (value: string | null) => {
  databaseSqlMode.value = false
  selectedDatabase.value = value
  if (value) {
    onDatabaseChange(value)
  }
}

const handleTableKeywordUpdate = (value: string) => {
  tableKeywordInput.value = value
}

const handleTableSelect = (tableName: string) => {
  databaseSqlMode.value = false
  onTableSelect(tableName)
}

const openDatabaseSql = () => {
  selectedTable.value = null
  databaseSqlMode.value = true
}

const backToDatabaseObjects = () => {
  databaseSqlMode.value = false
  clearSelectedTable()
}

// 创建数据库/表模态框状态
const showCreateDatabaseModal = ref(false)
const showCreateTableModal = ref(false)

const selectedServerType = computed(() => {
  const server = serverOptions.value.find(s => s.value === selectedServerId.value)
  return server?.type || ''
})

const handleCreateDatabaseSuccess = async () => {
  // 刷新数据库列表
  if (selectedServerId.value) {
    await onServerChange(selectedServerId.value, true)
  }
}

const handleCreateTableSuccess = async () => {
  showCreateTableModal.value = false
  // 刷新表列表
  await fetchTableList()
}

const handleDropDatabase = async () => {
  if (!selectedDatabase.value) return
  const name = selectedDatabase.value
  const confirmed = window.confirm(`确定要删除数据库「${name}」吗？\n此操作不可恢复，数据库中的所有表和数据将被永久删除！`)
  if (!confirmed) return

  try {
    const { dropDBManagerDatabaseAPI } = await import('@/api/modules/database')
    const res = await dropDBManagerDatabaseAPI({
      serverId: selectedServerId.value!,
      databaseName: name,
    })
    if (res.code === 0) {
      message.success(`数据库 ${name} 已删除`)
      selectedDatabase.value = null
      if (selectedServerId.value) {
        await onServerChange(selectedServerId.value, true)
      }
    } else {
      message.error(res.message || '删除失败')
    }
  } catch {
    return
  }
}
</script>

<template>
  <div
    class="db-manager flex gap-3 p-2 bg-[#eef2f7] rounded-lg"
    :class="fillHeight ? 'h-full min-h-0' : 'h-[650px]'"
  >
    <DatabaseManagerSidebar
      :selected-server-id="selectedServerId"
      :selected-database="selectedDatabase"
      :selected-table="selectedTable"
      :server-options="serverOptions"
      :database-options="databaseOptions"
      :table-keyword-input="tableKeywordInput"
      :loading-tables="loadingTables"
      :tables="tables"
      :filtered-table-count="sidebarTableTotal"
      :sidebar-table-page="sidebarTablePage"
      :sidebar-table-page-size="sidebarTablePageSize"
      :menu-options="menuOptions"
      @update:selected-server-id="handleServerModelUpdate"
      @update:selected-database="handleDatabaseModelUpdate"
      @update:table-keyword-input="handleTableKeywordUpdate"
      @update:sidebar-table-page="handleSidebarTablePageChange"
      @update:sidebar-table-page-size="handleSidebarTablePageSizeChange"
      @select-table="handleTableSelect"
      @search-tables="applyTableSearch"
      @reset-table-search="resetTableSearch"
      @create-database="showCreateDatabaseModal = true"
      @create-table="showCreateTableModal = true"
      @drop-database="handleDropDatabase"
    />

    <div class="flex-1 border border-slate-200 bg-white flex flex-col overflow-hidden shadow-sm text-sm rounded-md">
      <DatabaseManagerContextBar
        :selected-server-label="selectedServerLabel"
        :selected-database="selectedDatabase"
        :selected-table="selectedTable"
        :active-tab-label="activeTabLabel"
        :table-count="sidebarTableTotal"
        :database-sql-mode="databaseSqlMode"
        @back-to-tables="backToDatabaseObjects"
      />

      <!-- 未选中表时隐藏整排标签：这些标签都是针对具体表的操作，此处仅显示数据库级表概览 -->
      <div v-if="selectedTable" class="border-b border-slate-200 bg-[#f5f5f5] pt-1 px-2">
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

      <!-- 未选择表时显示表概览（数据库级别） -->
      <DatabaseTablesView
        v-if="selectedDatabase && !selectedTable && !databaseSqlMode"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :serverOptions="serverOptions"
        @select-table="handleTableSelect"
        @open-sql="openDatabaseSql"
      />

      <SqlView
        :key="`database-sql:${viewContextKey}`"
        v-if="selectedDatabase && !selectedTable && databaseSqlMode"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="null"
        :serverOptions="serverOptions"
      />

      <!-- 选择表后显示子视图组件 -->
      <DataView
        :key="`data:${viewContextKey}`"
        v-if="selectedTable && activeTab === 'data'"
        ref="dataViewRef"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        @select-table="handleTableSelect"
        @edit-record="handleEditRecord"
        @copy-record="handleCopyRecord"
      />
      <StructureView
        :key="`structure:${viewContextKey}`"
        v-if="selectedTable && activeTab === 'structure'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        :structureData="structureData"
        :loadingStructure="loadingStructure"
        @refresh="fetchTableStructure"
      />
      <InsertView
        :key="`insert:${viewContextKey}:${isEditing ? 'edit' : 'create'}`"
        v-if="selectedTable && activeTab === 'insert'"
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
        :key="`sql:${viewContextKey}`"
        v-if="selectedTable && activeTab === 'sql'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
      />
      <SearchView
        :key="`search:${viewContextKey}`"
        v-if="selectedTable && activeTab === 'search'"
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
        :key="`operations:${viewContextKey}`"
        v-if="selectedTable && activeTab === 'operations'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        @tableDropped="handleTableDropped"
        @tableRenamed="handleTableRenamed"
        @tableTruncated="handleTableTruncated"
      />
    </div>

    <!-- 创建数据库/表模态框 -->
    <CreateDatabaseModal
      v-model:show="showCreateDatabaseModal"
      :server-id="selectedServerId"
      :server-type="selectedServerType"
      @success="handleCreateDatabaseSuccess"
    />
    <CreateTableModal
      v-model:show="showCreateTableModal"
      :server-id="selectedServerId"
      :server-type="selectedServerType"
      :database-name="selectedDatabase"
      @success="handleCreateTableSuccess"
    />
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
  background-color: var(--bg-secondary-color) !important;
  border: 1px solid var(--n-border-color) !important;
  border-bottom: none !important;
  margin-right: 4px !important;
}

:deep(.n-tabs-tab--active) {
  background-color: var(--bg-default-color) !important;
  border-color: var(--n-border-color) !important;
  border-bottom: 1px solid var(--bg-default-color) !important;
  margin-bottom: -1px !important;
  font-weight: bold;
}
</style>
