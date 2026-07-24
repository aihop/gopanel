<script setup lang="ts">
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NModal, NInput, NSelect } from 'naive-ui'
import { renderIcon } from '@/utils'
import { useStructureView } from './useStructureView'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
  structureData: any[]
  loadingStructure: boolean
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const message = useMessage()
const {
  afterColumnOptions,
  columnForm,
  columnTypeOptions,
  dropForeignKey,
  fetchTableForeignKeys,
  fetchTableIndexes,
  fieldSummary,
  fkColumns,
  fkData,
  fkForm,
  fkRuleOptions,
  fkSummary,
  indexColumns,
  indexColumnsOptions,
  indexData,
  indexForm,
  indexSummary,
  indexTypeOptions,
  isEditColumn,
  isEditIndex,
  loadingFk,
  loadingIndex,
  loadingRefColumns,
  loadingRefTables,
  onRefTableChange,
  openAddColumnModal,
  openAddFkModal,
  openAddIndexModal,
  refColumns,
  refTables,
  selectedServerLabel,
  showColumnModal,
  showFkModal,
  showIndexModal,
  structureColumns,
  submitColumn,
  submitForeignKey,
  submitIndex,
  submittingColumn,
  submittingFk,
  submittingIndex
} = useStructureView(props, {
  refresh: () => emit('refresh')
}, message)

const handleRefresh = () => {
  emit('refresh')
  fetchTableIndexes()
  fetchTableForeignKeys()
}
</script>

<template>
  <div class="flex-1 flex flex-col relative overflow-hidden">
    <div
      v-if="!selectedTable"
      class="flex-1 flex items-center justify-center text-slate-400"
    >
      <n-empty :description="$t('database.selectTable')" />
    </div>
    <template v-else>
      <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs">
        <div class="text-slate-700 flex items-center gap-1">
          <n-icon :component="renderIcon('mdi:server')" />
          <span class="mr-2">{{ selectedServerLabel }}</span>
          <span>»</span>
          <n-icon
            :component="renderIcon('mdi:database')"
            class="ml-2"
          />
          <span class="mr-2">{{ selectedDatabase }}</span>
          <span>»</span>
          <n-icon
            :component="renderIcon('mdi:table-cog')"
            class="ml-2"
          />
          <span class="font-bold">{{ selectedTable }} ({{ $t('database.structure') }})</span>
        </div>
        <div class="flex gap-2">
          <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500 mr-2">
            <span class="px-2 py-1 rounded bg-slate-100">{{ fieldSummary.total }} 个字段</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ fieldSummary.primaryCount }} 个主键列</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ indexSummary.total }} 个索引项</span>
          </div>
          <n-button
            size="tiny"
            type="primary"
            @click="openAddColumnModal"
          >{{ $t('database.addColumn') }}</n-button>
          <n-button
            size="tiny"
            @click="handleRefresh"
            :loading="loadingStructure"
          >{{ $t('commons.button.refresh') }}</n-button>
        </div>
      </div>
      <div class="flex-1 overflow-auto bg-white flex flex-col">
        <div class="p-2 border-b border-slate-200">
          <n-data-table
            :columns="structureColumns"
            :data="structureData"
            :loading="loadingStructure"
            :pagination="false"
            :bordered="true"
            size="small"
            :scroll-x="structureColumns.length * 120"
            class="text-xs"
          />
        </div>

        <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs mt-4">
          <div class="text-slate-700 flex items-center gap-1">
            <n-icon :component="renderIcon('mdi:key-outline')" />
            <span class="font-bold">索引 (Indexes)</span>
            <span class="px-2 py-0.5 rounded bg-slate-100 text-[11px] text-slate-500 ml-2">{{ indexSummary.uniqueCount }} 个唯一索引</span>
            <span class="px-2 py-0.5 rounded bg-slate-100 text-[11px] text-slate-500">{{ indexSummary.primaryCount }} 个主键索引</span>
          </div>
          <div class="flex gap-2">
            <n-button
              size="tiny"
              type="primary"
              @click="openAddIndexModal"
            >添加索引</n-button>
            <n-button
              size="tiny"
              @click="fetchTableIndexes"
              :loading="loadingIndex"
            >刷新</n-button>
          </div>
        </div>
        <div class="p-2 flex-1">
          <n-data-table
            :columns="indexColumns"
            :data="indexData"
            :loading="loadingIndex"
            :pagination="false"
            :bordered="true"
            size="small"
            class="text-xs"
          />
        </div>

        <!-- ═══ 外键 (Foreign Keys) ═══ -->
        <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs">
          <div class="text-slate-700 flex items-center gap-1">
            <n-icon :component="renderIcon('mdi:link-variant')" />
            <span class="font-bold">外键 (Foreign Keys)</span>
            <span class="px-2 py-0.5 rounded bg-slate-100 text-[11px] text-slate-500 ml-2">{{ fkSummary.total }} 个外键</span>
            <span v-if="fkSummary.cascadeDelete > 0" class="px-2 py-0.5 rounded bg-slate-100 text-[11px] text-slate-500">{{ fkSummary.cascadeDelete }} 个级联删除</span>
          </div>
          <div class="flex gap-2">
            <n-button
              size="tiny"
              type="primary"
              @click="openAddFkModal"
            >添加外键</n-button>
            <n-button
              size="tiny"
              @click="fetchTableForeignKeys"
              :loading="loadingFk"
            >刷新</n-button>
          </div>
        </div>
        <div class="p-2 flex-1">
          <n-data-table
            :columns="fkColumns"
            :data="fkData"
            :loading="loadingFk"
            :pagination="false"
            :bordered="true"
            size="small"
            class="text-xs"
          />
        </div>
      </div>
    </template>

    <n-modal
      v-model:show="showColumnModal"
      preset="card"
      style="width: 480px"
      :title="isEditColumn ? '修改字段' : '添加字段'"
    >
      <div class="flex flex-col gap-4 text-sm">
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">字段名:</span>
          <n-input
            v-model:value="columnForm.name"
            placeholder="例如: user_id"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">类型:</span>
          <n-select
            v-model:value="columnForm.type"
            :options="columnTypeOptions"
            filterable
            tag
            :show-checkmark="false"
            placeholder="选择或输入类型，如 VARCHAR(255)"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">可空:</span>
          <n-switch v-model:value="columnForm.nullable" size="small" />
          <span class="text-slate-500 text-xs ml-1">{{ columnForm.nullable ? 'NULL' : 'NOT NULL' }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">默认值:</span>
          <n-input
            v-model:value="columnForm.defaultValue"
            placeholder="留空则不设默认值"
            class="flex-1"
            size="small"
            clearable
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">备注:</span>
          <n-input
            v-model:value="columnForm.comment"
            placeholder="字段备注说明"
            class="flex-1"
            size="small"
            clearable
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">自增:</span>
          <n-switch v-model:value="columnForm.autoIncrement" size="small" />
          <span class="text-slate-500 text-xs ml-1">AUTO_INCREMENT</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">位置:</span>
          <n-select
            v-model:value="columnForm.afterColumn"
            :options="afterColumnOptions"
            placeholder="默认追加到最后"
            class="flex-1"
            size="small"
            clearable
          />
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <n-button
            size="small"
            @click="showColumnModal = false"
          >取消</n-button>
          <n-button
            size="small"
            type="primary"
            @click="submitColumn"
            :loading="submittingColumn"
          >保存</n-button>
        </div>
      </div>
    </n-modal>

    <n-modal
      v-model:show="showIndexModal"
      preset="card"
      style="width: 400px;"
      :title="isEditIndex ? '修改索引' : '添加索引'"
    >
      <div class="flex flex-col gap-4 text-sm">
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">索引名:</span>
          <n-input
            v-model:value="indexForm.name"
            placeholder="留空则自动生成"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">类型:</span>
          <n-select
            v-model:value="indexForm.type"
            :options="indexTypeOptions"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-16 text-right">包含列:</span>
          <n-select
            v-model:value="indexForm.columns"
            multiple
            :options="indexColumnsOptions"
            class="flex-1"
            size="small"
            placeholder="请选择列"
          />
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <n-button
            size="small"
            @click="showIndexModal = false"
          >取消</n-button>
          <n-button
            size="small"
            type="primary"
            @click="submitIndex"
            :loading="submittingIndex"
          >{{ isEditIndex ? '保存修改' : '保存' }}</n-button>
        </div>
      </div>
    </n-modal>

    <!-- ═══ 添加外键 Modal ═══ -->
    <n-modal
      v-model:show="showFkModal"
      preset="card"
      style="width: 520px"
      title="添加外键"
    >
      <div class="flex flex-col gap-4 text-sm">
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">约束名:</span>
          <n-input
            v-model:value="fkForm.name"
            placeholder="留空则自动生成 (fk_表名_字段名)"
            class="flex-1"
            size="small"
            clearable
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">当前字段:</span>
          <n-select
            v-model:value="fkForm.column"
            :options="indexColumnsOptions"
            placeholder="选择本表的字段"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">引用表:</span>
          <n-select
            v-model:value="fkForm.refTable"
            :options="refTables"
            placeholder="选择引用的表"
            class="flex-1"
            size="small"
            :loading="loadingRefTables"
            @update:value="onRefTableChange"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">引用列:</span>
          <n-select
            v-model:value="fkForm.refColumn"
            :options="refColumns"
            placeholder="选择引用的列"
            class="flex-1"
            size="small"
            :loading="loadingRefColumns"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">删除规则:</span>
          <n-select
            v-model:value="fkForm.onDelete"
            :options="fkRuleOptions"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">更新规则:</span>
          <n-select
            v-model:value="fkForm.onUpdate"
            :options="fkRuleOptions"
            class="flex-1"
            size="small"
          />
        </div>
        <div class="text-xs text-slate-400 mt-1">
          外键约束要求引用表的引用列必须包含在索引中（主键或唯一索引）。
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button size="small" @click="showFkModal = false">取消</n-button>
          <n-button
            size="small"
            type="primary"
            :loading="submittingFk"
            @click="submitForeignKey"
          >保存</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
:deep(.db-structure-primary-col) {
  background: var(--bg-secondary-color);
  font-weight: 600;
}
</style>
