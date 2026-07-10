<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useMessage, useDialog, NEmpty, NButton, NInput, NInputGroup, NCard, NPopconfirm, NIcon, NModal, NCheckbox } from 'naive-ui'
import { renderIcon } from '@/utils'
import { execDBManagerSqlAPI, copyDBManagerTableAPI } from '@/api/modules/database'
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
const dialog = useDialog()
const newTableName = ref('')
const loadingTruncate = ref(false)
const loadingDrop = ref(false)
const loadingRename = ref(false)

// Copy table state
const showCopyModal = ref(false)
const copyTargetName = ref('')
const copyWithData = ref(true)
const loadingCopy = ref(false)

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
      message.error(res.msg || '清空失败')
    }
  } catch (err: any) {
    message.error('请求失败')
  } finally {
    loadingTruncate.value = false
  }
}

// Postgres 在表被其他对象（视图、外键等）依赖时会拒绝 DROP TABLE，这里识别这个特定报错，
// 询问用户是否改用 CASCADE 连带删除依赖对象，而不是默认就带上 CASCADE 静默删掉更多东西
const isDependencyError = (msg: string) => /depend(s|ency)? on it|dependent objects/i.test(msg || '')

const dropTableCascade = async () => {
  if (!props.selectedTable) return
  loadingDrop.value = true
  try {
    const q = getQuote()
    const res = await execSql(`DROP TABLE ${q}${props.selectedTable}${q} CASCADE`)
    if (res.code === 0) {
      message.success('表及其依赖对象已删除')
      emit('tableDropped')
    } else {
      message.error(res.msg || '删除失败')
    }
  } catch {
    message.error('请求失败')
  } finally {
    loadingDrop.value = false
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
    } else if (serverType.value === 'postgresql' && isDependencyError(res.msg)) {
      loadingDrop.value = false
      dialog.warning({
        title: '存在依赖对象',
        content: `表 ${props.selectedTable} 被其他数据库对象（如视图、外键）依赖，无法直接删除。是否级联删除（CASCADE）？这会连带删除所有依赖它的对象，此操作不可恢复！`,
        positiveText: '级联删除',
        negativeText: '取消',
        onPositiveClick: dropTableCascade
      })
      return
    } else {
      message.error(res.msg || '删除失败')
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
      message.error(res.msg || '重命名失败')
    }
  } catch (err: any) {
    message.error('请求失败')
  } finally {
    loadingRename.value = false
  }
}

const handleCopy = async () => {
  if (!props.selectedTable || !copyTargetName.value.trim()) {
    message.warning('请输入目标表名')
    return
  }
  const name = copyTargetName.value.trim()
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(name)) {
    message.warning('表名格式不正确，请使用字母、数字或下划线')
    return
  }
  if (name === props.selectedTable) {
    message.warning('目标表名不能与原表名相同')
    return
  }

  loadingCopy.value = true
  try {
    const res = await copyDBManagerTableAPI({
      serverId: props.selectedServerId!,
      databaseName: props.selectedDatabase!,
      sourceTable: props.selectedTable,
      targetTable: name,
      copyData: copyWithData.value
    })
    if (res.code === 0) {
      message.success(`表已复制到 ${name}`)
      showCopyModal.value = false
      copyTargetName.value = ''
      emit('tableRenamed', name)
    } else {
      message.error(res.msg || '复制失败')
    }
  } catch (err: any) {
    message.error(err?.message || '复制请求失败')
  } finally {
    loadingCopy.value = false
  }
}

// 表维护操作（OPTIMIZE / CHECK / ANALYZE / REPAIR）
const maintainResults = ref<{ operation: string; type: string; text: string }[]>([])
const loadingOptimize = ref(false)
const loadingCheck = ref(false)
const loadingAnalyze = ref(false)
const loadingRepair = ref(false)

const maintainOps = computed(() => {
  const t = serverType.value
  const ops: { key: string; label: string; icon: string; desc: string }[] = []
  ops.push({ key: 'optimize', label: 'OPTIMIZE', icon: 'mdi:database-cog-outline', desc: '整理表空间碎片，回收未使用空间' })
  ops.push({ key: 'check', label: 'CHECK', icon: 'mdi:shield-check-outline', desc: '检查表结构完整性' })
  ops.push({ key: 'analyze', label: 'ANALYZE', icon: 'mdi:chart-line', desc: '更新表统计信息，优化查询计划' })
  if (t === 'mysql' || t === 'mariadb') {
    ops.push({ key: 'repair', label: 'REPAIR', icon: 'mdi:wrench', desc: '修复损坏的表（仅 MySQL/MariaDB）' })
  }
  return ops
})

const getMaintainSql = (operation: string): string | null => {
  const q = getQuote()
  const table = `${q}${props.selectedTable}${q}`
  const t = serverType.value
  switch (operation) {
    case 'optimize':
      if (t === 'mysql' || t === 'mariadb') return `OPTIMIZE TABLE ${table}`
      if (t === 'postgresql') return `VACUUM ANALYZE ${table}`
      if (t === 'sqlite') return `PRAGMA optimize`
      return null
    case 'check':
      if (t === 'mysql' || t === 'mariadb') return `CHECK TABLE ${table}`
      if (t === 'sqlite') return `PRAGMA integrity_check`
      return null
    case 'analyze':
      if (t === 'mysql' || t === 'mariadb') return `ANALYZE TABLE ${table}`
      if (t === 'postgresql') return `ANALYZE ${table}`
      if (t === 'sqlite') return `ANALYZE`
      return null
    case 'repair':
      if (t === 'mysql' || t === 'mariadb') return `REPAIR TABLE ${table}`
      return null
    default:
      return null
  }
}

const handleMaintain = async (operation: string) => {
  if (!props.selectedTable) {
    message.warning('请先选择表')
    return
  }
  const sql = getMaintainSql(operation)
  if (!sql) {
    message.warning(`当前数据库类型不支持 ${operation.toUpperCase()} 操作`)
    return
  }
  const loadingMap: Record<string, any> = {
    optimize: loadingOptimize,
    check: loadingCheck,
    analyze: loadingAnalyze,
    repair: loadingRepair
  }
  loadingMap[operation].value = true
  try {
    const res = await execSql(sql)
    if (res.code === 0) {
      const result = res.data || {}
      if (result.type === 'query' && Array.isArray(result.rows)) {
        maintainResults.value = result.rows.map((row: any) => ({
          operation: operation.toUpperCase(),
          type: row.Msg_type || row.type || '-',
          text: row.Msg_text || row.text || JSON.stringify(row)
        }))
      } else {
        maintainResults.value = [{
          operation: operation.toUpperCase(),
          type: 'OK',
          text: `${operation.toUpperCase()} 执行成功`
        }]
      }
      message.success(`${operation.toUpperCase()} 执行成功`)
    } else {
      maintainResults.value = [{
        operation: operation.toUpperCase(),
        type: 'ERROR',
        text: res.msg || '操作失败'
      }]
      message.error(res.msg || `${operation.toUpperCase()} 执行失败`)
    }
  } catch (err: any) {
    maintainResults.value = [{
      operation: operation.toUpperCase(),
      type: 'ERROR',
      text: err?.message || '请求失败'
    }]
    message.error(err?.message || '请求失败')
  } finally {
    loadingMap[operation].value = false
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

          <n-card title="复制表" size="small" class="shadow-sm">
            <div class="text-xs text-slate-500 mb-3">
              复制当前表的结构和数据到一张新表。新表将与原表结构一致。
            </div>
            <n-button type="primary" ghost @click="showCopyModal = true">
              <template #icon><n-icon :component="renderIcon('mdi:content-copy')" /></template>
              复制表
            </n-button>
          </n-card>

          <n-card title="维护工具" size="small" class="shadow-sm">
            <div class="text-xs text-slate-500 mb-3">
              执行表级维护操作：优化表空间、检查完整性、更新统计信息或修复损坏表。
            </div>
            <div class="grid grid-cols-2 xl:grid-cols-4 gap-3">
              <div
                v-for="op in maintainOps"
                :key="op.key"
                class="rounded border border-slate-200 p-3 hover:border-sky-300 hover:bg-sky-50 transition-colors"
              >
                <div class="font-semibold text-slate-700 mb-1 text-xs">{{ op.label }}</div>
                <div class="text-[11px] text-slate-400 mb-3 leading-tight">{{ op.desc }}</div>
                <n-button
                  size="tiny"
                  :type="op.key === 'check' ? 'info' : 'default'"
                  ghost
                  :loading="({ optimize: loadingOptimize, check: loadingCheck, analyze: loadingAnalyze, repair: loadingRepair } as any)[op.key]"
                  @click="handleMaintain(op.key)"
                >
                  <template #icon><n-icon :component="renderIcon(op.icon)" /></template>
                  执行
                </n-button>
              </div>
            </div>

            <div v-if="maintainResults.length > 0" class="mt-4">
              <div class="border-t border-slate-200 pt-3">
                <div class="text-xs font-semibold text-slate-600 mb-2">执行结果</div>
                <div class="flex flex-col gap-1">
                  <div
                    v-for="(r, idx) in maintainResults"
                    :key="idx"
                    class="flex items-start gap-2 text-xs px-3 py-1.5 rounded"
                    :class="r.type === 'ERROR' ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700'"
                  >
                    <span class="font-mono font-semibold whitespace-nowrap">[{{ r.operation }}]</span>
                    <span class="font-mono text-[11px]">{{ r.text }}</span>
                  </div>
                </div>
              </div>
            </div>
          </n-card>

        </div>
      </div>
    </template>
  </div>

  <!-- Copy Table Modal -->
  <n-modal
    v-model:show="showCopyModal"
    preset="card"
    style="width: 460px"
    title="复制表"
  >
    <div class="flex flex-col gap-4 text-sm">
      <div class="flex items-center gap-2">
        <span class="w-24 text-right text-slate-600">源表名:</span>
        <span class="font-semibold text-slate-800">{{ selectedTable }}</span>
      </div>
      <div class="flex items-center gap-2">
        <span class="w-24 text-right text-slate-600">目标表名:</span>
        <n-input
          v-model:value="copyTargetName"
          placeholder="输入新表名..."
          size="small"
          class="flex-1"
          clearable
        />
      </div>
      <div class="flex items-center gap-2">
        <span class="w-24 text-right text-slate-600">&nbsp;</span>
        <n-checkbox v-model:checked="copyWithData">
          复制数据（含所有记录）
        </n-checkbox>
      </div>
      <div class="text-xs text-slate-400 mt-1">
        若不勾选"复制数据"，则仅复制表结构（字段、索引、约束），不复制记录。
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <n-button size="small" @click="showCopyModal = false">取消</n-button>
        <n-button
          size="small"
          type="primary"
          :loading="loadingCopy"
          @click="handleCopy"
        >开始复制</n-button>
      </div>
    </template>
  </n-modal>
</template>
