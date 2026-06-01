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
        class="app-grid-card"
      >
        <div class="app-grid-card__glow pointer-events-none absolute -right-8 -top-10 h-[120px] w-[120px] rounded-full blur-[20px]"></div>
        <div class="relative z-[1] flex h-full flex-col gap-[18px] p-[22px]">
          <div class="flex justify-between gap-[14px] max-sm:flex-col">
            <div class="flex min-w-0 flex-1 items-start gap-3">
              <div class="app-grid-card__icon flex h-[54px] w-[54px] flex-shrink-0 items-center justify-center rounded-[18px]">
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
            <div class="app-grid-card__info rounded-2xl px-3 py-2.5">
              <span class="mb-1 block text-[0.72rem] text-slate-400">来源</span>
              <span class="block break-words text-[0.88rem] font-semibold text-slate-800">{{ item.resource || "应用商店" }}</span>
            </div>
            <div
              v-if="item.versions && item.versions.length"
              class="app-grid-card__info rounded-2xl px-3 py-2.5"
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

<style scoped>
.app-grid-card {
  position: relative;
  overflow: hidden;
  border-radius: 24px;
  border: 1px solid color-mix(in srgb, var(--border-color) 88%, transparent);
  background:
    radial-gradient(circle at top right, color-mix(in srgb, var(--primary-color) 12%, transparent), transparent 28%),
    linear-gradient(180deg, color-mix(in srgb, var(--bg-default-color) 98%, white), color-mix(in srgb, var(--bg-secondary-color) 92%, transparent));
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.06);
  transition: transform 260ms ease, box-shadow 260ms ease, border-color 260ms ease;
}
.app-grid-card:hover {
  transform: translateY(-4px);
  border-color: color-mix(in srgb, var(--primary-color) 22%, transparent);
  box-shadow: 0 22px 44px rgba(15, 23, 42, 0.1);
}
.app-grid-card__glow {
  background: color-mix(in srgb, var(--primary-color) 14%, transparent);
}
.app-grid-card__icon {
  border: 1px solid color-mix(in srgb, var(--primary-color) 22%, var(--bg-secondary-color));
  background: linear-gradient(135deg, color-mix(in srgb, var(--primary-color) 18%, var(--bg-default-color)), color-mix(in srgb, var(--bg-default-color) 75%, transparent));
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--bg-default-color) 65%, transparent);
}
.app-grid-card__info {
  border: 1px solid color-mix(in srgb, var(--border-color) 90%, transparent);
  background: color-mix(in srgb, var(--bg-secondary-color) 95%, transparent);
}
</style>
