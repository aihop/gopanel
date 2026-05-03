<template>
  <div
    v-if="validate"
    class="mt-3 rounded-[20px] bg-base-100 p-4 px-6 shadow"
  >
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <n-tag
          class="uppercase"
          :type="validate.runtimeKind === 'docker' ? 'success' : validate.runtimeKind === 'podman' ? 'warning' : 'default'"
        >
          {{ runtimeBadgeText }}
        </n-tag>
        <n-tag
          size="small"
          :type="validate.hostPinned ? 'success' : 'default'"
        >
          {{ validate.hostPinned ? '固定 Socket' : '自动探测' }}
        </n-tag>
        <span class="text-sm text-gray-500">Current: {{ currentRuntimeHost }}</span>
        <span class="text-sm text-gray-500">Configured: {{ validate.configuredHost || '-' }}</span>
        <span class="text-sm text-gray-500">OS: {{ validate.os }}/{{ validate.arch }}</span>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <n-tag
          size="small"
          :type="validate.compose?.ok ? 'success' : 'error'"
        >
          Compose: {{ validate.compose?.ok ? `${validate.compose.bin} ${validate.compose.prefix}` : '不可用' }}
        </n-tag>
        <n-tag
          size="small"
          :type="validate.gpc?.reachable ? 'success' : 'warning'"
        >
          GPC: {{ validate.gpc?.reachable ? 'OK' : '未连接' }}
        </n-tag>
      </div>
    </div>
    <div
      v-if="validate.notes?.length"
      class="mt-3 space-y-1 text-xs text-orange-600"
    >
      <div
        v-for="(n, i) in validate.notes"
        :key="i"
      >- {{ n }}</div>
    </div>
    <div
      v-if="dockerOnly"
      class="mt-3 rounded-lg bg-orange-50 p-3 text-xs text-orange-700"
    >
      当前运行时为 {{ validate.runtimeKind }}，此页面的 daemon.json/iptables 等配置主要针对 Docker。Podman 模式下仅支持镜像加速（Linux 需连接 GPC；macOS 需 podman machine 可用）。
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  validate: any
  runtimeBadgeText: string
  currentRuntimeHost: string
  dockerOnly: boolean
}>()
</script>
