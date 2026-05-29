<script setup lang="ts">
import { ref, computed, h, onMounted, watch } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NPagination, NPopconfirm, NCheckbox, NTag, NInput, NModal } from 'naive-ui'
import { renderIcon } from '@/utils'
import { getDBManagerTableListAPI, execDBManagerSqlAPI } from '@/api/modules/database'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  serverOptions: any[]
}>()

const emit = defineEmits<{
  (e: 'selectTable', tableName: string): void
}>()

const message = useMessage()
const loading = ref(false)
const tables = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const checkedKeys = ref<string[]>([])
const batchLoading = ref(false)

// 视图管理
const viewMode = ref<'tables' | 'views'>('tables')
const views = ref<string[]>([])
const loadingViews = ref(false)
const viewDefinition = ref('')
const viewDefinitionTitle = ref('')
const showViewDefModal = ref(false)

const serverType = computed(() => {
  const s = props.serverOptions.find(s => s.value === props.selectedServerId)
  return s?.type || ''
})

const formatSize = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(1)} ${units[i]}`
}

const formatCount = (n: number) => {
  if (n === null || n === undefined) return '-'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return String(n)
}

const columns = [
  {
    type: 'selection' as const,
    width: 40,
    disabled: () => false
  },
  {
    title: '表名',
    key: 'name',
    width: 200,
    render: (row: any) => {
      return h('span', {
        class: 'cursor-pointer text-blue-600 hover:text-blue-800 font-medium',
        onClick: () => emit('selectTable', row.name)
      }, row.name)
    }
  },
  {
    title: '引擎',
    key: 'engine',
    width: 100,
    render: (row: any) => row.engine || '-'
  },
  {
    title: '行数',
    key: 'rowCount',
    width: 100,
    sorter: true,
    render: (row: any) => formatCount(row.rowCount)
  },
  {
    title: '大小',
    key: 'sizeBytes',
    width: 110,
    sorter: true,
    render: (row: any) => formatSize(row.sizeBytes)
  },
  {
    title: '字符集',
    key: 'collation',
    width: 130,
    render: (row: any) => row.collation || '-'
  },
  {
    title: '创建时间',
    key: 'createTime',
    width: 160,
    render: (row: any) => row.createTime || '-'
  },
  {
    title: '更新时间',
    key: 'updateTime',
    width: 160,
    render: (row: any) => row.updateTime || '-'
  }
]

const fetchData = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) return
  loading.value = true
  try {
    const res = await getDBManagerTableListAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      page: page.value,
      limit: limit.value
    })
    if (res.code === 0 && res.data) {
      tables.value = res.data.items || []
      total.value = res.data.total || 0
    } else {
      tables.value = []
      total.value = 0
    }
  } catch {
    tables.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const fetchViews = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) return
  loadingViews.value = true
  const t = serverType.value
  let sql = ''
  if (t === 'mysql' || t === 'mariadb') {
    sql = `SHOW FULL TABLES WHERE TABLE_TYPE = 'VIEW'`
  } else if (t === 'postgresql') {
    sql = `SELECT table_name FROM information_schema.views WHERE table_schema = 'public'`
  } else if (t === 'sqlite') {
    sql = `SELECT name FROM sqlite_master WHERE type = 'view' AND name NOT LIKE 'sqlite_%'`
  }
  if (!sql) { loadingViews.value = false; views.value = []; return }
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
    if (res.code === 0 && res.data && res.data.type === 'query') {
      const rows = res.data.rows || []
      if (t === 'mysql' || t === 'mariadb') {
        views.value = rows.map((r: any) => Object.values(r)[0] as string)
      } else {
        const key = Object.keys(rows[0] || {})[0]
        views.value = rows.map((r: any) => r[key] || '')
      }
    } else {
      views.value = []
    }
  } catch {
    views.value = []
  } finally {
    loadingViews.value = false
  }
}

const viewDefinitionSql = (viewName: string) => {
  const t = serverType.value
  const q = t === 'mysql' || t === 'mariadb' ? '`' : '"'
  if (t === 'mysql' || t === 'mariadb') return `SHOW CREATE VIEW ${q}${viewName}${q}`
  if (t === 'postgresql') return `SELECT pg_get_viewdef('${viewName.replace(/'/g, "''")}', true) AS definition`
  if (t === 'sqlite') return `SELECT sql FROM sqlite_master WHERE type = 'view' AND name = '${viewName.replace(/'/g, "''")}'`
  return ''
}

const openViewDefinition = async (viewName: string) => {
  const sql = viewDefinitionSql(viewName)
  if (!sql) { message.warning('当前数据库类型不支持查看视图定义'); return }
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId!, databaseName: props.selectedDatabase!, sql })
    if (res.code === 0 && res.data && res.data.type === 'query' && res.data.rows && res.data.rows.length > 0) {
      const row = res.data.rows[0]
      if (serverType.value === 'mysql' || serverType.value === 'mariadb') {
        viewDefinition.value = Object.values(row)[1] as string || Object.values(row)[0] as string
      } else {
        viewDefinition.value = Object.values(row)[0] as string
      }
    } else {
      viewDefinition.value = '-- 无法获取视图定义'
    }
  } catch {
    viewDefinition.value = '-- 获取视图定义失败'
  }
  viewDefinitionTitle.value = viewName
  showViewDefModal.value = true
}

const dropView = async (viewName: string) => {
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId!, databaseName: props.selectedDatabase!, sql: `DROP VIEW ${q}${viewName}${q}` })
    if (res.code === 0) {
      message.success(`视图 ${viewName} 已删除`)
      fetchViews()
    } else {
      message.error(res.message || '删除视图失败')
    }
  } catch {
    message.error('删除视图请求失败')
  }
}

const handleBatchTruncate = async () => {
  if (checkedKeys.value.length === 0) return
  batchLoading.value = true
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  let success = 0
  for (const name of checkedKeys.value) {
    try {
      const sql = serverType.value === 'sqlite'
        ? `DELETE FROM ${q}${name}${q}`
        : `TRUNCATE TABLE ${q}${name}${q}`
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId!,
        databaseName: props.selectedDatabase!,
        sql
      })
      if (res.code === 0) success++
    } catch { /* skip */ }
  }
  message.success(`已清空 ${success}/${checkedKeys.value.length} 个表`)
  checkedKeys.value = []
  if (success > 0) fetchData()
  batchLoading.value = false
}

const handleBatchDrop = async () => {
  if (checkedKeys.value.length === 0) return
  batchLoading.value = true
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  let success = 0
  for (const name of checkedKeys.value) {
    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId!,
        databaseName: props.selectedDatabase!,
        sql: `DROP TABLE ${q}${name}${q}`
      })
      if (res.code === 0) success++
    } catch { /* skip */ }
  }
  message.success(`已删除 ${success}/${checkedKeys.value.length} 个表`)
  checkedKeys.value = []
  if (success > 0) fetchData()
  batchLoading.value = false
}

watch(() => [props.selectedServerId, props.selectedDatabase], () => {
  page.value = 1
  checkedKeys.value = []
  viewMode.value = 'tables'
  fetchData()
})

watch(viewMode, (mode) => {
  if (mode === 'views') fetchViews()
})

onMounted(fetchData)
</script>

<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <div class="p-1.5 border-b border-slate-200 bg-white flex gap-1 text-xs">
      <n-button
        size="tiny"
        :type="viewMode === 'tables' ? 'primary' : 'default'"
        @click="viewMode = 'tables'"
      >
        <template #icon><n-icon :component="renderIcon('mdi:table-multiple')" /></template>
        表
      </n-button>
      <n-button
        size="tiny"
        :type="viewMode === 'views' ? 'primary' : 'default'"
        @click="viewMode = 'views'"
      >
        <template #icon><n-icon :component="renderIcon('mdi:eye-outline')" /></template>
        视图
      </n-button>
    </div>

    <!-- 表概览 -->
    <template v-if="viewMode === 'tables'">
      <div class="p-2 border-b border-slate-200 bg-[#f8f9fa] flex justify-between items-center text-xs">
        <div class="text-slate-700 flex items-center gap-2">
          <n-icon :component="renderIcon('mdi:table-multiple')" />
          <span class="font-semibold">表概览</span>
          <span class="text-slate-400">·</span>
          <span>{{ total }} 个表</span>
          <span v-if="checkedKeys.length > 0" class="text-blue-600 ml-2">已选 {{ checkedKeys.length }} 个</span>
        </div>
        <div class="flex gap-2 items-center">
          <template v-if="checkedKeys.length > 0">
            <n-popconfirm @positive-click="handleBatchTruncate">
              <template #trigger>
                <n-button size="tiny" type="warning" ghost :loading="batchLoading">清空选中</n-button>
              </template>
              确定要清空选中的 {{ checkedKeys.length }} 个表吗？此操作不可恢复！
            </n-popconfirm>
            <n-popconfirm @positive-click="handleBatchDrop">
              <template #trigger>
                <n-button size="tiny" type="error" ghost :loading="batchLoading">删除选中</n-button>
              </template>
              确定要删除选中的 {{ checkedKeys.length }} 个表吗？此操作不可恢复！
            </n-popconfirm>
          </template>
          <n-button size="tiny" @click="fetchData" :loading="loading">刷新</n-button>
        </div>
      </div>
      <div class="flex-1 p-2 bg-white overflow-auto">
        <n-empty v-if="!loading && tables.length === 0" description="当前数据库中没有表" class="mt-10" />
        <n-data-table
          v-else
          :columns="columns"
          :data="tables"
          :loading="loading"
          :pagination="false"
          :bordered="true"
          size="small"
          class="text-xs"
          :row-key="(row: any) => row.name"
          @update:checked-row-keys="checkedKeys = $event"
        />
      </div>
      <div v-if="total > 0" class="p-2 border-t border-slate-200 bg-white flex justify-end">
        <n-pagination
          :page="page"
          :page-size="limit"
          :item-count="total"
          :page-sizes="[20, 30, 50, 100]"
          size="small"
          show-size-picker
          @update:page="page = $event; fetchData()"
          @update:page-size="limit = $event; page = 1; fetchData()"
        />
      </div>
    </template>

    <!-- 视图列表 -->
    <template v-if="viewMode === 'views'">
      <div class="p-2 border-b border-slate-200 bg-[#f8f9fa] flex justify-between items-center text-xs">
        <div class="text-slate-700 flex items-center gap-2">
          <n-icon :component="renderIcon('mdi:eye-outline')" />
          <span class="font-semibold">视图管理</span>
          <span class="text-slate-400">·</span>
          <span>{{ views.length }} 个视图</span>
        </div>
        <div class="flex gap-2">
          <n-button size="tiny" @click="fetchViews" :loading="loadingViews">刷新</n-button>
        </div>
      </div>
      <div class="flex-1 p-2 bg-white overflow-auto">
        <n-empty v-if="!loadingViews && views.length === 0" description="当前数据库中没有视图" class="mt-10" />
        <div v-else class="flex flex-col gap-2">
          <div
            v-for="v in views"
            :key="v"
            class="flex items-center justify-between rounded border border-slate-200 px-3 py-2 hover:bg-slate-50"
          >
            <div class="flex items-center gap-2">
              <n-icon :component="renderIcon('mdi:eye-outline')" class="text-slate-400" />
              <span class="font-medium text-sm">{{ v }}</span>
            </div>
            <div class="flex gap-2">
              <n-button size="tiny" @click="openViewDefinition(v)">定义</n-button>
              <n-popconfirm @positive-click="dropView(v)">
                <template #trigger>
                  <n-button size="tiny" type="error" ghost>删除</n-button>
                </template>
                确定要删除视图 {{ v }} 吗？
              </n-popconfirm>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>

  <!-- 视图定义 Modal -->
  <n-modal
    v-model:show="showViewDefModal"
    preset="card"
    style="width: 700px"
    :title="`视图定义 - ${viewDefinitionTitle}`"
  >
    <n-input
      :value="viewDefinition"
      type="textarea"
      :rows="12"
      readonly
      class="font-mono text-xs"
    />
    <template #footer>
      <div class="flex justify-end">
        <n-button size="small" @click="showViewDefModal = false">关闭</n-button>
      </div>
    </template>
  </n-modal>
</template>
