<script setup lang="ts">
defineProps<{
  loading: boolean
  apps: any[]
}>()

const emit = defineEmits<{
  (e: "detail", item: any): void
  (e: "install", item: any): void
}>()
</script>

<template>
  <n-spin :show="loading">
    <div
      v-if="apps.length"
      class="grid gap-[18px] [grid-template-columns:repeat(auto-fill,minmax(280px,1fr))] max-sm:grid-cols-1"
    >
      <div
        v-for="item in apps"
        :key="item.id"
        class="relative overflow-hidden rounded-[24px] border border-[rgba(226,232,240,0.88)] bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.12),transparent_28%),linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_14px_36px_rgba(15,23,42,0.06)] transition-[transform,box-shadow,border-color] duration-[260ms] ease-[ease] hover:-translate-y-1 hover:border-[rgba(59,130,246,0.22)] hover:shadow-[0_22px_44px_rgba(15,23,42,0.1)]"
      >
        <div class="pointer-events-none absolute -right-8 -top-10 h-[120px] w-[120px] rounded-full bg-[rgba(59,130,246,0.14)] blur-[20px]"></div>
        <div class="relative z-[1] flex h-full flex-col gap-[18px] p-[22px]">
          <div class="flex justify-between gap-[14px] max-sm:flex-col">
            <div class="flex min-w-0 flex-1 items-start gap-3">
              <div class="flex h-[54px] w-[54px] flex-shrink-0 items-center justify-center rounded-[18px] border border-[rgba(219,234,254,0.9)] bg-[linear-gradient(135deg,rgba(239,246,255,0.95),rgba(255,255,255,0.75))] shadow-[inset_0_1px_0_rgba(255,255,255,0.65)]">
                <img
                  v-if="item.icon"
                  :src="item.icon"
                  alt="icon"
                  class="h-10 w-10 object-contain"
                />
                <span
                  v-else
                  class="text-base font-bold text-blue-600"
                >
                  {{ item.name?.slice(0, 1)?.toUpperCase() }}
                </span>
              </div>
              <div class="min-w-0 flex-1">
                <div class="truncate text-base font-semibold text-slate-900">{{ item.name }}</div>
                <div class="mt-1 flex items-center justify-between gap-2">
                  <div class="truncate text-sm text-slate-500">{{ item.type }}</div>
                  <n-tag
                    v-if="item.installing"
                    type="warning"
                    size="small"
                    round
                  >安装中</n-tag>
                  <n-tag
                    v-else-if="item.installed"
                    type="success"
                    size="small"
                    round
                  >已安装</n-tag>
                </div>
              </div>
            </div>
            <div class="flex flex-wrap items-center justify-end gap-2 max-sm:justify-start">
              <n-button
                secondary
                size="small"
                @click="emit('detail', item)"
              >{{ $t("app.detail") }}</n-button>
              <n-button
                v-if="item.installed"
                secondary
                type="info"
                size="small"
                disabled
              >{{ $t("app.install") }}</n-button>
              <n-button
                v-else
                secondary
                type="info"
                size="small"
                :disabled="item.installing"
                @click="emit('install', item)"
              >
                {{ item.installing ? $t("commons.status.installing") : $t("commons.operate.install") }}
              </n-button>
            </div>
          </div>

          <p class="m-0 min-h-[66px] text-[0.92rem] leading-[1.7] text-slate-500">{{ item.shortDescZh || item.description || "暂无应用说明，点击详情查看完整信息。" }}</p>

          <div class="grid grid-cols-2 gap-[10px] max-sm:grid-cols-1">
            <div class="rounded-2xl border border-[rgba(226,232,240,0.9)] bg-[rgba(248,250,252,0.95)] px-3 py-2.5">
              <span class="mb-1 block text-[0.72rem] text-slate-400">来源</span>
              <span class="block break-words text-[0.88rem] font-semibold text-slate-800">{{ item.resource || "应用商店" }}</span>
            </div>
            <div
              v-if="item.versions && item.versions.length"
              class="rounded-2xl border border-[rgba(226,232,240,0.9)] bg-[rgba(248,250,252,0.95)] px-3 py-2.5"
            >
              <span class="mb-1 block text-[0.72rem] text-slate-400">版本</span>
              <span class="block break-words text-[0.88rem] font-semibold text-slate-800">{{ item.versions[0] }}</span>
            </div>
          </div>

          <div
            v-if="item.versions && item.versions.length > 1"
            class="flex flex-wrap items-center gap-2 pt-0.5"
          >
            <n-tag
              v-for="version in item.versions.slice(0, 3)"
              :key="version"
              size="small"
              round
              :bordered="false"
            >
              {{ version }}
            </n-tag>
            <span
              v-if="item.versions.length > 3"
              class="text-xs text-slate-400"
            >
              +{{ item.versions.length - 3 }} 个版本
            </span>
          </div>
        </div>
      </div>
    </div>
    <div
      v-else
      class="px-5 py-16 text-center text-[0.95rem] text-slate-400"
    >
      暂无匹配的应用
    </div>
  </n-spin>
</template>
