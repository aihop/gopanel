<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage, NEmpty, NIcon, NButton, NInput, NInputGroup, NCard, NPopconfirm } from 'naive-ui'
import { execDBManagerSqlAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

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

const serverType = computed(() => {
  if (!props.selectedServerId) return ''
  const s = props.serverOptions.find(s => s.value === props.selectedServerId)
  return s ? s.type : ''
})

const getQuote = () => {
  return serverType.value === 'mysql' ? '`' : '"'
}

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
      <div class="p-2 border-b border-slate-200 flex justify-between items-center bg-[#f8f9fa] text-xs">
        <div class="text-slate-700 flex items-center gap-1">
          <n-icon :component="renderIcon('mdi:server')" />
          <span class="mr-2">{{ selectedServerId ? serverOptions.find(s => s.value === selectedServerId)?.label : '' }}</span>
          <span>»</span>
          <n-icon :component="renderIcon('mdi:database')" class="ml-2" />
          <span class="mr-2">{{ selectedDatabase }}</span>
          <span>»</span>
          <n-icon :component="renderIcon('mdi:table')" class="ml-2" />
          <span class="font-bold">{{ selectedTable }} (表操作)</span>
        </div>
      </div>
      <div class="flex-1 overflow-auto p-4 bg-white">
        <div class="max-w-3xl mx-auto flex flex-col gap-4">
          
          <n-card title="表维护" size="small" class="shadow-sm">
            <div class="flex gap-4">
              <n-popconfirm @positive-click="handleTruncate">
                <template #trigger>
                  <n-button type="warning" ghost :loading="loadingTruncate">
                    <template #icon><n-icon :component="renderIcon('mdi:delete-sweep-outline')" /></template>
                    清空数据 (TRUNCATE)
                  </n-button>
                </template>
                确定要清空表中的所有数据吗？此操作不可恢复！
              </n-popconfirm>

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
          </n-card>

          <n-card title="表选项" size="small" class="shadow-sm">
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
