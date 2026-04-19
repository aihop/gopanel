<template>
  <div class="rounded-[28px] border border-blue-100/80 bg-base-100 shadow-[0_18px_48px_rgba(15,23,42,0.08)]">
    <div class="p-8">
      <div>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="space-y-2">
            <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Host Monitor</div>
            <div class="text-2xl font-semibold fg-base-100">CPU 使用率监控</div>
            <div class="text-sm leading-7 text-slate-500">
              聚焦 CPU 实时占用、均值和峰值变化，统一为更简洁的大圆角卡片风格。
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
      <div>
        <div class="hidden items-center gap-2 lg:flex">
          <span class="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-600">CPU</span>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-500">实时趋势</span>
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
            <div class="text-sm font-medium text-slate-600">CPU 曲线</div>
            <div class="mt-1 text-xs text-slate-400">按时间分段查看当前主机 CPU 占用率变化</div>
          </div>
          <div class="flex items-center gap-2 text-xs text-slate-400">
            <span class="h-2.5 w-2.5 rounded-full bg-sky-500"></span>
            CPU
          </div>
        </div>
        <SvgTrendChart
          :x-labels="chartLabels"
          :tooltip-labels="chartLabels"
          :series="chartSeries"
          :y-formatter="formatPercent"
          empty-text="暂无 CPU 监控数据"
        />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import SvgTrendChart from "@/components/monitor/SvgTrendChart.vue"
import { computed, ref, watch } from "vue"

type CpuData = {
	date?: string[]
	value?: number[]
}

const props = defineProps<{
	data?: CpuData
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
	const cpuValues = values.map(item => Number(item || 0))
	const avg = cpuValues.length ? cpuValues.reduce((sum, cur) => sum + cur, 0) / cpuValues.length : 0

	return [
		{ label: "当前占用", value: formatPercent(latest), desc: "最近一个采样点的 CPU 占用率" },
		{ label: "平均占用", value: formatPercent(avg), desc: "当前时间段内的平均 CPU 占用率" },
		{ label: "峰值占用", value: formatPercent(cpuValues.length ? Math.max(...cpuValues) : 0), desc: "当前时间段内的最高 CPU 占用率" }
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
		name: "CPU",
		values: (props.data?.value || []).map(item => Number(item || 0)),
		color: "#0ea5e9",
		fillFrom: "rgba(14, 165, 233, 0.28)",
		fillTo: "rgba(14, 165, 233, 0.02)",
		valueFormatter: formatPercent
	}
])

function handleRangeChange(value: string[] | null) {
	range.value = value && value.length === 2 ? [value[0], value[1]] : null
	onSearch()
}
const startTime = new Date(range.value[0]).toISOString()
const endTime = new Date(range.value[1]).toISOString()

const onSearch = () => {
	if (!range.value || range.value.length < 2) {
		return
	}
	const params = {
		startTime,
		endTime,
		param: "cpu"
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
