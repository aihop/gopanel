<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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

const selectedServerLabel = computed(() => {
  return props.selectedServerId
    ? props.serverOptions.find(s => s.value === props.selectedServerId)?.label || ''
    : ''
})

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

const fieldRows = computed(() => {
  return (props.structureData || []).map((col: any) => ({
    ...col,
    isTextLike: String(col.Type || '').toLowerCase().includes('text') || String(col.Type || '').toLowerCase().includes('char'),
    isPrimary: col.Key === 'PRI'
  }))
})

const activeConditionCount = computed(() => {
  return Object.values(searchData.value).filter((item) => {
    return item.operator === 'IS NULL' || item.operator === 'IS NOT NULL' || item.value !== ''
  }).length
})

const searchSummary = computed(() => {
  return {
    total: fieldRows.value.length,
    primaryCount: fieldRows.value.filter(row => row.isPrimary).length,
    textCount: fieldRows.value.filter(row => row.isTextLike).length
  }
})

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
    const column = props.structureData.find((col: any) => col.Field === key)
    const isText = column?.Type && (column.Type.includes('text') || column.Type.includes('char'))
    searchData.value[key].operator = isText ? 'LIKE' : '='
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
          <span class="mr-2">{{ selectedServerLabel }}</span>
          <span>»</span>
          <n-icon :component="renderIcon('mdi:database')" class="ml-2" />
          <span class="mr-2">{{ selectedDatabase }}</span>
          <span>»</span>
          <n-icon :component="renderIcon('mdi:table')" class="ml-2" />
          <span class="font-bold">{{ selectedTable }} (多字段搜索)</span>
        </div>
        <div class="flex gap-2">
          <div class="hidden xl:flex items-center gap-2 text-[11px] text-slate-500 mr-2">
            <span class="px-2 py-1 rounded bg-slate-100">{{ searchSummary.total }} 个字段</span>
            <span class="px-2 py-1 rounded bg-slate-100">{{ searchSummary.primaryCount }} 个主键列</span>
            <span class="px-2 py-1 rounded bg-blue-50 text-blue-600">{{ activeConditionCount }} 个生效条件</span>
          </div>
          <n-button size="tiny" @click="handleClear">清空条件</n-button>
          <n-button size="tiny" type="primary" @click="handleSearch">执行搜索</n-button>
        </div>
      </div>
      <div class="flex-1 overflow-auto p-4 bg-white">
        <div class="max-w-4xl mx-auto border border-slate-200">
          <div class="border-b border-slate-200 bg-slate-50 px-3 py-2 text-[11px] text-slate-500 flex items-center gap-2">
            <span>文本字段默认使用 `LIKE` 模糊匹配</span>
            <span>·</span>
            <span>数值字段默认使用精确匹配</span>
            <span>·</span>
            <span>`IS NULL` / `IS NOT NULL` 无需填写值</span>
          </div>
          <div class="bg-[#e9ecef] border-b border-slate-200 p-2 font-bold text-xs flex">
            <div class="w-1/4">字段 (Field)</div>
            <div class="w-1/4">类型 (Type)</div>
            <div class="w-1/4">操作符 (Operator)</div>
            <div class="w-1/4">值 (Value)</div>
          </div>
          <div
            v-for="(col, index) in fieldRows"
            :key="`${selectedTable || 'table'}-${col.Field || index}`"
            class="flex border-b border-slate-100 p-2 items-center text-xs hover:bg-slate-50"
          >
            <div class="w-1/4 font-semibold text-slate-700 break-all pr-2">
              {{ col.Field }}
              <span
                v-if="col.isPrimary"
                class="ml-1 px-1.5 py-0.5 rounded bg-amber-50 text-amber-600 text-[10px]"
              >主键</span>
            </div>
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
