<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, NEmpty, NIcon, NButton, NSpin, NInput } from 'naive-ui'
import { insertDBManagerRecordAPI, updateDBManagerRecordAPI } from '@/api/modules/database'
import { renderIcon } from '@/utils'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
  structureData: any[]
  loadingStructure: boolean
  isEditing: boolean
  recordData: Record<string, any>
  originalRecordData: Record<string, any>
}>()

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'success'): void
}>()

const message = useMessage()
const { t } = useI18n()
const savingRecord = ref(false)
const localRecordData = ref<Record<string, any>>({})

const selectedServerLabel = computed(() => {
  return props.selectedServerId
    ? props.serverOptions.find(s => s.value === props.selectedServerId)?.label || ''
    : ''
})

const fieldRows = computed(() => {
  return (props.structureData || []).map((col: any) => {
    const type = String(col.Type || '').toLowerCase()
    const extra = String(col.Extra || '').toLowerCase()
    const nullable = col.Null === 'YES' || col.Null === 1 || col.Null === true
    const defaultValue = col.Default
    return {
      ...col,
      nullable,
      defaultValue,
      isPrimary: col.Key === 'PRI',
      isAutoIncrement: extra.includes('auto_increment') || extra.includes('identity') || type.includes('serial'),
      isTextLike: type.includes('text') || type.includes('char') || type.includes('json'),
      isJsonType: type.includes('json'),
      typeLabel: col.Type || '-'
    }
  })
})

const formSummary = computed(() => {
  const rows = fieldRows.value
  return {
    total: rows.length,
    primaryCount: rows.filter(row => row.isPrimary).length,
    autoIncrementCount: rows.filter(row => row.isAutoIncrement).length,
    nullableCount: rows.filter(row => row.nullable).length
  }
})

const rebuildLocalRecordData = () => {
  const nextRecord: Record<string, any> = {}
  const columns = Array.isArray(props.structureData) ? props.structureData : []

  if (columns.length > 0) {
    columns.forEach((col: any) => {
      const field = col?.Field
      if (!field) return
      nextRecord[field] = props.recordData?.[field] ?? ''
    })
  } else {
    Object.assign(nextRecord, props.recordData || {})
  }

  localRecordData.value = nextRecord
}

watch(
  () => [props.recordData, props.structureData, props.selectedTable, props.isEditing],
  rebuildLocalRecordData,
  { deep: true, immediate: true }
)

const submitRecord = async () => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  
  savingRecord.value = true
  try {
    const submitData: Record<string, any> = {}
    fieldRows.value.forEach((col: any) => {
      const field = col.Field
      let value = localRecordData.value[field]

      if (!props.isEditing && col.isAutoIncrement && (value === '' || value === null || value === undefined)) {
        return
      }

      if (col.isJsonType && value === '') {
        value = null
      }

      submitData[field] = value
    })

    let res;
    if (props.isEditing) {
      res = await updateDBManagerRecordAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        data: submitData,
        conditions: props.originalRecordData
      })
    } else {
      res = await insertDBManagerRecordAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        tableName: props.selectedTable,
        data: submitData
      })
    }

    if (res.code === 0) {
      message.success(t(props.isEditing ? 'database.recordUpdateSuccess' : 'database.recordInsertSuccess'))
      emit('success')
    } else {
      message.error(res.message || t('database.recordSaveFailed'))
    }
  } catch (error: any) {
  } finally {
    savingRecord.value = false
  }
}

const applyFieldDefault = (field: string, value: any) => {
  localRecordData.value[field] = value ?? ''
}

const applyFieldNull = (field: string) => {
  localRecordData.value[field] = null
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
    <div
      v-else-if="loadingStructure"
      class="flex-1 flex items-center justify-center"
    >
      <n-spin size="medium" />
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
            :component="renderIcon('mdi:table')"
            class="ml-2"
          />
          <span class="font-bold">{{ selectedTable }} ({{ isEditing ? '编辑记录' : '插入记录' }})</span>
        </div>
        <div class="flex gap-2">
          <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500 mr-2">
            <span class="px-2 py-1 rounded bg-slate-100">{{ formSummary.total }} 个字段</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ formSummary.primaryCount }} 个主键列</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ formSummary.autoIncrementCount }} 个自增列</span>
          </div>
          <n-button
            size="tiny"
            @click="emit('cancel')"
          >取消</n-button>
          <n-button
            size="tiny"
            type="primary"
            @click="submitRecord"
            :loading="savingRecord"
          >保存</n-button>
        </div>
      </div>
      <div class="flex-1 overflow-auto p-4 bg-white">
        <div class="max-w-4xl mx-auto border border-slate-200">
          <div class="bg-[#e9ecef] border-b border-slate-200 p-2 font-bold text-xs flex">
            <div class="w-[22%]">字段 (Field)</div>
            <div class="w-[24%]">类型 / 约束</div>
            <div class="w-[54%]">值 (Value)</div>
          </div>
          <div
            v-for="(col, index) in fieldRows"
            :key="`${selectedTable || 'table'}-${col.Field || index}`"
            class="flex border-b border-slate-100 p-2 items-center text-xs hover:bg-slate-50"
          >
            <div class="w-[22%] font-semibold text-slate-700 break-all pr-2">
              {{ col.Field }}
              <div class="mt-1 flex flex-wrap gap-1">
                <span
                  v-if="col.isPrimary"
                  class="px-1.5 py-0.5 rounded bg-amber-50 text-amber-600 text-[10px]"
                >主键</span>
                <span
                  v-if="col.isAutoIncrement"
                  class="px-1.5 py-0.5 rounded bg-sky-50 text-sky-600 text-[10px]"
                >自增</span>
                <span
                  v-if="col.nullable"
                  class="px-1.5 py-0.5 rounded bg-slate-100 text-slate-500 text-[10px]"
                >可空</span>
              </div>
            </div>
            <div class="w-[24%] text-slate-500 break-all pr-2">
              <div>{{ col.typeLabel }}</div>
              <div
                v-if="col.defaultValue !== undefined && col.defaultValue !== null && col.defaultValue !== ''"
                class="mt-1 text-[10px] text-slate-400"
              >
                默认值: {{ col.defaultValue }}
              </div>
            </div>
            <div class="w-[54%]">
              <n-input
                v-if="col.isTextLike"
                v-model:value="localRecordData[col.Field]"
                type="textarea"
                :autosize="{ minRows: 1, maxRows: 5 }"
                size="small"
              />
              <n-input
                v-else
                v-model:value="localRecordData[col.Field]"
                type="text"
                size="small"
              />
              <div class="mt-1 flex flex-wrap gap-2 text-[10px]">
                <button
                  v-if="col.defaultValue !== undefined && col.defaultValue !== null && col.defaultValue !== ''"
                  type="button"
                  class="text-blue-600 hover:text-blue-700"
                  @click="applyFieldDefault(col.Field, col.defaultValue)"
                >
                  填入默认值
                </button>
                <button
                  v-if="col.nullable"
                  type="button"
                  class="text-slate-500 hover:text-slate-700"
                  @click="applyFieldNull(col.Field)"
                >
                  设为 NULL
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
