<script setup lang="ts">
import { ref, h, computed, watch } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NPopconfirm, NModal, NInput, NSelect } from 'naive-ui'
import { execDBManagerSqlAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

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

// 结构操作状态
const showColumnModal = ref(false)
const isEditColumn = ref(false)
const submittingColumn = ref(false)
const columnForm = ref({
  oldName: '',
  name: '',
  type: ''
})

const openAddColumnModal = () => {
  isEditColumn.value = false
  columnForm.value = { oldName: '', name: '', type: 'VARCHAR(255)' }
  showColumnModal.value = true
}

const openEditColumnModal = (row: any) => {
  isEditColumn.value = true
  columnForm.value = {
    oldName: row.Field,
    name: row.Field,
    type: row.Type
  }
  showColumnModal.value = true
}

const dropColumn = async (row: any) => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  const server = props.serverOptions.find(s => s.value === props.selectedServerId)
  const isPg = server?.type === 'postgresql'
  
  const sql = isPg 
    ? `ALTER TABLE "${props.selectedTable}" DROP COLUMN "${row.Field}"`
    : `ALTER TABLE \`${props.selectedTable}\` DROP COLUMN \`${row.Field}\``

  try {
    const res = await execDBManagerSqlAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      sql
    })
    if (res.code === 0) {
      message.success('删除字段成功')
      emit('refresh')
      fetchTableIndexes()
    } else {
      message.error(res.message || '删除字段失败')
    }
  } catch (e) {
    message.error('执行删除请求失败')
  }
}

const submitColumn = async () => {
  if (!columnForm.value.name || !columnForm.value.type) {
    message.warning('字段名和类型不能为空')
    return
  }
  
  const server = props.serverOptions.find(s => s.value === props.selectedServerId)
  const isPg = server?.type === 'postgresql'
  const table = props.selectedTable
  let sql = ''

  if (isEditColumn.value) {
    if (isPg) {
      const queries = []
      if (columnForm.value.oldName !== columnForm.value.name) {
        queries.push(`ALTER TABLE "${table}" RENAME COLUMN "${columnForm.value.oldName}" TO "${columnForm.value.name}"`)
      }
      if (columnForm.value.type) {
        queries.push(`ALTER TABLE "${table}" ALTER COLUMN "${columnForm.value.name}" TYPE ${columnForm.value.type} USING "${columnForm.value.name}"::${columnForm.value.type}`)
      }
      sql = queries.join('; ')
    } else {
      sql = `ALTER TABLE \`${table}\` CHANGE COLUMN \`${columnForm.value.oldName}\` \`${columnForm.value.name}\` ${columnForm.value.type}`
    }
  } else {
    if (isPg) {
      sql = `ALTER TABLE "${table}" ADD COLUMN "${columnForm.value.name}" ${columnForm.value.type}`
    } else {
      sql = `ALTER TABLE \`${table}\` ADD COLUMN \`${columnForm.value.name}\` ${columnForm.value.type}`
    }
  }

  submittingColumn.value = true
  try {
    const res = await execDBManagerSqlAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      sql
    })
    if (res.code === 0) {
      message.success('操作成功')
      showColumnModal.value = false
      emit('refresh')
      fetchTableIndexes()
    } else {
      message.error(res.message || '操作失败')
    }
  } catch (e) {
    message.error('执行失败')
  } finally {
    submittingColumn.value = false
  }
}

// 索引操作状态
const indexData = ref<any[]>([])
const loadingIndex = ref(false)
const showIndexModal = ref(false)
const submittingIndex = ref(false)
const indexForm = ref({
  name: '',
  type: 'INDEX',
  columns: [] as string[]
})

const indexTypeOptions = [
  { label: 'PRIMARY', value: 'PRIMARY' },
  { label: 'UNIQUE', value: 'UNIQUE' },
  { label: 'INDEX', value: 'INDEX' }
]

const indexColumnsOptions = computed(() => {
  return props.structureData.map((col: any) => ({
    label: col.Field,
    value: col.Field
  }))
})

const fetchTableIndexes = async () => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  
  const server = props.serverOptions.find(s => s.value === props.selectedServerId)
  if (!server) return

  loadingIndex.value = true
  let sql = ''
  if (server.type === 'mysql') {
    sql = `SHOW INDEX FROM \`${props.selectedTable}\``
  } else {
    // postgresql
    sql = `
      SELECT
        i.relname as "Key_name",
        ix.indisunique as "Non_unique",
        a.attname as "Column_name",
        am.amname as "Index_type"
      FROM
        pg_class t,
        pg_class i,
        pg_index ix,
        pg_attribute a,
        pg_am am
      WHERE
        t.oid = ix.indrelid
        and i.oid = ix.indexrelid
        and a.attrelid = t.oid
        and a.attnum = ANY(ix.indkey)
        and i.relam = am.oid
        and t.relkind = 'r'
        and t.relname = '${props.selectedTable}'
    `
  }

  try {
    const res = await execDBManagerSqlAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      sql: sql
    })
    
    if (res.code === 0 && res.data && res.data.type === 'query') {
      if (server.type === 'postgresql') {
        // postgresql 的字段映射到 mysql 的字段名以便统一处理
        indexData.value = res.data.rows.map((row: any) => ({
          Key_name: row.Key_name,
          Non_unique: row.Non_unique ? 0 : 1,
          Column_name: row.Column_name,
          Index_type: row.Index_type
        })) || []
      } else {
        indexData.value = res.data.rows || []
      }
    }
  } catch (error) {
    message.error("获取索引数据失败")
  } finally {
    loadingIndex.value = false
  }
}

watch(() => props.selectedTable, () => {
  if (props.selectedTable) {
    fetchTableIndexes()
  } else {
    indexData.value = []
  }
})

const openAddIndexModal = () => {
  indexForm.value = { name: '', type: 'INDEX', columns: [] }
  showIndexModal.value = true
}

const dropIndex = async (row: any) => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  const server = props.serverOptions.find(s => s.value === props.selectedServerId)
  const isPg = server?.type === 'postgresql'
  const table = props.selectedTable
  const indexName = row.Key_name
  
  let sql = ''
  if (isPg) {
    if (indexName.endsWith('_pkey')) {
      sql = `ALTER TABLE "${table}" DROP CONSTRAINT "${indexName}"`
    } else {
      sql = `DROP INDEX "${indexName}"`
    }
  } else {
    if (indexName === 'PRIMARY') {
      sql = `ALTER TABLE \`${table}\` DROP PRIMARY KEY`
    } else {
      sql = `ALTER TABLE \`${table}\` DROP INDEX \`${indexName}\``
    }
  }

  try {
    const res = await execDBManagerSqlAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      sql
    })
    if (res.code === 0) {
      message.success('删除索引成功')
      fetchTableIndexes()
    } else {
      message.error(res.message || '删除索引失败')
    }
  } catch (e) {
    message.error('执行删除请求失败')
  }
}

const submitIndex = async () => {
  if (!indexForm.value.type || indexForm.value.columns.length === 0) {
    message.warning('请选择索引类型和相关字段')
    return
  }
  
  const server = props.serverOptions.find(s => s.value === props.selectedServerId)
  const isPg = server?.type === 'postgresql'
  const table = props.selectedTable
  let sql = ''
  
  const colStr = indexForm.value.columns.map(c => isPg ? `"${c}"` : `\`${c}\``).join(', ')
  const indexName = indexForm.value.name || `${table}_${indexForm.value.columns.join('_')}_idx`

  if (isPg) {
    if (indexForm.value.type === 'PRIMARY') {
      sql = `ALTER TABLE "${table}" ADD PRIMARY KEY (${colStr})`
    } else if (indexForm.value.type === 'UNIQUE') {
      sql = `CREATE UNIQUE INDEX "${indexName}" ON "${table}" (${colStr})`
    } else {
      sql = `CREATE INDEX "${indexName}" ON "${table}" (${colStr})`
    }
  } else {
    if (indexForm.value.type === 'PRIMARY') {
      sql = `ALTER TABLE \`${table}\` ADD PRIMARY KEY (${colStr})`
    } else if (indexForm.value.type === 'UNIQUE') {
      sql = `ALTER TABLE \`${table}\` ADD UNIQUE INDEX \`${indexName}\` (${colStr})`
    } else {
      sql = `ALTER TABLE \`${table}\` ADD INDEX \`${indexName}\` (${colStr})`
    }
  }

  submittingIndex.value = true
  try {
    const res = await execDBManagerSqlAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      sql
    })
    if (res.code === 0) {
      message.success('创建索引成功')
      showIndexModal.value = false
      fetchTableIndexes()
    } else {
      message.error(res.message || '创建索引失败')
    }
  } catch (e) {
    message.error('执行失败')
  } finally {
    submittingIndex.value = false
  }
}

const indexColumns = [
  {
    title: '操作',
    key: 'actions',
    width: 80,
    render(row: any) {
      return h(NPopconfirm, { onPositiveClick: () => dropIndex(row) }, {
        trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
        default: () => `确定要删除索引 ${row.Key_name} 吗？`
      })
    }
  },
  { title: '键名 (Key_name)', key: 'Key_name', width: 150 },
  { title: '类型 (Type)', key: 'Index_type', width: 100 },
  { 
    title: '唯一 (Unique)', 
    key: 'Non_unique', 
    width: 100,
    render: (row: any) => row.Non_unique === 0 ? '是' : '否'
  },
  { title: '字段 (Column)', key: 'Column_name', width: 150 }
]

const structureColumns = computed(() => {
  if (!props.structureData || props.structureData.length === 0) return []
  const firstRow = props.structureData[0]
  const keys = Object.keys(firstRow)
  
  const actionCol = {
    title: '操作',
    key: 'actions',
    fixed: 'left' as const,
    width: 120,
    render(row: any) {
      return h('div', { class: 'flex gap-2' }, [
        h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => openEditColumnModal(row) }, { default: () => '修改' }),
        h(NPopconfirm, { onPositiveClick: () => dropColumn(row) }, {
          trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
          default: () => `确定要删除字段 ${row.Field} 吗？`
        })
      ])
    }
  }
  return [
    actionCol,
    ...keys.map(col => ({ title: col, key: col, ellipsis: { tooltip: true as const } }))
  ]
})
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
          <span class="mr-2">{{ selectedServerId ? serverOptions.find(s => s.value === selectedServerId)?.label : '' }}</span>
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
          <n-button
            size="tiny"
            type="primary"
            @click="openAddColumnModal"
          >{{ $t('database.addColumn') }}</n-button>
          <n-button
            size="tiny"
            @click="emit('refresh')"
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
      </div>
    </template>

    <n-modal
      v-model:show="showColumnModal"
      preset="card"
      class="w-[400px]"
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
          <n-input
            v-model:value="columnForm.type"
            placeholder="例如: INT, VARCHAR(255)"
            class="flex-1"
            size="small"
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
      class="w-[400px]"
      title="添加索引"
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
          >保存</n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>