<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NInput, NAlert, NPopconfirm } from 'naive-ui'
import { execDBManagerSqlAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'
import DatabaseWorkspaceHeader from './DatabaseWorkspaceHeader.vue'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
}>()

const message = useMessage()
const sqlQuery = ref('')
const executingSql = ref(false)
const sqlResult = ref<any>(null)
const sqlHistory = ref<Array<{ title: string; sql: string }>>([])

const historyStorageKey = computed(() => {
  return props.selectedServerId && props.selectedDatabase
    ? `sql_history_${props.selectedServerId}_${props.selectedDatabase}`
    : null
})

const loadHistory = () => {
  const key = historyStorageKey.value
  if (!key) { sqlHistory.value = []; return }
  try {
    const raw = localStorage.getItem(key)
    sqlHistory.value = raw ? JSON.parse(raw) : []
  } catch {
    sqlHistory.value = []
  }
}

const saveHistory = () => {
  const key = historyStorageKey.value
  if (!key) return
  try {
    localStorage.setItem(key, JSON.stringify(sqlHistory.value))
  } catch { /* quota exceeded - ignore */ }
}

const clearHistory = () => {
  const key = historyStorageKey.value
  sqlHistory.value = []
  if (key) localStorage.removeItem(key)
}

onMounted(loadHistory)

const selectedServerLabel = computed(() => {
  return props.selectedServerId
    ? props.serverOptions.find(s => s.value === props.selectedServerId)?.label || ''
    : ''
})

const quickSqlTemplates = computed(() => {
  if (!props.selectedTable) return []

  const tableName = props.selectedTable
  return [
    {
      label: '浏览前 20 行',
      sql: `SELECT * FROM \`${tableName}\` LIMIT 20;`
    },
    {
      label: '统计总行数',
      sql: `SELECT COUNT(*) AS total_count FROM \`${tableName}\`;`
    },
    {
      label: '按主键倒序',
      sql: `SELECT * FROM \`${tableName}\` ORDER BY 1 DESC LIMIT 20;`
    },
    {
      label: '插入模板',
      sql: `INSERT INTO \`${tableName}\` ()\nVALUES ();`
    },
    {
      label: '更新模板',
      sql: `UPDATE \`${tableName}\`\nSET \nWHERE ;`
    },
    {
      label: '删除模板',
      sql: `DELETE FROM \`${tableName}\`\nWHERE ;`
    }
  ]
})

const sqlResultColumns = computed(() => {
  if (sqlResult.value && sqlResult.value.type === 'query' && sqlResult.value.columns) {
    return sqlResult.value.columns.map((col: string) => ({
      title: col,
      key: col,
      ellipsis: { tooltip: true as const }
    }))
  }
  return []
})

const sqlResultSummary = computed(() => {
  if (!sqlResult.value) return []
  if (sqlResult.value.type === 'query') {
    return [
      `${sqlResult.value.rows?.length || 0} 行`,
      `${sqlResult.value.columns?.length || 0} 列`,
      '查询结果'
    ]
  }
  return [
    `${sqlResult.value.affected || 0} 行受影响`,
    '执行语句'
  ]
})

watch(
  () => [props.selectedServerId, props.selectedDatabase],
  () => {
    sqlResult.value = null
    loadHistory()
  },
  { immediate: true }
)

const applyQuickSql = (sql: string) => {
  sqlQuery.value = sql
}

const pushSqlHistory = (sql: string) => {
  const normalized = sql.trim()
  if (!normalized) return
  const nextTitle = normalized.split('\n')[0].slice(0, 48)
  sqlHistory.value = [
    { title: nextTitle, sql: normalized },
    ...sqlHistory.value.filter(item => item.sql !== normalized)
  ].slice(0, 6)
  saveHistory()
}

const executeSql = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) {
    message.warning("请先选择服务器和数据库")
    return
  }
  if (!sqlQuery.value.trim()) return

  executingSql.value = true
  sqlResult.value = null
  
  try {
    const res = await execDBManagerSqlAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      sql: sqlQuery.value
    })
    
    if (res.code === 0) {
      sqlResult.value = res.data
      pushSqlHistory(sqlQuery.value)
      if (res.data.type === 'query') {
        message.success(`查询成功，共 ${res.data.rows?.length || 0} 条记录`)
      } else {
        message.success(`执行成功，影响 ${res.data.affected || 0} 行`)
      }
    } 
  } catch (error: any) {
    message.error(error.message || "执行 SQL 失败")
  } finally {
    executingSql.value = false
  }
}
</script>

<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <DatabaseWorkspaceHeader
      :server-label="selectedServerLabel"
      :database-name="selectedDatabase"
      :table-name="selectedTable"
      :title="selectedTable ? `${selectedTable} (SQL)` : 'SQL 工作台'"
      icon="mdi:console"
    >
      <template #summary>
        <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500">
          <span class="px-2 py-1 rounded bg-slate-100">{{ quickSqlTemplates.length }} 个快捷模板</span>
          <span
            v-if="sqlResultSummary.length > 0"
            class="px-2 py-1 rounded bg-blue-50 text-blue-600"
          >{{ sqlResultSummary.join(' · ') }}</span>
        </div>
      </template>
    </DatabaseWorkspaceHeader>
    <div
      v-if="quickSqlTemplates.length > 0"
      class="px-2 py-2 border-b border-slate-200 bg-white flex flex-wrap gap-2"
    >
      <n-button
        v-for="item in quickSqlTemplates"
        :key="item.label"
        size="tiny"
        quaternary
        @click="applyQuickSql(item.sql)"
      >
        {{ item.label }}
      </n-button>
    </div>
    <div class="h-48 border-b border-slate-200 relative flex flex-col p-2 bg-white">
      <div class="text-xs text-slate-600 mb-1 font-semibold flex items-center gap-1">
        <n-icon :component="renderIcon('mdi:console')" />
        在数据库 {{ selectedDatabase }}{{ selectedTable ? ` / 表 ${selectedTable}` : '' }} 中执行 SQL:
      </div>
      <n-input
        v-model:value="sqlQuery"
        type="textarea"
        :placeholder="selectedTable ? `输入 SQL 语句，例如: SELECT * FROM \`${selectedTable}\` LIMIT 20;` : '输入 SQL 语句，例如: SELECT * FROM users LIMIT 10;'"
        class="flex-1 border border-slate-300 font-mono text-xs"
        @keydown.ctrl.enter="executeSql"
        @keydown.meta.enter="executeSql"
      />
      <div class="flex justify-end mt-2 gap-2">
        <n-button
          size="small"
          @click="sqlQuery = ''"
        >清空</n-button>
        <n-button
          type="primary"
          size="small"
          @click="executeSql"
          :loading="executingSql"
          :disabled="!sqlQuery.trim() || !selectedDatabase"
        >执行 (Ctrl+Enter)</n-button>
      </div>
    </div>
    <div class="flex-1 overflow-auto p-2 bg-[#f0f0f0]">
      <div
        v-if="sqlHistory.length > 0"
        class="mb-2 border border-slate-200 bg-white"
      >
        <div class="px-3 py-2 border-b border-slate-200 text-xs font-semibold text-slate-700 flex justify-between items-center">
          <span>最近执行</span>
          <n-popconfirm @positive-click="clearHistory">
            <template #trigger>
              <n-button size="tiny" text type="error">清空</n-button>
            </template>
            确定要清空当前数据库的历史记录吗？
          </n-popconfirm>
        </div>
        <div class="p-2 flex flex-wrap gap-2">
          <n-button
            v-for="item in sqlHistory"
            :key="item.sql"
            size="tiny"
            quaternary
            @click="applyQuickSql(item.sql)"
          >
            {{ item.title }}
          </n-button>
        </div>
      </div>
      <n-empty
        v-if="!sqlResult"
        description="执行结果将显示在这里"
        class="mt-10"
      />
      <template v-else>
        <n-alert
          v-if="sqlResult.type === 'exec'"
          type="success"
          title="执行成功"
        >
          影响行数: {{ sqlResult.affected }}
        </n-alert>
        <div
          v-else
          class="h-full bg-white border border-slate-200"
        >
          <n-data-table
            :columns="sqlResultColumns"
            :data="sqlResult.rows"
            :bordered="true"
            size="small"
            :scroll-x="sqlResultColumns.length * 120"
            class="text-xs"
            flex-height
          />
        </div>
      </template>
    </div>
  </div>
</template>
