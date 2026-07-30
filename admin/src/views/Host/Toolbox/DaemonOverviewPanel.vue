<template>
  <div class="dmn-overview rounded-[28px] border border-blue-100/80 bg-white/86 p-8 shadow-[0_28px_80px_rgba(15,23,42,0.08)] backdrop-blur-xl sm:p-10">
    <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
      <div class="max-w-3xl space-y-4">
        <div class="inline-flex rounded-full border border-blue-200 bg-blue-50 px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
          Daemon Center
        </div>
        <div class="text-4xl font-semibold leading-[1.08] fg-base-100 sm:text-5xl">守护进程</div>
        <div class="text-base leading-8 text-slate-500 sm:text-lg">
          统一管理常驻进程、配置文件与运行状态，支持批量启动、停止、重载与日志查看
        </div>
      </div>
      <div class="grid gap-3 sm:grid-cols-4 lg:min-w-[560px]">
        <div
          v-for="item in summaryCards"
          :key="item.label"
          class="rounded-[24px] border border-slate-200 bg-slate-50/80 p-5"
        >
          <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
            {{ item.label }}
          </div>
          <div class="mt-3 text-xl font-semibold fg-base-100">{{ item.value }}</div>
          <div class="mt-2 text-sm leading-6 text-slate-500">{{ item.desc }}</div>
        </div>
      </div>
    </div>

    <div class="mt-8 flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
      <div class="flex flex-wrap items-center gap-3">
        <n-tag
          round
          :bordered="false"
          :type="isRunning ? 'success' : 'warning'"
          class="!px-4 !py-2"
        >
          服务状态 · {{ isRunning ? "已启动" : "未启动" }}
        </n-tag>
        <n-tag
          round
          :bordered="false"
          type="info"
        >
          当前视图 · {{ activeTab === "list" ? "进程列表" : "配置文件" }}
        </n-tag>
        <n-tag
          v-if="agentStatus.online && agentStatus.version"
          round
          :bordered="false"
          type="default"
        >
          gp-agent · v{{ agentStatus.version }}
        </n-tag>
        <n-tag
          v-if="agentUpdate?.needUpdate"
          round
          :bordered="false"
          type="warning"
        >
          有新版 v{{ agentUpdate.latestVersion }}
        </n-tag>
        <n-button
          v-if="agentUpdate?.needUpdate && agentStatus.online"
          size="small"
          type="warning"
          round
          :loading="updatingAgent"
          @click="emit('update-agent')"
        >
          更新 gp-agent
        </n-button>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <n-button
          type="primary"
          ghost
          class="!rounded-[18px] px-5"
          @click="emit('daemon-start')"
        >
          全部启动
        </n-button>
        <n-button
          type="error"
          ghost
          class="!rounded-[18px] px-5"
          @click="emit('daemon-stop')"
        >
          全部停止
        </n-button>
        <n-button
          ghost
          class="!rounded-[18px] px-5"
          @click="emit('refresh')"
        >刷新</n-button>
        <n-button
          type="primary"
          class="!rounded-[18px] px-5 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
          @click="emit('create')"
        >
          创建守护进程
        </n-button>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
defineProps<{
  summaryCards: Array<{ label: string; value: string | number; desc: string }>
  isRunning: boolean
  activeTab: string
  agentStatus: { online: boolean; error?: string; version?: string }
  agentUpdate?: { needUpdate: boolean; currentVersion?: string; latestVersion?: string }
  updatingAgent?: boolean
}>()

const emit = defineEmits<{
  (e: "daemon-start"): void
  (e: "daemon-stop"): void
  (e: "refresh"): void
  (e: "create"): void
  (e: "update-agent"): void
}>()
</script>

<style>
.theme-dark .dmn-overview .text-slate-500 {
  color: var(--fg-secondary-color) !important;
}
.theme-dark .dmn-overview .text-slate-400 {
  color: var(--fg-secondary-color) !important;
}
.theme-dark .dmn-overview .border-slate-200 {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .dmn-overview .bg-slate-50\/80 {
  background-color: color-mix(in srgb, var(--bg-default-color) 80%, transparent) !important;
}
.theme-dark .dmn-overview .bg-white\/86 {
  background-color: color-mix(in srgb, var(--bg-default-color) 86%, transparent) !important;
}
.theme-dark .dmn-overview .border-blue-100\/80 {
  border-color: color-mix(in srgb, var(--primary-color) 20%, transparent) !important;
}
.theme-dark .dmn-overview .border-blue-200 {
  border-color: color-mix(in srgb, var(--primary-color) 30%, transparent) !important;
}
.theme-dark .dmn-overview .bg-blue-50 {
  background-color: color-mix(in srgb, var(--primary-color) 10%, var(--bg-default-color)) !important;
}
</style>
