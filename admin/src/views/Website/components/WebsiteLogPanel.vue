<template>
  <div
    v-if="showStructuredList"
    class="rounded-2xl border border-slate-200 bg-white"
  >
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
      <div class="flex items-center gap-2">
        <div class="text-xs font-medium uppercase tracking-[0.16em] text-slate-500">{{ logPanelTitle }}</div>
      </div>
      <div class="flex items-center gap-2">
        <n-button
          size="small"
          quaternary
          @click="emit('update:show-filters', !showFilters)"
        >
          {{ showFilters ? "收起筛选" : "筛选" }}
        </n-button>
        <n-button
          v-if="canToggleView"
          size="small"
          quaternary
          @click="emit('update:raw-mode', true)"
        >
          原始日志
        </n-button>
        <div class="text-xs text-slate-400">
          {{ loading ? "加载中..." : `第 ${page} / ${Math.max(total, 1)} 页` }}
        </div>
      </div>
    </div>

    <div
      v-if="showFilters"
      class="border-b border-slate-100 bg-slate-50/70 px-4 py-3"
    >
      <div class="flex flex-wrap gap-2">
        <n-button
          v-for="item in statusFilters"
          :key="item.value"
          size="small"
          :type="statusFilter === item.value ? 'primary' : 'default'"
          :ghost="statusFilter !== item.value"
          @click="emit('update:status-filter', item.value)"
        >
          {{ item.label }}
        </n-button>
      </div>
      <div class="mt-3 flex flex-wrap gap-2">
        <n-input
          :value="searchKeyword"
          clearable
          size="small"
          placeholder="搜索 IP、页面、状态码"
          class="w-full max-w-xs"
          @update:value="emit('update:search-keyword', $event)"
        />
        <n-button
          size="small"
          ghost
          @click="emit('copy-path')"
        >
          复制日志路径
        </n-button>
      </div>
      <div class="mt-3 break-all text-xs text-slate-400">
        {{ logPath || "日志路径将在首次读取后显示" }}
      </div>
    </div>

    <div class="max-h-[65vh] overflow-auto">
      <div
        v-if="loading"
        class="px-4 py-6 text-sm text-slate-500"
      >
        正在读取访问日志...
      </div>
      <div
        v-else-if="filteredEntries.length"
        class="divide-y divide-slate-100"
      >
        <div
          v-for="(item, index) in filteredEntries"
          :key="`${index}-${item.raw}`"
          class="cursor-pointer px-4 py-3 transition-colors hover:bg-slate-50"
          :class="selectedLogRaw === item.raw && detailVisible ? 'bg-slate-50' : ''"
          @click="emit('open-detail', item)"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <div class="flex items-start gap-3">
                <div class="shrink-0 rounded-lg bg-slate-100 px-2.5 py-1 text-sm font-semibold tabular-nums text-slate-700">
                  {{ item.ip }}
                </div>
                <div class="min-w-0 flex-1">
                  <div
                    class="truncate text-sm font-semibold text-slate-900"
                    :title="item.path"
                  >
                    {{ item.path }}
                  </div>
                  <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                    <span class="tabular-nums">{{ item.timeText }}</span>
                    <span>{{ item.method }}</span>
                    <span>{{ item.statusText }}</span>
                    <span>{{ item.durationText }}</span>
                    <template v-if="isErrorView">
                      <span v-if="item.host">{{ item.host }}</span>
                      <span v-if="item.userAgent">{{ item.userAgent }}</span>
                    </template>
                  </div>
                </div>
              </div>
            </div>
            <n-tag
              round
              size="small"
              :bordered="false"
              :type="getStatusTagType(item.status)"
            >
              {{ item.statusText }}
            </n-tag>
          </div>
        </div>
      </div>
      <div
        v-else-if="parsedEntries.length"
        class="px-4 py-6 text-sm text-slate-500"
      >
        当前筛选条件下暂无匹配记录
      </div>
      <div
        v-else
        class="px-4 py-6 text-sm text-slate-500"
      >
        {{ emptyText }}
      </div>
    </div>
  </div>

  <div
    v-else
    class="rounded-2xl border border-slate-200 bg-black"
  >
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
      <div class="text-xs font-medium uppercase tracking-[0.16em] text-emerald-400">{{ logPanelTitle }}</div>
      <div class="flex items-center gap-2">
        <n-button
          v-if="canToggleView"
          size="small"
          quaternary
          @click="emit('update:raw-mode', false)"
        >
          简约视图
        </n-button>
        <div class="text-xs text-slate-400">
          {{ loading ? "加载中..." : `第 ${page} / ${Math.max(total, 1)} 页` }}
        </div>
      </div>
    </div>
    <div class="max-h-[65vh] overflow-auto whitespace-pre-wrap px-4 py-4 font-mono text-xs leading-6 text-emerald-300">
      <div v-if="loading">正在读取访问日志...</div>
      <div v-else-if="logContent">{{ logContent }}</div>
      <div
        v-else
        class="text-slate-500"
      >
        {{ emptyText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton, NInput, NTag } from "naive-ui"
import { getStatusTagType, statusFilters, type ParsedLogEntry, type StatusFilter } from "./websiteLogHelpers"

defineProps<{
  showStructuredList: boolean
  showFilters: boolean
  statusFilter: StatusFilter
  searchKeyword: string
  logPath: string
  loading: boolean
  filteredEntries: ParsedLogEntry[]
  parsedEntries: ParsedLogEntry[]
  selectedLogRaw: string
  detailVisible: boolean
  page: number
  total: number
  canToggleView: boolean
  isErrorView: boolean
  emptyText: string
  logPanelTitle: string
  logContent: string
}>()

const emit = defineEmits<{
  (e: "update:show-filters", value: boolean): void
  (e: "update:raw-mode", value: boolean): void
  (e: "update:status-filter", value: StatusFilter): void
  (e: "update:search-keyword", value: string): void
  (e: "copy-path"): void
  (e: "open-detail", item: ParsedLogEntry): void
}>()
</script>
