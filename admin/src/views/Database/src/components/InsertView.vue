<script setup lang="ts">
import { ref } from 'vue'
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
const savingRecord = ref(false)
const localRecordData = ref({ ...props.recordData })

const submitRecord = async () => {
  if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
  
  savingRecord.value = true
  try {
    const submitData = { ...localRecordData.value }
    props.structureData.forEach(col => {
      if (submitData[col.Field] === '' && !props.isEditing && col.Key === 'PRI') {
        delete submitData[col.Field]
      }
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
      message.success(props.isEditing ? '更新成功' : '插入成功')
      emit('success')
    } else {
      message.error(res.message || '保存失败')
    }
  } catch (error) {
    message.error('保存请求失败')
  } finally {
    savingRecord.value = false
  }
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
          <span class="mr-2">{{ selectedServerId ? serverOptions.find(s => s.value === selectedServerId)?.label : '' }}</span>
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
            <div class="w-1/4">字段 (Field)</div>
            <div class="w-1/4">类型 (Type)</div>
            <div class="w-1/2">值 (Value)</div>
          </div>
          <div
            v-for="(col, index) in structureData"
            :key="index"
            class="flex border-b border-slate-100 p-2 items-center text-xs hover:bg-slate-50"
          >
            <div class="w-1/4 font-semibold text-slate-700 break-all pr-2">
              {{ col.Field }}
              <span
                v-if="col.Key === 'PRI'"
                class="text-amber-500 ml-1"
                title="Primary Key"
              >🔑</span>
            </div>
            <div class="w-1/4 text-slate-500 break-all pr-2">{{ col.Type }}</div>
            <div class="w-1/2">
              <n-input
                v-if="col.Type && (col.Type.includes('text') || col.Type.includes('char') || col.Type.includes('json'))"
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
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>