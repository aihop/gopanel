<template>
  <div class="grid gap-5 rounded-[22px] p-[22px] shadow-[0_1px_2px_rgba(37,99,235,0.05)] lg:grid-cols-[minmax(0,1.3fr)_minmax(0,2fr)]">
    <div class="flex min-w-0 flex-col justify-center gap-2">
      <div class="text-xs font-bold uppercase tracking-[0.08em] text-blue-600">{{ t("home.baseInfo") }}</div>
      <div class="text-2xl font-bold leading-[1.1] text-slate-900">{{ baseInfo.hostname || "--" }}</div>
      <div class="break-words text-[13px] leading-[1.6] text-slate-500">
        <div>{{ [baseInfo.os, baseInfo.platform, baseInfo.platformVersion, baseInfo.kernelArch].filter(Boolean).join(" · ") || "--" }}</div>
        <div class="mt-3 flex items-center gap-3">
          <n-tag
            size="small"
            :bordered="false"
            :type="lowPowerMode ? 'warning' : 'info'"
            round
          >
            {{ lowPowerMode ? "省电模式" : "标准模式" }}
          </n-tag>
          <n-switch
            size="small"
            :value="lowPowerMode"
            @update:value="emit('set-low-power-mode', $event)"
          />
        </div>
      </div>
    </div>

    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <div class="flex min-w-0 flex-col gap-1.5 rounded-2xl border border-[rgba(147,197,253,0.45)] bg-white/90 px-3.5 py-3">
        <span class="text-xs font-semibold text-slate-500">{{ t("home.systemInfo") }}</span>
        <span class="overflow-hidden text-ellipsis whitespace-nowrap text-sm font-semibold text-slate-900">{{ baseInfo.os || "--" }}</span>
      </div>
      <div class="flex min-w-0 flex-col gap-1.5 rounded-2xl border border-[rgba(147,197,253,0.45)] bg-white/90 px-3.5 py-3">
        <span class="text-xs font-semibold text-slate-500">{{ t("home.uptime") }}</span>
        <span class="overflow-hidden text-ellipsis whitespace-nowrap text-sm font-semibold text-slate-900">{{ currentInfo.timeSinceUptime || "--" }}</span>
      </div>
      <div class="flex min-w-0 flex-col gap-1.5 rounded-2xl border border-[rgba(147,197,253,0.45)] bg-white/90 px-3.5 py-3">
        <span class="text-xs font-semibold text-slate-500">{{ t("home.runningTime") }}</span>
        <span class="overflow-hidden text-ellipsis whitespace-nowrap text-sm font-semibold text-slate-900">{{ formatUptime(currentInfo.uptime) }}</span>
      </div>
      <div class="flex min-w-0 flex-col gap-1.5 rounded-2xl border border-[rgba(147,197,253,0.45)] bg-white/90 px-3.5 py-3">
        <span class="text-xs font-semibold text-slate-500">{{ t("menu.process") }}</span>
        <span class="overflow-hidden text-ellipsis whitespace-nowrap text-sm font-semibold text-slate-900">{{ currentInfo.procs }}</span>
      </div>
      <div class="flex min-w-0 flex-col gap-1.5 rounded-2xl border border-[rgba(147,197,253,0.45)] bg-white/90 px-3.5 py-3">
        <span class="text-xs font-semibold text-slate-500">IPv4</span>
        <span class="overflow-hidden text-ellipsis whitespace-nowrap text-sm font-semibold text-slate-900">{{ baseInfo.ipv4Addr || "--" }}</span>
      </div>
      <div class="flex min-w-0 flex-col gap-1.5 rounded-2xl border border-[rgba(147,197,253,0.45)] bg-white/90 px-3.5 py-3">
        <span class="text-xs font-semibold text-slate-500">{{ t("home.kernelVersion") }}</span>
        <div class="flex items-center justify-between gap-2">
          <span class="min-w-0 truncate text-sm font-semibold text-slate-900">{{ shortText(baseInfo.kernelVersion || "--", 20) }}</span>
          <div class="flex items-center gap-2">
            <n-button
              size="tiny"
              quaternary
              type="warning"
              :loading="memoryCleaning"
              :disabled="memoryCleaning"
              @click="emit('memory-clean')"
            >
              清理内存
            </n-button>
            <n-button
              size="tiny"
              quaternary
              type="primary"
              :loading="cpuRelieving"
              :disabled="cpuRelieving"
              @click="emit('cpu-relieve')"
            >
              释放 CPU
            </n-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { useI18n } from "vue-i18n"
import { formatUptime, shortText } from "./dashboardStatusHelpers"

defineProps<{
  baseInfo: Dashboard.BaseInfo
  currentInfo: Dashboard.CurrentInfo
  lowPowerMode: boolean
  memoryCleaning: boolean
  cpuRelieving: boolean
}>()

const emit = defineEmits<{
  (e: "set-low-power-mode", value: boolean): void
  (e: "memory-clean"): void
  (e: "cpu-relieve"): void
}>()

const { t } = useI18n()
</script>
