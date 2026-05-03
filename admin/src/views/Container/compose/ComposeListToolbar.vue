<template>
  <div class="mb-4 flex items-center justify-between">
    <n-button
      type="primary"
      @click="emit('create')"
    >
      创建编排
    </n-button>
    <n-space align="center">
      <n-popover
        trigger="click"
        placement="bottom-start"
        :width="300"
      >
        <template #trigger>
          <div class="rounded-full border border-gray-200 bg-base-100 px-5 py-2 text-sm">列表设置</div>
        </template>
        <div class="flex items-center gap-4 text-nowrap bg-base-100">
          刷新频率
          <n-select :options="refreshOptions" />
        </div>
      </n-popover>
      <n-input
        :value="searchName"
        placeholder="搜索"
        clearable
        class="mr-[15px] w-[240px] rounded-[30px]"
        @keyup.enter="emit('search')"
        @update:value="emit('update:search-name', $event)"
      >
        <template #suffix></template>
      </n-input>
    </n-space>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  searchName?: string
}>()

void props

const emit = defineEmits<{
  (e: "create"): void
  (e: "search"): void
  (e: "update:search-name", value: string): void
}>()

const refreshOptions = [
  { label: "不刷新", value: 0 },
  { label: "5秒/次", value: 5 },
  { label: "10秒/次", value: 10 },
  { label: "30秒/次", value: 30 },
  { label: "60秒/次", value: 60 },
  { label: "120秒/次", value: 120 },
  { label: "300秒/次", value: 300 }
]
</script>
