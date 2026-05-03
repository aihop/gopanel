<template>
  <n-modal
    :show="show"
    preset="dialog"
    title="容器运行时问题修复"
    positive-text="关闭"
    :show-icon="false"
    @update:show="emit('update:show', $event)"
    @positive-click="emit('update:show', false)"
  >
    <div class="space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="text-sm text-gray-600">
          {{ runtimeDetailText }}
        </div>
        <div class="text-xs text-gray-500">
          Configured: {{ validate?.configuredHost || '-' }} / 模式: {{ validate?.hostPinned ? '固定 Socket' : '自动探测' }}
        </div>
        <n-button
          v-if="canAutoRepair"
          :loading="autoRepairLoading"
          :disabled="!validate?.gpc?.reachable"
          type="primary"
          @click="emit('auto-repair')"
        >
          自动修复
        </n-button>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <n-tag :type="validate?.runtime?.serviceActive ? 'success' : 'warning'">
          Service: {{ validate?.runtime?.serviceActive ? 'active' : 'inactive' }}
        </n-tag>
        <n-tag :type="validate?.runtime?.apiReady ? 'success' : 'warning'">
          API: {{ validate?.runtime?.apiReady ? 'ready' : 'not ready' }}
        </n-tag>
        <n-tag :type="validate?.gpc?.reachable ? 'success' : 'warning'">
          GPC: {{ validate?.gpc?.reachable ? 'OK' : '未连接' }}
        </n-tag>
      </div>

      <div
        v-if="validate?.notes?.length"
        class="rounded-lg bg-orange-50 p-3 text-xs text-orange-700"
      >
        <div
          v-for="(n, i) in validate.notes"
          :key="i"
        >- {{ n }}</div>
      </div>

      <div class="flex flex-wrap gap-2">
        <n-button
          v-if="canAutoRepair"
          :loading="repairSocketLoading"
          :disabled="!validate?.gpc?.reachable || autoRepairLoading"
          type="warning"
          @click="emit('repair-socket')"
        >
          修复 Podman Socket 权限
        </n-button>
        <n-button
          v-if="canAutoRepair"
          :loading="repairLingerLoading"
          :disabled="!validate?.gpc?.reachable || autoRepairLoading"
          @click="emit('repair-linger')"
        >
          启用 Linger（rootless 保活）
        </n-button>
      </div>

      <div
        v-if="validate && canAutoRepair && !validate?.gpc?.reachable"
        class="text-xs text-gray-500"
      >
        GPC 未连接时无法执行一键修复，请先在服务器上启用/启动 gpc helper。
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
defineProps<{
  show: boolean
  validate: any
  runtimeDetailText: string
  canAutoRepair: boolean
  autoRepairLoading: boolean
  repairSocketLoading: boolean
  repairLingerLoading: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "auto-repair"): void
  (e: "repair-socket"): void
  (e: "repair-linger"): void
}>()
</script>
