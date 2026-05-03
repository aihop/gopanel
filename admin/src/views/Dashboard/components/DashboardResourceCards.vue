<template>
  <div class="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
    <n-popover
      placement="bottom"
      :width="cpuPopoverWidth"
      trigger="hover"
    >
      <template #trigger>
        <div class="flex min-h-[224px] flex-col justify-between gap-[18px] rounded-[20px] px-[22px] py-[22px] shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition hover:border-blue-300 hover:bg-white">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-[13px] font-semibold text-slate-500">CPU</div>
              <div class="mt-1.5 text-[30px] font-bold leading-[1.1] text-slate-900">{{ formatNumber(currentInfo.cpuUsedPercent) }}%</div>
            </div>
            <ProgressRing
              title=""
              :value="formatNumber(currentInfo.cpuUsedPercent)"
              :size="100"
              :stroke-width="7"
            />
          </div>
          <div class="flex flex-wrap gap-2.5 text-[13px] text-slate-500">
            <span>{{ t("home.core") }} {{ baseInfo.cpuCores }}</span>
            <span>{{ t("home.logicCore") }} {{ baseInfo.cpuLogicalCores }}</span>
          </div>
          <div class="text-[13px] leading-[1.6] text-slate-400">{{ shortText(baseInfo.cpuModelName, 46) }}</div>
        </div>
      </template>
      <div class="space-y-3">
        <n-tag v-if="baseInfo.cpuModelName">{{ baseInfo.cpuModelName }}</n-tag>
        <div class="grid grid-cols-2 gap-1">
          <div
            v-for="(item, index) of currentInfo.cpuPercent"
            :key="index"
          >
            <n-tag
              v-if="cpuShowAll || index < 24"
              class="!w-[140px] !justify-start !text-left"
            >
              CPU-{{ index }}: {{ formatNumber(item) }}%
            </n-tag>
          </div>
        </div>
        <div v-if="currentInfo.cpuPercent.length > 24">
          <n-button
            v-if="!cpuShowAll"
            text
            type="primary"
            size="small"
            @click="emit('toggle-cpu-show-all', true)"
          >
            {{ t("commons.button.showAll") }}
          </n-button>
          <n-button
            v-else
            text
            type="primary"
            size="small"
            @click="emit('toggle-cpu-show-all', false)"
          >
            {{ t("commons.button.hideSome") }}
          </n-button>
        </div>
      </div>
    </n-popover>

    <n-popover
      placement="bottom"
      :width="360"
      trigger="hover"
    >
      <template #trigger>
        <div class="flex min-h-[224px] flex-col justify-between gap-[18px] rounded-[20px] px-[22px] py-[22px] shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition hover:border-blue-300 hover:bg-white">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-[13px] font-semibold text-slate-500">{{ t("monitor.memory") }}</div>
              <div class="mt-1.5 text-[30px] font-bold leading-[1.1] text-slate-900">{{ computeSize(currentInfo.memoryUsed) }}</div>
            </div>
            <ProgressRing
              title=""
              :value="formatNumber(currentInfo.memoryUsedPercent)"
              :size="100"
              :stroke-width="7"
            />
          </div>
          <div class="flex flex-wrap gap-2.5 text-[13px] text-slate-500">
            <span>{{ t("home.total") }} {{ computeSize(currentInfo.memoryTotal) }}</span>
            <span>{{ t("home.free") }} {{ computeSize(currentInfo.memoryAvailable) }}</span>
          </div>
          <div class="text-[13px] leading-[1.6] text-slate-400">{{ t("home.percent") }} {{ formatNumber(currentInfo.memoryUsedPercent) }}%</div>
        </div>
      </template>
      <div class="grid grid-cols-1 gap-2">
        <n-tag>{{ t("home.total") }}: {{ computeSize(currentInfo.memoryTotal) }}</n-tag>
        <n-tag>{{ t("home.used") }}: {{ computeSize(currentInfo.memoryUsed) }}</n-tag>
        <n-tag>{{ t("home.free") }}: {{ computeSize(currentInfo.memoryAvailable) }}</n-tag>
        <n-tag v-if="currentInfo.swapMemoryTotal">
          Swap: {{ computeSize(currentInfo.swapMemoryUsed) }} / {{ computeSize(currentInfo.swapMemoryTotal) }}
        </n-tag>
        <n-popconfirm
          :show-icon="false"
          @positive-click="emit('memory-clean')"
        >
          <template #trigger>
            <n-button
              size="small"
              type="warning"
              ghost
              :loading="memoryCleaning"
              :disabled="memoryCleaning"
            >
              清理缓存
            </n-button>
          </template>
          该操作会尝试清理 Linux 内核缓存以回收内存，不会中断 HTTP 服务，但可能短暂增加磁盘 IO。继续？
        </n-popconfirm>
      </div>
    </n-popover>

    <n-popover
      placement="bottom"
      :width="320"
      trigger="hover"
    >
      <template #trigger>
        <div class="flex min-h-[224px] flex-col justify-between gap-[18px] rounded-[20px] px-[22px] py-[22px] shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition hover:border-blue-300 hover:bg-white">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-[13px] font-semibold text-slate-500">{{ t("home.load") }}</div>
              <div class="mt-1.5 text-[30px] font-bold leading-[1.1] text-slate-900">{{ formatNumber(currentInfo.loadUsagePercent) }}%</div>
            </div>
            <ProgressRing
              title=""
              :value="formatNumber(currentInfo.loadUsagePercent)"
              :size="100"
              :stroke-width="7"
            />
          </div>
          <div class="flex flex-wrap gap-2.5 text-[13px] text-slate-500">
            <span>1m {{ formatNumber(currentInfo.load1) }}</span>
            <span>5m {{ formatNumber(currentInfo.load5) }}</span>
            <span>15m {{ formatNumber(currentInfo.load15) }}</span>
          </div>
          <div class="text-[13px] leading-[1.6] text-slate-400">{{ loadStatus(currentInfo.loadUsagePercent, t) }}</div>
        </div>
      </template>
      <div class="grid grid-cols-1 gap-2">
        <n-tag>{{ t("home.loadAverage", 1) }}: {{ formatNumber(currentInfo.load1) }}</n-tag>
        <n-tag>{{ t("home.loadAverage", 5) }}: {{ formatNumber(currentInfo.load5) }}</n-tag>
        <n-tag>{{ t("home.loadAverage", 15) }}: {{ formatNumber(currentInfo.load15) }}</n-tag>
      </div>
    </n-popover>
  </div>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { computeSize } from "@/utils/util"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import ProgressRing from "./ProgressRing.vue"
import { formatNumber, loadStatus, loadWidth, shortText } from "./dashboardStatusHelpers"

const props = defineProps<{
  baseInfo: Dashboard.BaseInfo
  currentInfo: Dashboard.CurrentInfo
  cpuShowAll: boolean
  memoryCleaning: boolean
}>()

const emit = defineEmits<{
  (e: "toggle-cpu-show-all", value: boolean): void
  (e: "memory-clean"): void
}>()

const { t } = useI18n()

const cpuPopoverWidth = computed(() => loadWidth(props.cpuShowAll, props.currentInfo.cpuPercent || []))
</script>
