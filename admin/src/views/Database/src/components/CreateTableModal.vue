<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NModal, NInput, NSelect, NButton, NIcon, NSwitch, NInputNumber, useMessage } from 'naive-ui'
import { renderIcon } from '@/utils'
import { useI18n } from 'vue-i18n'
import { createDBManagerTableAPI } from '@/api/modules/database'

const props = defineProps<{
  serverId: number | null
  serverType?: string
  databaseName: string | null
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

const message = useMessage()
const { t } = useI18n()
const show = defineModel<boolean>('show', { default: false })

const tableName = ref('')
const tableComment = ref('')
const engine = ref('InnoDB')
const charset = ref('utf8mb4')
const collation = ref('utf8mb4_unicode_ci')
const submitting = ref(false)

interface ColumnRow {
  key: number
  name: string
  type: string
  length: string
  nullable: boolean
  defaultValue: string
  autoIncrement: boolean
  comment: string
  isPrimary: boolean
}

let columnKeyCounter = 0
const columns = ref<ColumnRow[]>([])

const engineOptions = [
  { label: t('databaseManager.createTable.engineRecommended'), value: 'InnoDB' },
  { label: 'MyISAM', value: 'MyISAM' },
  { label: 'MEMORY', value: 'MEMORY' },
]

const charsetOptions = [
  { label: t('databaseManager.createTable.charsetRecommended'), value: 'utf8mb4' },
  { label: 'utf8', value: 'utf8' },
  { label: 'latin1', value: 'latin1' },
  { label: 'gbk', value: 'gbk' },
]

const collationOptions = [
  { label: t('databaseManager.createTable.collationRecommended'), value: 'utf8mb4_unicode_ci' },
  { label: 'utf8mb4_general_ci', value: 'utf8mb4_general_ci' },
  { label: 'utf8mb4_bin', value: 'utf8mb4_bin' },
  { label: 'utf8_general_ci', value: 'utf8_general_ci' },
]

const columnTypeOptions = [
  { label: 'INT', value: 'INT' },
  { label: 'BIGINT', value: 'BIGINT' },
  { label: 'SMALLINT', value: 'SMALLINT' },
  { label: 'TINYINT', value: 'TINYINT' },
  { label: 'VARCHAR', value: 'VARCHAR' },
  { label: 'CHAR', value: 'CHAR' },
  { label: 'TEXT', value: 'TEXT' },
  { label: 'LONGTEXT', value: 'LONGTEXT' },
  { label: 'MEDIUMTEXT', value: 'MEDIUMTEXT' },
  { label: 'BLOB', value: 'BLOB' },
  { label: 'DECIMAL', value: 'DECIMAL' },
  { label: 'FLOAT', value: 'FLOAT' },
  { label: 'DOUBLE', value: 'DOUBLE' },
  { label: 'DATE', value: 'DATE' },
  { label: 'DATETIME', value: 'DATETIME' },
  { label: 'TIMESTAMP', value: 'TIMESTAMP' },
  { label: 'BOOLEAN', value: 'BOOLEAN' },
  { label: 'JSON', value: 'JSON' },
  { label: 'ENUM', value: 'ENUM' },
]

const needsLength = computed(() => {
  const lengthTypes = new Set(['VARCHAR', 'CHAR', 'INT', 'BIGINT', 'SMALLINT', 'TINYINT', 'DECIMAL', 'FLOAT', 'DOUBLE'])
  return (col: ColumnRow) => lengthTypes.has(col.type.toUpperCase())
})

const addColumn = () => {
  columnKeyCounter++
  columns.value.push({
    key: columnKeyCounter,
    name: '',
    type: 'VARCHAR',
    length: '255',
    nullable: true,
    defaultValue: '',
    autoIncrement: false,
    comment: '',
    isPrimary: false,
  })
}

const removeColumn = (key: number) => {
  columns.value = columns.value.filter(c => c.key !== key)
}

const moveColumnUp = (index: number) => {
  if (index <= 0) return
  const temp = columns.value[index]
  columns.value[index] = columns.value[index - 1]
  columns.value[index - 1] = temp
}

const moveColumnDown = (index: number) => {
  if (index >= columns.value.length - 1) return
  const temp = columns.value[index]
  columns.value[index] = columns.value[index + 1]
  columns.value[index + 1] = temp
}

const hasPrimaryKey = computed(() => columns.value.some(c => c.isPrimary))

const handleReset = () => {
  tableName.value = ''
  tableComment.value = ''
  engine.value = 'InnoDB'
  charset.value = 'utf8mb4'
  collation.value = 'utf8mb4_unicode_ci'
  columns.value = []
  columnKeyCounter = 0
}

const handleSubmit = async () => {
  if (!props.serverId || !props.databaseName) {
    message.warning(t('databaseManager.createTable.selectServerAndDbFirst'))
    return
  }

  const name = tableName.value.trim()
  if (!name) {
    message.warning(t('databaseManager.createTable.tableNameRequired'))
    return
  }
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(name)) {
    message.warning(t('databaseManager.createTable.tableNameFormatInvalid'))
    return
  }

  if (columns.value.length === 0) {
    message.warning(t('databaseManager.createTable.addFieldFirst'))
    return
  }

  for (const col of columns.value) {
    if (!col.name.trim()) {
      message.warning(t('databaseManager.createTable.fieldNameRequired'))
      return
    }
    if (!col.type.trim()) {
      message.warning(t('databaseManager.createTable.fieldTypeRequired', { name: col.name }))
      return
    }
  }

  submitting.value = true
  try {
    const payload: any = {
      serverId: props.serverId,
      databaseName: props.databaseName,
      tableName: name,
      columns: columns.value.map(col => ({
        name: col.name.trim(),
        type: col.length ? `${col.type}(${col.length})` : col.type,
        nullable: col.nullable,
        defaultValue: col.defaultValue || '',
        autoIncrement: col.autoIncrement,
        comment: col.comment || '',
        isPrimary: col.isPrimary,
      })),
    }

    if (props.serverType === 'mysql' || props.serverType === 'mariadb') {
      payload.engine = engine.value
      payload.charset = charset.value
      payload.collation = collation.value
      if (tableComment.value) payload.comment = tableComment.value
    }

    const res = await createDBManagerTableAPI(payload)
    if (res.code === 0) {
      message.success(t('databaseManager.createTable.success', { name }))
      show.value = false
      emit('success')
    } else {
      message.error(res.message || t('databaseManager.createTable.failed'))
    }
  } catch (error: any) {
    void 0
  } finally {
    submitting.value = false
  }
}

watch(show, (val) => {
  if (!val) handleReset()
  if (val && columns.value.length === 0) addColumn()
})
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    style="width: 900px; max-height: 85vh;"
    :title="t('databaseManager.createTable.title')"
    :mask-closable="false"
  >
    <div class="flex flex-col gap-4 text-sm max-h-[65vh] overflow-y-auto pr-1">
      <!-- 基本信息 -->
      <div class="grid grid-cols-2 gap-4">
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600 shrink-0">{{ t('databaseManager.createTable.tableNameLabel') }}</span>
          <n-input
            v-model:value="tableName"
            :placeholder="t('databaseManager.createTable.tableNamePlaceholder')"
            size="small"
            clearable
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600 shrink-0">{{ t('databaseManager.createTable.commentLabel') }}</span>
          <n-input
            v-model:value="tableComment"
            :placeholder="t('databaseManager.createTable.commentPlaceholder')"
            size="small"
            clearable
          />
        </div>
        <template v-if="serverType === 'mysql' || serverType === 'mariadb'">
          <div class="flex items-center gap-2">
            <span class="w-20 text-right text-slate-600 shrink-0">引擎:</span>
            <n-select v-model:value="engine" :options="engineOptions" size="small" />
          </div>
          <div class="flex items-center gap-2">
            <span class="w-20 text-right text-slate-600 shrink-0">字符集:</span>
            <n-select v-model:value="charset" :options="charsetOptions" size="small" />
          </div>
        </template>
      </div>

      <!-- 字段列表 -->
      <div class="border border-slate-200 rounded overflow-hidden">
        <div class="bg-slate-100 px-2 py-1.5 font-bold text-xs flex items-center gap-2 border-b border-slate-200">
          <n-icon :component="renderIcon('mdi:table-column')" />
          <span>字段定义</span>
          <span class="text-slate-400 font-normal">({{ columns.length }} 个字段)</span>
        </div>

        <!-- 表头 -->
        <div class="bg-[#f0f4f8] border-b border-slate-200 px-1 py-1 text-[11px] font-semibold text-slate-600 flex items-center gap-1">
          <div class="w-8 shrink-0 text-center">#</div>
          <div class="w-7 shrink-0"></div>
          <div class="w-28 shrink-0">字段名</div>
          <div class="w-40 shrink-0">类型</div>
          <div class="w-14 shrink-0 text-center">长度</div>
          <div class="w-14 shrink-0 text-center">可空</div>
          <div class="w-14 shrink-0 text-center">自增</div>
          <div class="w-14 shrink-0 text-center">主键</div>
          <div class="w-28 shrink-0">默认值</div>
          <div class="flex-1 min-w-0">备注</div>
        </div>

        <!-- 行 -->
        <div
          v-for="(col, index) in columns"
          :key="col.key"
          class="flex items-center gap-1 px-1 py-1 border-b border-slate-100 hover:bg-slate-50 text-xs"
        >
          <div class="w-8 shrink-0 text-center text-slate-400 font-mono">{{ index + 1 }}</div>
          <div class="w-7 shrink-0 flex gap-0.5">
            <n-button size="tiny" quaternary @click="moveColumnUp(index)" :disabled="index === 0">
              <template #icon><n-icon :component="renderIcon('mdi:chevron-up')" /></template>
            </n-button>
            <n-button size="tiny" quaternary @click="moveColumnDown(index)" :disabled="index >= columns.length - 1">
              <template #icon><n-icon :component="renderIcon('mdi:chevron-down')" /></template>
            </n-button>
          </div>
          <div class="w-28 shrink-0">
            <n-input v-model:value="col.name" placeholder="字段名" size="tiny" />
          </div>
          <div class="w-40 shrink-0">
            <n-select
              v-model:value="col.type"
              :options="columnTypeOptions"
              size="tiny"
              :show-checkmark="false"
              filterable
              :input-props="{ style: 'font-size: 12px' }"
            />
          </div>
          <div class="w-14 shrink-0 text-center">
            <n-input
              v-if="needsLength(col)"
              v-model:value="col.length"
              placeholder="255"
              size="tiny"
              style="width: 50px; text-align: center;"
            />
          </div>
          <div class="w-14 shrink-0 flex justify-center">
            <n-switch v-model:value="col.nullable" size="small" />
          </div>
          <div class="w-14 shrink-0 flex justify-center">
            <n-switch v-model:value="col.autoIncrement" size="small" :disabled="col.type.toUpperCase() !== 'INT' && col.type.toUpperCase() !== 'BIGINT'" />
          </div>
          <div class="w-14 shrink-0 flex justify-center">
            <n-switch v-model:value="col.isPrimary" size="small" />
          </div>
          <div class="w-28 shrink-0">
            <n-input v-model:value="col.defaultValue" placeholder="NULL" size="tiny" />
          </div>
          <div class="flex-1 min-w-0 flex gap-1">
            <n-input v-model:value="col.comment" placeholder="备注" size="tiny" class="flex-1" />
            <n-button size="tiny" quaternary type="error" @click="removeColumn(col.key)">
              <template #icon><n-icon :component="renderIcon('mdi:close')" /></template>
            </n-button>
          </div>
        </div>
      </div>

      <n-button size="tiny" quaternary @click="addColumn" class="self-start">
        <template #icon><n-icon :component="renderIcon('mdi:plus')" /></template>
        添加字段
      </n-button>

      <div v-if="!hasPrimaryKey" class="text-xs text-amber-600 bg-amber-50 px-2 py-1 rounded">
        <n-icon :component="renderIcon('mdi:alert-circle-outline')" class="mr-1" />
        建议为表设置主键字段
      </div>
    </div>

    <template #footer>
      <div class="flex justify-between items-center">
        <div class="text-xs text-slate-400">
          共 {{ columns.length }} 个字段
        </div>
        <div class="flex gap-2">
          <n-button size="small" @click="show = false">取消</n-button>
          <n-button
            size="small"
            type="primary"
            :loading="submitting"
            @click="handleSubmit"
          >创建表</n-button>
        </div>
      </div>
    </template>
  </n-modal>
</template>
