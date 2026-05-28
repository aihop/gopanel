<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useMessage, NEmpty, NButton, NInput, NInputGroup, NCard, NPopconfirm } from 'naive-ui'
import { execDBManagerSqlAPI } from '@/api/modules/database'
import DatabaseWorkspaceHeader from './DatabaseWorkspaceHeader.vue'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
}>()

const emit = defineEmits<{
  (e: 'tableDropped'): void
  (e: 'tableRenamed', newName: string): void
  (e: 'tableTruncated'): void
}>()

const message = useMessage()
const newTableName = ref('')
const loadingTruncate = ref(false)
const loadingDrop = ref(false)
const loadingRename = ref(false)

const selectedServerLabel = computed(() => {
  if (!props.selectedServerId) return ''
  return props.serverOptions.find(s => s.value === props.selectedServerId)?.label || ''
})

const serverType = computed(() => {
  if (!props.selectedServerId) return ''
  const s = props.serverOptions.find(s => s.value === props.selectedServerId)
  return s ? s.type : ''
})

const getQuote = () => {
  return serverType.value === 'mysql' ? '`' : '"'
}

const tableSummary = computed(() => {
  const quote = getQuote()
  return {
    engine: serverType.value || '-',
    quotedName: props.selectedTable ? `${quote}${props.selectedTable}${quote}` : '-',
    databaseName: props.selectedDatabase || '-'
  }
})

watch(() => props.selectedTable, (tableName) => {
  newTableName.value = tableName || ''
}, { immediate: true })

const execSql = async (sql: string) => {
  if (!props.selectedServerId || !props.selectedDatabase) throw new Error('Missing server or database')
  return await execDBManagerSqlAPI({
    serverId: props.selectedServerId,
    databaseName: props.selectedDatabase,
    sql
  })
}

const handleTruncate = async () => {
  if (!props.selectedTable) return
  loadingTruncate.value = true
  try {
    const q = getQuote()
    let sql = `TRUNCATE TABLE ${q}${props.selectedTable}${q}`
    if (serverType.value === 'sqlite') {
      sql = `DELETE FROM ${q}${props.selectedTable}${q}`
    }
    const res = await execSql(sql)
    if (res.code === 0) {
      message.success('表数据已清空')
      emit('tableTruncated')
    } else {
      message.error(res.message || '清空失败')
    }
  } catch (err: any) {
    message.error('请求失败')
  } finally {
    loadingTruncate.value = false
  }
}

const handleDrop = async () => {
  if (!props.selectedTable) return
  loadingDrop.value = true
  try {
    const q = getQuote()
    const res = await execSql(`DROP TABLE ${q}${props.selectedTable}${q}`)
    if (res.code === 0) {
      message.success('表已删除')
      emit('tableDropped')
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (err: any) {
    message.error('请求失败')
  } finally {
    loadingDrop.value = false
  }
}

const handleRename = async () => {
  if (!props.selectedTable || !newTableName.value) {
    message.warning('请输入新表名')
    return
  }
  loadingRename.value = true
  try {
    const q = getQuote()
    const res = await execSql(`ALTER TABLE ${q}${props.selectedTable}${q} RENAME TO ${q}${newTableName.value}${q}`)
    if (res.code === 0) {
      message.success('表重命名成功')
      emit('tableRenamed', newTableName.value)
      newTableName.value = ''
    } else {
      message.error(res.message || '重命名失败')
    }
  } catch (err: any) {
    message.error('请求失败')
  } finally {
    loadingRename.value = false
  }
}
</script>

<template>
  <div class="flex-1 flex flex-col relative overflow-hidden">
    <div v-if="!selectedTable" class="flex-1 flex items-center justify-center text-slate-400">
      <n-empty :description="$t('database.selectTable')" />
    </div>
    <template v-else>
      <DatabaseWorkspaceHeader
        :server-label="selectedServerLabel"
        :database-name="selectedDatabase"
        :table-name="selectedTable"
        :title="`${selectedTable} (表操作)`"
        icon="mdi:cog-outline"
      >
        <template #summary>
          <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500">
            <span class="px-2 py-1 rounded bg-slate-100 uppercase">{{ tableSummary.engine }}</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ tableSummary.quotedName }}</span>
          </div>
        </template>
      </DatabaseWorkspaceHeader>
      <div class="flex-1 overflow-auto p-4 bg-white">
        <div class="max-w-3xl mx-auto flex flex-col gap-4">
          <n-card title="表摘要" size="small" class="shadow-sm">
            <div class="grid grid-cols-3 gap-3 text-xs">
              <div class="rounded border border-slate-200 bg-slate-50 px-3 py-2">
                <div class="text-slate-400">数据库</div>
                <div class="mt-1 font-semibold text-slate-700 break-all">{{ tableSummary.databaseName }}</div>
              </div>
              <div class="rounded border border-slate-200 bg-slate-50 px-3 py-2">
                <div class="text-slate-400">表标识</div>
                <div class="mt-1 font-semibold text-slate-700 break-all">{{ tableSummary.quotedName }}</div>
              </div>
              <div class="rounded border border-slate-200 bg-slate-50 px-3 py-2">
                <div class="text-slate-400">数据库类型</div>
                <div class="mt-1 font-semibold text-slate-700 uppercase">{{ tableSummary.engine }}</div>
              </div>
            </div>
          </n-card>

          <n-card title="表维护" size="small" class="shadow-sm">
            <div class="text-xs text-slate-500 mb-3">
              用于执行表级维护动作。`TRUNCATE` 会清空记录但保留表结构，`DROP` 会直接删除整张表。
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div class="rounded border border-amber-200 bg-amber-50 p-3">
                <div class="font-semibold text-amber-700 mb-2">清空数据</div>
                <div class="text-xs text-amber-700/80 mb-3">
                  保留字段和索引，仅删除当前表中的所有记录。
                </div>
                <n-popconfirm @positive-click="handleTruncate">
                  <template #trigger>
                    <n-button type="warning" ghost :loading="loadingTruncate">
                      <template #icon><n-icon :component="renderIcon('mdi:delete-sweep-outline')" /></template>
                      清空数据 (TRUNCATE)
                    </n-button>
                  </template>
                  确定要清空表中的所有数据吗？此操作不可恢复！
                </n-popconfirm>
              </div>

              <div class="rounded border border-red-200 bg-red-50 p-3">
                <div class="font-semibold text-red-700 mb-2">删除整表</div>
                <div class="text-xs text-red-700/80 mb-3">
                  删除表结构、记录和相关对象，执行后将从当前数据库移除该表。
                </div>
                <n-popconfirm @positive-click="handleDrop">
                  <template #trigger>
                    <n-button type="error" ghost :loading="loadingDrop">
                      <template #icon><n-icon :component="renderIcon('mdi:table-remove')" /></template>
                      删除表 (DROP)
                    </n-button>
                  </template>
                  确定要删除整个表吗？此操作不可恢复！
                </n-popconfirm>
              </div>
            </div>
          </n-card>

          <n-card title="表选项" size="small" class="shadow-sm">
            <div class="text-xs text-slate-500 mb-3">
              重命名只会修改表名，不会改变字段和记录。建议仅使用字母、数字和下划线。
            </div>
            <div class="flex items-center gap-4">
              <div class="w-24 text-slate-600 font-medium">重命名表到：</div>
              <n-input-group>
                <n-input v-model:value="newTableName" placeholder="输入新表名..." />
                <n-button type="primary" @click="handleRename" :loading="loadingRename">执行</n-button>
              </n-input-group>
            </div>
          </n-card>

        </div>
      </div>
    </template>
  </div>
</template>
