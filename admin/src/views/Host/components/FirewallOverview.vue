<script setup lang="ts">
defineProps<{
  summaryCards: Array<{ label: string; value: string; desc: string }>
  base: Record<string, any>
  isRunning: boolean
}>()

const emit = defineEmits<{
  (e: "operate", operation: string): void
  (e: "refresh"): void
}>()
</script>

<template>
  <div class="bg-white/86 rounded-[28px] border border-blue-100/80 p-8 shadow-[0_28px_80px_rgba(15,23,42,0.08)] backdrop-blur-xl sm:p-10">
    <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
      <div class="max-w-3xl space-y-4">
        <div class="inline-flex rounded-full border border-blue-200 bg-blue-50 px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
          Firewall Center
        </div>
        <div class="text-4xl font-semibold leading-[1.08] fg-base-100 sm:text-5xl">防火墙</div>
        <div class="text-base leading-8 text-slate-500 sm:text-lg">
          开启防火墙后，系统将对所有入站和出站流量进行过滤，防止未授权访问
        </div>
      </div>
      <div class="grid gap-3 sm:grid-cols-4 lg:min-w-[560px]">
        <div
          v-for="item in summaryCards"
          :key="item.label"
          class="rounded-[24px] border border-slate-200 bg-slate-50/80 p-5"
        >
          <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">{{ item.label }}</div>
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
          {{ base.name || "未检测" }} · {{ isRunning ? "已启动" : "未启动" }}
        </n-tag>
        <n-tag
          round
          :bordered="false"
          type="info"
        >版本：{{ base.version || "--" }}</n-tag>
        <n-tag
          round
          :bordered="false"
          type="default"
        >Ping：<span class="text-warning">待支持</span></n-tag>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <n-popconfirm
          v-if="!isRunning"
          @positive-click="emit('operate', 'start')"
        >
          <template #trigger>
            <n-button
              type="primary"
              class="!rounded-[18px] px-5 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
            >
              启动
            </n-button>
          </template>
          将会启动当前系统防火墙，是否继续？
        </n-popconfirm>
        <n-popconfirm
          v-else
          @positive-click="emit('operate', 'stop')"
        >
          <template #trigger>
            <n-button
              ghost
              type="warning"
              class="!rounded-[18px] px-5"
            >关闭</n-button>
          </template>
          系统防火墙关闭后，服务器将失去安全防护，是否继续？
        </n-popconfirm>
        <n-popconfirm @positive-click="emit('operate', 'restart')">
          <template #trigger>
            <n-button
              ghost
              class="!rounded-[18px] px-5"
            >重启</n-button>
          </template>
          将对当前防火墙执行重启操作，是否继续？
        </n-popconfirm>
        <n-button
          ghost
          class="!rounded-[18px] px-5"
          @click="emit('refresh')"
        >刷新</n-button>
      </div>
    </div>
  </div>
</template>
