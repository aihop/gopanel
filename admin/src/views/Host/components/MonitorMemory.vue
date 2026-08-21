<template>
  <div class="rounded-[28px] border border-blue-100/80 bg-base-100 shadow-[0_18px_48px_rgba(15,23,42,0.08)]">
    <div class="p-8">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-2">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Host Monitor</div>
          <div class="text-2xl font-semibold fg-base-100">{{ t('hostMonitor.memory.title') }}</div>
          <div class="text-sm leading-7 text-slate-500">
            {{ t('hostMonitor.memory.intro') }}
          </div>
        </div>
        <div class="rounded-2xl border border-blue-100 bg-blue-50/70 p-2">
          <n-date-picker
            :formatted-value="range"
            value-format="yyyy-MM-dd HH:mm:ss"
            type="datetimerange"
            clearable
            @update:formatted-value="handleRangeChange"
          />
        </div>
      </div>
    </div>
    <div class="space-y-5 p-8">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div
          v-for="item in summaryCards"
          :key="item.label"
          class="rounded-2xl border border-slate-200 bg-slate-50/80 p-4"
        >
          <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">{{ item.label }}</div>
          <div class="mt-3 text-2xl font-semibold fg-base-100">{{ item.value }}</div>
          <div class="mt-2 text-sm text-slate-500">{{ item.desc }}</div>
        </div>
      </div>
      <div class="rounded-[24px] border border-slate-100 bg-slate-50/80 p-4">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <div class="text-sm font-medium text-slate-600">{{ t('hostMonitor.memory.curveTitle') }}</div>
            <div class="mt-1 text-xs text-slate-400">{{ t('hostMonitor.memory.curveHelper') }}</div>
          </div>
          <div class="flex items-center gap-2 text-xs text-slate-400">
            <span class="h-2.5 w-2.5 rounded-full bg-emerald-500"></span>
            {{ t('monitor.memory') }}
          </div>
        </div>
        <SvgTrendChart
          :x-labels="chartLabels"
          :tooltip-labels="chartLabels"
          :series="chartSeries"
          :y-formatter="formatPercent"
          :empty-text="t('hostMonitor.memory.empty')"
        />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import SvgTrendChart from "@/components/monitor/SvgTrendChart.vue"
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

type MemoryData = {
	date?: string[]
	value?: number[]
}

const props = defineProps<{
	data?: MemoryData
	rangeDate?: string[]
}>()

const emit = defineEmits(["search"])
const range = ref<[string, string] | null>(
	props.rangeDate && props.rangeDate.length === 2 ? [props.rangeDate[0], props.rangeDate[1]] : null
)

const formatPercent = (value?: number) => `${Number(value || 0).toFixed(2)}%`

const summaryCards = computed(() => {
	const values = props.data?.value || []
	const latest = values.length > 0 ? values[values.length - 1] : 0
	const memoryValues = values.map(item => Number(item || 0))
	const avg = memoryValues.length ? memoryValues.reduce((sum, cur) => sum + cur, 0) / memoryValues.length : 0
	return [
		{ label: t('hostMonitor.memory.currentUsage'), value: formatPercent(latest), desc: t('hostMonitor.memory.currentUsageDesc') },
		{ label: t('hostMonitor.memory.avgUsage'), value: formatPercent(avg), desc: t('hostMonitor.memory.avgUsageDesc') },
		{ label: t('hostMonitor.memory.peakUsage'), value: formatPercent(memoryValues.length ? Math.max(...memoryValues) : 0), desc: t('hostMonitor.memory.peakUsageDesc') }
	]
})

const chartLabels = computed(() => {
	const dates = props.data?.date || []
	return dates.map(item => {
		const now = new Date(item)
		return `${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")} ${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`
	})
})

const chartSeries = computed(() => [
	{
		name: t('monitor.memory'),
		values: (props.data?.value || []).map(item => Number(item || 0)),
		color: "#10b981",
		fillFrom: "rgba(16, 185, 129, 0.24)",
		fillTo: "rgba(16, 185, 129, 0.02)",
		valueFormatter: formatPercent
	}
])

function handleRangeChange(value: string[] | null) {
	range.value = value && value.length === 2 ? [value[0], value[1]] : null
	onSearch()
}

function onSearch() {
	if (!range.value || range.value.length < 2) {
		return
	}
	const params = {
		startTime: new Date(range.value[0]).toISOString(),
		endTime: new Date(range.value[1]).toISOString(),
		param: "memory"
	}
	emit("search", params)
}

watch(
	() => props.rangeDate,
	val => {
		range.value = val && val.length === 2 ? [val[0], val[1]] : null
		onSearch()
	}
)
</script>