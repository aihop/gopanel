<script setup lang="ts">
defineEmits<{
  (e: "cloud-apply"): void
  (e: "sync"): void
  (e: "upload"): void
  (e: "refresh"): void
}>()
</script>

<template>
  <div class="px-4">
    <div class="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
      <div class="max-w-3xl space-y-3">
        <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Security</div>
        <div class="text-2xl font-semibold fg-base-100">{{ $t("ssl.sslManagement") }}</div>
        <div class="text-sm leading-7 fg-secondary-color">
          {{ $t("ssl.sslManagementHelper") }}
        </div>
      </div>
      <n-space>
        <n-button type="primary" @click="$emit('cloud-apply')">{{ $t("ssl.cdnApply") }}</n-button>
        <n-button ghost type="primary" @click="$emit('sync')">{{ $t("ssl.sync") }}</n-button>
        <n-button ghost type="primary" @click="$emit('upload')">{{ $t("ssl.upload") }}</n-button>
        <n-button ghost @click="$emit('refresh')">{{ $t("ssl.refresh") }}</n-button>
      </n-space>
    </div>
  </div>
</template>
