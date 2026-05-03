<template>
  <n-modal
    :show="show"
    preset="card"
    :style="{ width: 'min(820px, calc(100vw - 32px))' }"
    :title="title"
    size="small"
    @update:show="emit('update:show', $event)"
  >
    <template v-if="entry">
      <div class="space-y-4">
        <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4">
          <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
            关键信息
          </div>
          <div class="grid gap-3 text-sm text-slate-600 md:grid-cols-2">
            <div class="rounded-lg bg-white px-3 py-3">
              <div class="text-xs text-slate-400">来源 IP</div>
              <div class="mt-1 font-medium">{{ entry.ip }}</div>
            </div>
            <div class="rounded-lg bg-white px-3 py-3">
              <div class="text-xs text-slate-400">状态码</div>
              <div class="mt-1 font-medium">{{ entry.statusText }}</div>
            </div>
            <div class="rounded-lg bg-white px-3 py-3">
              <div class="text-xs text-slate-400">请求方法</div>
              <div class="mt-1 font-medium">{{ entry.method }}</div>
            </div>
            <div class="rounded-lg bg-white px-3 py-3">
              <div class="text-xs text-slate-400">请求时间</div>
              <div class="mt-1 font-medium">{{ entry.timeText }}</div>
            </div>
            <div class="rounded-lg bg-white px-3 py-3">
              <div class="text-xs text-slate-400">耗时</div>
              <div class="mt-1 font-medium">{{ entry.durationText }}</div>
            </div>
            <div
              v-if="entry.sizeText"
              class="rounded-lg bg-white px-3 py-3"
            >
              <div class="text-xs text-slate-400">响应大小</div>
              <div class="mt-1 font-medium">{{ entry.sizeText }}</div>
            </div>
            <div class="rounded-lg bg-white px-3 py-3 md:col-span-2">
              <div class="text-xs text-slate-400">访问页面</div>
              <div class="mt-1 break-all font-medium">{{ entry.path }}</div>
            </div>
          </div>
        </div>

        <div class="rounded-xl border border-slate-200 bg-white px-4 py-4">
          <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-500">
            补充信息
          </div>
          <div class="grid gap-3 text-sm text-slate-600 md:grid-cols-2">
            <div
              v-if="entry.host"
              class="rounded-lg bg-slate-50 px-3 py-3"
            >
              <div class="text-xs text-slate-400">域名</div>
              <div class="mt-1 font-medium">{{ entry.host }}</div>
            </div>
            <div class="rounded-lg bg-slate-50 px-3 py-3">
              <div class="text-xs text-slate-400">解析状态</div>
              <div class="mt-1 font-medium">{{ entry.parsed ? "已结构化" : "原始日志行" }}</div>
            </div>
            <div
              v-if="entry.referer"
              class="rounded-lg bg-slate-50 px-3 py-3 md:col-span-2"
            >
              <div class="text-xs text-slate-400">Referer</div>
              <div class="mt-1 break-all font-medium">{{ entry.referer }}</div>
            </div>
            <div
              v-if="entry.userAgentFull"
              class="rounded-lg bg-slate-50 px-3 py-3 md:col-span-2"
            >
              <div class="text-xs text-slate-400">User-Agent</div>
              <div class="mt-1 break-all font-medium">{{ entry.userAgentFull }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-4 rounded-xl border border-slate-200 bg-slate-900 px-4 py-4">
        <div class="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-slate-400">
          格式化数据
        </div>
        <div class="break-all whitespace-pre-wrap font-mono text-[11px] leading-5 text-slate-200">
          {{ entry.formattedRaw }}
        </div>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { NModal } from "naive-ui"
import type { ParsedLogEntry } from "./websiteLogHelpers"

defineProps<{
  show: boolean
  title: string
  entry: ParsedLogEntry | null
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()
</script>
