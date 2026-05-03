<script setup lang="ts">
defineProps<{
  searchValue: string
}>()

const emit = defineEmits<{
  (e: "update:searchValue", value: string): void
  (e: "search"): void
  (e: "pull"): void
  (e: "load"): void
  (e: "build"): void
  (e: "prune"): void
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

const handleSearchUpdate = (value: string) => {
  emit("update:searchValue", value)
}
</script>

<template>
  <n-space class="mb-4">
    <n-button
      type="primary"
      @click="emit('pull')"
    >{{ $t("container.imagePull") }}</n-button>
    <n-button @click="emit('load')">{{ $t("container.imageImport") }}</n-button>
    <n-button @click="emit('build')">{{ $t("container.imageBuild") }}</n-button>
    <n-button
      type="error"
      @click="emit('prune')"
    >{{ $t("container.imageDelete") }}</n-button>
  </n-space>

  <div class="mb-4 flex items-center justify-between">
    <h2 class="text-lg font-semibold">{{ $t("container.image") }}</h2>
    <n-space>
      <n-popover
        trigger="click"
        placement="bottom-start"
        :width="300"
      >
        <template #trigger>
          <div class="rounded-full border border-gray-200 bg-base-100 p-2 px-5 text-sm">列表设置</div>
        </template>
        <div class="flex items-center gap-4 text-nowrap bg-base-100">
          刷新频率
          <n-select :options="refreshOptions" />
        </div>
      </n-popover>
      <n-input
        :value="searchValue"
        placeholder="搜索"
        clearable
        @update:value="handleSearchUpdate"
        @keyup.enter="emit('search')"
      >
        <template #suffix>
          <n-icon name="search" />
        </template>
      </n-input>
    </n-space>
  </div>
</template>
