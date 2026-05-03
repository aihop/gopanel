<script setup lang="ts">
import { getRuntimeKindLabel, getRuntimeModeLabel, getRunUserLabel } from "@/utils/runtime"
import {
  canCancelInstall,
  disableRebuild,
  disableRestart,
  disableStart,
  disableStop,
  disableUninstall,
  statusLabel,
  statusType
} from "./installedAppHelpers"

defineProps<{
  apps: any[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: "detail", item: any): void
  (e: "cancel", item: any): void
  (e: "log", item: any): void
  (e: "operate", item: any, operation: string): void
  (e: "rebuild", item: any): void
  (e: "delete", item: any): void
}>()
</script>

<template>
  <n-spin :show="loading">
    <div
      v-if="apps.length"
      class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3"
    >
      <div
        v-for="item in apps"
        :key="item.id"
        class="relative overflow-hidden rounded-2xl border border-slate-200/80 bg-gradient-to-b from-white to-slate-50/80 p-5 shadow-sm transition hover:-translate-y-1 hover:border-blue-200/60 hover:shadow-md"
      >
        <div class="pointer-events-none absolute -right-8 -top-10 h-28 w-28 rounded-full bg-blue-500/10 blur-2xl"></div>

        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50/70">
              <img
                v-if="item.app?.icon"
                :src="item.app.icon"
                alt="icon"
                class="h-8 w-8 object-contain"
              />
              <span
                v-else
                class="text-base font-bold text-blue-600"
              >{{ item.name?.slice(0, 1)?.toUpperCase() }}</span>
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <div
                  class="truncate text-base font-semibold text-slate-900 hover:underline"
                  @click="emit('detail', item)"
                >{{ item.name }}</div>
                <n-tag
                  v-if="item.status"
                  :type="statusType(item.status)"
                  size="small"
                  round
                >{{ statusLabel(item.status) }}</n-tag>
                <n-button
                  v-if="canCancelInstall(item)"
                  tertiary
                  type="error"
                  size="tiny"
                  @click="emit('cancel', item)"
                >取消安装</n-button>
              </div>
              <div class="mt-1 truncate text-sm text-slate-500">容器名：{{ item.containerName || "-" }}</div>
              <div class="mt-2 flex flex-wrap items-center gap-2">
                <n-tag
                  size="small"
                  round
                  :bordered="false"
                  type="info"
                >{{ getRuntimeKindLabel(item) }}</n-tag>
                <n-tag
                  size="small"
                  round
                  :bordered="false"
                  :type="item.runtimeMode === 'rootless' ? 'success' : 'default'"
                >{{ getRuntimeModeLabel(item) }}</n-tag>
              </div>
              <div class="mt-1 text-xs text-slate-500">运行用户：{{ getRunUserLabel(item) }}</div>
            </div>
          </div>
        </div>

        <p
          v-if="item.description"
          class="mt-4 line-clamp-3 min-h-[3.5rem] text-sm leading-6 text-slate-600"
        >
          {{ item.description }}
        </p>

        <div class="mt-4 grid grid-cols-2 gap-2">
          <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
            <div class="text-xs text-slate-400">版本</div>
            <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.version || "-" }}</div>
          </div>
          <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
            <div class="text-xs text-slate-400">安装时间</div>
            <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.createdAt || "-" }}</div>
          </div>
          <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
            <div class="text-xs text-slate-400">HTTP 端口</div>
            <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.httpPort || "-" }}</div>
          </div>
          <div class="rounded-xl border border-slate-200/70 bg-white/70 p-3">
            <div class="text-xs text-slate-400">HTTPS 端口</div>
            <div class="mt-1 break-words text-sm font-semibold text-slate-800">{{ item.httpsPort || "-" }}</div>
          </div>
        </div>

        <div class="mt-4 flex flex-wrap items-center justify-end gap-2">
          <n-button
            secondary
            size="small"
            @click="emit('log', item)"
          >日志</n-button>
          <n-button
            secondary
            size="small"
            :disabled="disableStart(item)"
            @click="emit('operate', item, 'start')"
          >启动</n-button>
          <n-button
            secondary
            size="small"
            :disabled="disableStop(item)"
            @click="emit('operate', item, 'stop')"
          >停止</n-button>
          <n-button
            secondary
            size="small"
            :disabled="disableRestart(item)"
            @click="emit('operate', item, 'restart')"
          >重启</n-button>
          <n-button
            secondary
            type="warning"
            size="small"
            :disabled="disableRebuild(item)"
            @click="emit('rebuild', item)"
          >重建</n-button>
          <n-button
            secondary
            type="error"
            size="small"
            :disabled="disableUninstall(item)"
            @click="emit('delete', item)"
          >卸载</n-button>
        </div>
      </div>
    </div>

    <div
      v-else
      class="py-16 text-center text-sm text-slate-400"
    >
      暂无已安装应用
    </div>
  </n-spin>
</template>
