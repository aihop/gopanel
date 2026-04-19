<script setup lang="ts">
import { ref, watch } from 'vue'
import { NInput, NSelect, NButton, NIcon, NEmpty, NSpin } from 'naive-ui'
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
  (e: 'search', conditions: any[]): void
  (e: 'cancel'): void
}>()

const searchData = ref<Record<string, { operator: string, value: string }>>({})

const operatorOptions = [
  { label: 'LIKE %...%', value: 'LIKE' },
  { label: '=', value: '=' },
  { label: '!=', value: '!=' },
  { label: '<', value: '<' },
  { label: '>', value: '>' },
  { label: '<=', value: '<=' },
  { label: '>=', value: '>=' },
  { label: 'IS NULL', value: 'IS NULL' },
  { label: 'IS NOT NULL', value: 'IS NOT NULL' }
]

watch(() => props.structureData, (newVal) => {
  const data: Record<string, { operator: string, value: string }> = {}
  newVal.forEach(col => {
    data[col.Field] = {
      operator: col.Type && (col.Type.includes('text') || col.Type.includes('char')) ? 'LIKE' : '=',
      value: ''
    }
  })
  searchData.value = data
}, { immediate: true, deep: true })

const handleSearch = () => {
  const conditions: any[] = []
  for (const [key, item] of Object.entries(searchData.value)) {
    if (item.operator === 'IS NULL' || item.operator === 'IS NOT NULL') {
      conditions.push({ column: key, operator: item.operator, value: '' })
    } else if (item.value !== '') {
      conditions.push({ column: key, operator: item.operator, value: item.value })
    }
  }
  emit('search', conditions)
}

const handleClear = () => {
  Object.keys(searchData.value).forEach(key => {
    searchData.value[key].value = ''
  })
}
</script>

<template>
  <div class="flex-1 flex flex-col relative overflow-hidden">
    <div v-if="!selectedTable" class="flex-1 flex items-center justify-center text-slate-400">
      <n-empty :description="$t('database.selectTable')" />
    </div>
    <div v-else-if="loadingStructure" class="flex-1 flex items-center justify-center">
      <n-spin size="medium" />
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
          <span class="font-bold">{{ selectedTable }} (多字段搜索)</span>
        </div>
        <div class="flex gap-2">
          <n-button size="tiny" @click="handleClear">清空条件</n-button>
          <n-button size="tiny" type="primary" @click="handleSearch">执行搜索</n-button>
        </div>
      </div>
      <div class="flex-1 overflow-auto p-4 bg-white">
        <div class="max-w-4xl mx-auto border border-slate-200">
          <div class="bg-[#e9ecef] border-b border-slate-200 p-2 font-bold text-xs flex">
            <div class="w-1/4">字段 (Field)</div>
            <div class="w-1/4">类型 (Type)</div>
            <div class="w-1/4">操作符 (Operator)</div>
            <div class="w-1/4">值 (Value)</div>
          </div>
          <div v-for="(col, index) in structureData" :key="index" class="flex border-b border-slate-100 p-2 items-center text-xs hover:bg-slate-50">
            <div class="w-1/4 font-semibold text-slate-700 break-all pr-2">{{ col.Field }}</div>
            <div class="w-1/4 text-slate-500 break-all pr-2">{{ col.Type }}</div>
            <div class="w-1/4 pr-2">
              <n-select v-model:value="searchData[col.Field].operator" :options="operatorOptions" size="small" />
            </div>
            <div class="w-1/4">
              <n-input
                v-model:value="searchData[col.Field].value"
                :disabled="searchData[col.Field].operator === 'IS NULL' || searchData[col.Field].operator === 'IS NOT NULL'"
                type="text"
                size="small"
                @keyup.enter="handleSearch"
              />
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
