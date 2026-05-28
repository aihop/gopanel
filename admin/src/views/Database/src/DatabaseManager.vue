<script setup lang="ts">
import { ref } from 'vue'
import { useMessage, NTabs, NTab, NIcon } from 'naive-ui'
import { renderIcon } from '@/utils'

import DataView from './components/DataView.vue'
import DatabaseManagerContextBar from './components/DatabaseManagerContextBar.vue'
import DatabaseManagerSidebar from './components/DatabaseManagerSidebar.vue'
import StructureView from './components/StructureView.vue'
import InsertView from './components/InsertView.vue'
import SqlView from './components/SqlView.vue'
import SearchView from './components/SearchView.vue'
import OperationsView from './components/OperationsView.vue'
import { useDatabaseManager } from './useDatabaseManager'

const props = defineProps<{
  defaultServerId?: number | null
  defaultDatabaseName?: string | null
}>()

const message = useMessage()
const dataViewRef = ref()
const {
  activeTab,
  activeTabLabel,
  applyTableSearch,
  clearSelectedTable,
  databaseOptions,
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
  selectedServerId.value = value
  if (value) {
    onServerChange(value)
  }
}

const handleDatabaseModelUpdate = (value: string | null) => {
  selectedDatabase.value = value
  if (value) {
    onDatabaseChange(value)
  }
}

const handleTableKeywordUpdate = (value: string) => {
  tableKeywordInput.value = value
}
</script>

<template>
  <div class="db-manager flex h-[650px] gap-3 p-2 bg-[#eef2f7] rounded-lg">
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
      @select-table="onTableSelect"
      @search-tables="applyTableSearch"
      @reset-table-search="resetTableSearch"
    />

    <div class="flex-1 border border-slate-200 bg-white flex flex-col overflow-hidden shadow-sm text-sm rounded-md">
      <DatabaseManagerContextBar
        :selected-server-label="selectedServerLabel"
        :selected-database="selectedDatabase"
        :selected-table="selectedTable"
        :active-tab-label="activeTabLabel"
        :table-count="sidebarTableTotal"
        @back-to-tables="clearSelectedTable"
      />

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
        :key="`data:${viewContextKey}`"
        v-if="activeTab === 'data'"
        ref="dataViewRef"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
        @select-table="onTableSelect"
        @edit-record="handleEditRecord"
        @copy-record="handleCopyRecord"
      />
      <StructureView
        :key="`structure:${viewContextKey}`"
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
        :key="`insert:${viewContextKey}:${isEditing ? 'edit' : 'create'}`"
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
        :key="`sql:${viewContextKey}`"
        v-if="activeTab === 'sql'"
        :selectedServerId="selectedServerId"
        :selectedDatabase="selectedDatabase"
        :selectedTable="selectedTable"
        :serverOptions="serverOptions"
      />
      <SearchView
        :key="`search:${viewContextKey}`"
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
        :key="`operations:${viewContextKey}`"
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
