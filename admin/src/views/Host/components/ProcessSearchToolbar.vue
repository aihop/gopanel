<script setup lang="ts">
defineProps<{
  mode: "process" | "network"
  processSearch: { pid: string; name: string; username: string }
  networkSearch: { processID: string; processName: string; port: string }
}>()

const emit = defineEmits<{
  (e: "search"): void
  (e: "reset"): void
  (e: "update:process-pid", value: string): void
  (e: "update:process-name", value: string): void
  (e: "update:process-username", value: string): void
  (e: "update:network-process-id", value: string): void
  (e: "update:network-process-name", value: string): void
  (e: "update:network-port", value: string): void
}>()
</script>

<template>
  <div class="mb-4 flex items-center gap-4">
    <template v-if="mode === 'process'">
      <n-input
        :value="processSearch.pid"
        class="w-[150px]"
        placeholder="进程ID"
        @update:value="emit('update:process-pid', $event)"
      />
      <n-input
        :value="processSearch.name"
        class="w-[150px]"
        placeholder="名称"
        @update:value="emit('update:process-name', $event)"
      />
      <n-input
        :value="processSearch.username"
        class="w-[150px]"
        placeholder="用户"
        @update:value="emit('update:process-username', $event)"
      />
    </template>
    <template v-else>
      <n-input
        :value="networkSearch.processID"
        class="w-[150px]"
        placeholder="进程ID"
        @update:value="emit('update:network-process-id', $event)"
      />
      <n-input
        :value="networkSearch.processName"
        class="w-[150px]"
        placeholder="进程名称"
        @update:value="emit('update:network-process-name', $event)"
      />
      <n-input
        :value="networkSearch.port"
        class="w-[150px]"
        placeholder="端口"
        @update:value="emit('update:network-port', $event)"
      />
    </template>
    <n-button
      type="primary"
      @click="emit('search')"
    >
      <template #icon>
        <Icon
          name="ion:search-outline"
          :size="18"
        />
      </template>
      搜索
    </n-button>
    <n-button @click="emit('reset')">
      <template #icon>
        <Icon
          name="ion:settings-outline"
          :size="18"
        />
      </template>
      重置
    </n-button>
  </div>
</template>
