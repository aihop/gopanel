<template>
  <div class="space-y-8">
    <div class="rounded-2xl border border-slate-200 bg-base-100 p-6 shadow-sm">
      <div class="mb-5 flex items-center justify-between gap-4">
        <div>
          <div class="text-sm font-medium text-slate-500">{{ t("monitor.disk") }}</div>
          <div class="mt-1 text-xl font-semibold fg-base-100">{{ currentInfo.diskData.length }} {{ t("home.blockDevice") }}</div>
        </div>
        <n-button
          v-if="hasMoreDisks"
          text
          type="primary"
          @click="emit('toggle-disk-expanded')"
        >
          {{ diskExpanded ? t("tabs.hide") : t("tabs.more") }}
        </n-button>
      </div>
      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2 2xl:grid-cols-3">
        <n-popover
          v-for="(item, index) in visibleDiskData"
          :key="`disk-${index}`"
          placement="bottom"
          :width="420"
          trigger="hover"
        >
          <template #trigger>
            <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 transition hover:border-blue-300 hover:bg-white">
              <div class="flex min-w-0 items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold fg-base-100">{{ item.path }}</div>
                  <div class="mt-1 flex flex-wrap gap-2 text-xs text-slate-500">
                    <span>{{ item.type }}</span>
                    <span>{{ shortText(item.device, 20) }}</span>
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-base font-semibold fg-base-100">{{ formatNumber(item.usedPercent) }}%</div>
                  <div class="text-xs text-slate-500">{{ computeSize(item.used) }} / {{ computeSize(item.total) }}</div>
                </div>
              </div>
              <n-progress
                class="mt-3"
                type="line"
                :show-indicator="false"
                :height="8"
                :processing="false"
                :color="progressColor(item.usedPercent)"
                :rail-color="'#e2e8f0'"
                :percentage="formatNumber(item.usedPercent)"
              />
            </div>
          </template>
          <div class="space-y-2">
            <n-tag>{{ t("home.mount") }}: {{ item.path }}</n-tag>
            <n-tag>{{ t("commons.table.type") }}: {{ item.type }}</n-tag>
            <n-tag>{{ t("home.fileSystem") }}: {{ item.device }}</n-tag>
            <n-tag>Inode: {{ item.inodesUsed }} / {{ item.inodesTotal }}</n-tag>
            <n-tag>{{ t("home.free") }}: {{ computeSize(item.free) }}</n-tag>
          </div>
        </n-popover>
      </div>
    </div>

    <div
      v-if="acceleratorList.length"
      class="rounded-2xl border border-slate-200 bg-base-100 p-6 shadow-sm"
    >
      <div class="mb-5 flex items-center justify-between gap-4">
        <div>
          <div class="text-sm font-medium text-slate-500">GPU / XPU</div>
          <div class="mt-1 text-xl font-semibold fg-base-100">{{ acceleratorList.length }} {{ t("home.blockDevice") }}</div>
        </div>
        <n-button
          v-if="hasMoreAccelerators"
          text
          type="primary"
          @click="emit('toggle-accelerator-expanded')"
        >
          {{ acceleratorExpanded ? t("tabs.hide") : t("tabs.more") }}
        </n-button>
      </div>
      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <n-popover
          v-for="item in visibleAccelerators"
          :key="item.key"
          placement="bottom"
          :width="320"
          trigger="hover"
        >
          <template #trigger>
            <div
              class="cursor-pointer rounded-2xl border border-slate-200 bg-slate-50 p-4 transition hover:border-blue-300 hover:bg-white"
              @click="emit('go-gpu')"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-sm font-semibold fg-base-100">{{ item.title }}</div>
                  <div class="mt-1 truncate text-xs text-slate-500">{{ item.name }}</div>
                </div>
                <div class="text-right">
                  <div class="text-base font-semibold fg-base-100">{{ item.util }}%</div>
                  <div class="text-xs text-slate-500">{{ item.extra }}</div>
                </div>
              </div>
              <n-progress
                class="mt-3"
                type="line"
                :show-indicator="false"
                :height="8"
                :processing="false"
                :color="progressColor(item.util)"
                :rail-color="'#e2e8f0'"
                :percentage="item.util"
              />
            </div>
          </template>
          <div class="space-y-2">
            <n-tag>{{ item.name }}</n-tag>
            <n-tag>{{ t("monitor.gpuUtil") }}: {{ item.rawUtil }}</n-tag>
            <n-tag>{{ t("monitor.temperature") }}: {{ item.temperature }}</n-tag>
            <n-tag>{{ t("monitor.powerUsage") }}: {{ item.power }}</n-tag>
            <n-tag>{{ t("monitor.memoryUsage") }}: {{ item.memory }}</n-tag>
          </div>
        </n-popover>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { computeSize } from "@/utils/util"
import { useI18n } from "vue-i18n"
import { formatNumber, progressColor, shortText } from "./dashboardStatusHelpers"

interface AcceleratorItem {
  key: string
  title: string
  name: string
  util: number
  rawUtil: string
  temperature: string
  power: string
  memory: string
  extra: string
}

defineProps<{
  currentInfo: Dashboard.CurrentInfo
  visibleDiskData: any[]
  hasMoreDisks: boolean
  diskExpanded: boolean
  acceleratorList: AcceleratorItem[]
  visibleAccelerators: AcceleratorItem[]
  hasMoreAccelerators: boolean
  acceleratorExpanded: boolean
}>()

const emit = defineEmits<{
  (e: "toggle-disk-expanded"): void
  (e: "toggle-accelerator-expanded"): void
  (e: "go-gpu"): void
}>()

const { t } = useI18n()
</script>
