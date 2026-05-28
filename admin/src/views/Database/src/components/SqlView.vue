<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NInput, NAlert } from 'naive-ui'
import { execDBManagerSqlAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

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

watch(
  () => [props.selectedServerId, props.selectedDatabase, props.selectedTable],
  () => {
    sqlResult.value = null
  }
)

const applyQuickSql = (sql: string) => {
  sqlQuery.value = sql
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
      if (res.data.type === 'query') {
        message.success(`查询成功，共 ${res.data.rows?.length || 0} 条记录`)
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
    <div class="p-2 border-b border-slate-200 bg-[#f8f9fa] text-xs text-slate-700 flex items-center gap-1">
      <n-icon :component="renderIcon('mdi:server')" />
      <span class="mr-2">{{ selectedServerLabel }}</span>
      <span>»</span>
      <n-icon
        :component="renderIcon('mdi:database')"
        class="ml-2"
      />
      <span class="font-bold">{{ selectedDatabase }}</span>
      <template v-if="selectedTable">
        <span>»</span>
        <n-icon
          :component="renderIcon('mdi:table')"
          class="ml-2"
        />
        <span class="font-bold">{{ selectedTable }}</span>
      </template>
    </div>
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
