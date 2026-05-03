<template>
  <n-modal
    :show="show"
    preset="card"
    :style="{ width: 'min(720px, calc(100vw - 32px))' }"
    title="最近一天 IP 统计"
    size="small"
    @update:show="emit('update:show', $event)"
  >
    <div
      v-if="loading"
      class="py-10 text-center text-sm text-slate-500"
    >
      正在统计最近一天的访问 IP...
    </div>
    <template v-else-if="stats">
      <div class="space-y-4">
        <div class="grid gap-3 md:grid-cols-3">
          <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
            <div class="text-xs text-slate-400">统计日期</div>
            <div class="mt-2 text-lg font-semibold text-slate-800">{{ stats.date }}</div>
          </div>
          <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
            <div class="text-xs text-slate-400">独立 IP</div>
            <div class="mt-2 text-lg font-semibold text-slate-800">{{ stats.uniqueIpCount }}</div>
          </div>
          <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
            <div class="text-xs text-slate-400">请求数</div>
            <div class="mt-2 text-lg font-semibold text-slate-800">{{ stats.requestCount }}</div>
          </div>
        </div>

        <div class="rounded-xl border border-slate-200 bg-white px-4 py-4">
          <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
            Top IP
          </div>
          <div
            v-if="stats.topIps.length"
            class="space-y-2"
          >
            <div
              v-for="item in stats.topIps"
              :key="item.ip"
              class="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-3 text-sm text-slate-700"
            >
              <div class="font-medium tabular-nums">{{ item.ip }}</div>
              <div class="text-slate-500">{{ item.count }} 次</div>
            </div>
          </div>
          <div
            v-else
            class="text-sm text-slate-500"
          >
            最近一天暂无访问记录
          </div>
        </div>

        <div class="break-all text-xs text-slate-400">
          {{ stats.path }}
        </div>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import { NModal } from "naive-ui"

defineProps<{
  show: boolean
  loading: boolean
  stats: Website.WebSiteTodayIPStats | null
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()
</script>
