<template>
  <div class="mt-3 rounded-[20px] bg-base-100 p-4 px-6 shadow">
    <div class="flex items-center justify-between">
      <n-space align="center">
        <n-tag type="success" class="uppercase">{{ daemon.containerType }}</n-tag>
        <n-tag
          v-if="daemon.status"
          type="warning"
        >
          {{ dockerStatusText[daemon.status] }}
        </n-tag>
        <span class="text-sm text-gray-500">版本: {{ daemon.version }}</span>
      </n-space>
      <n-space v-if="daemon.status">
        <n-button
          v-if="daemon.status === dockerStatus.Stopped"
          :loading="statusLoading"
          type="primary"
          @click="emit('update-status', 'start')"
        >
          {{ $t("container.start") }}
        </n-button>
        <n-popconfirm
          v-else
          @positive-click="emit('update-status', 'stop')"
        >
          <template #trigger>
            <n-button
              :loading="statusLoading"
              type="warning"
            >停止</n-button>
          </template>
          是否停止？
        </n-popconfirm>
        <n-popconfirm @positive-click="emit('update-status', 'restart')">
          <template #trigger>
            <n-button
              :loading="reloadLoading"
              :disabled="daemon.status === dockerStatus.Stopped"
              type="error"
            >
              重启
            </n-button>
          </template>
          是否重启
        </n-popconfirm>
        <n-button
          :disabled="!validate"
          :type="repairHintType"
          @click="emit('open-repair')"
        >
          问题修复
        </n-button>
      </n-space>
    </div>
  </div>
</template>

<script setup lang="ts">
import { dockerStatus, dockerStatusText } from "../../../enums/dockerStatus.enum"

defineProps<{
  daemon: any
  validate: any
  statusLoading: boolean
  reloadLoading: boolean
  repairHintType: string
}>()

const emit = defineEmits<{
  (e: "update-status", operation: string): void
  (e: "open-repair"): void
}>()
</script>
