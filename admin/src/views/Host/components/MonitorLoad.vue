<template>
  <div class="rounded-[28px] border border-blue-100/80 bg-base-100 p-8 shadow-[0_18px_48px_rgba(15,23,42,0.08)]">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="space-y-2">
        <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Host Monitor</div>
        <div class="text-2xl font-semibold fg-base-100">系统平均负载</div>
        <div class="text-sm leading-7 text-slate-500">
          同时观察 1 / 5 / 15 分钟负载和资源使用率，卡片样式统一为更轻盈的大圆角风格。
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

    <div class="space-y-5">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
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
            <div class="text-sm font-medium text-slate-600">负载趋势</div>
            <div class="mt-1 text-xs text-slate-400">负载曲线与资源占用对照查看系统忙闲状态</div>
          </div>
          <div class="flex items-center gap-3 text-xs text-slate-400">
            <span class="rounded-full bg-blue-50 px-3 py-1 font-medium text-blue-600">1 / 5 / 15 分钟</span>
            <span class="rounded-full bg-slate-100 px-3 py-1 font-medium text-slate-500">资源使用率</span>
          </div>
        </div>
        <SvgTrendChart
          :x-labels="chartLabels"
          :tooltip-labels="chartLabels"
          :series="chartSeries"
          :y-formatter="formatAxisValue"
          empty-text="暂无负载监控数据"
        />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import SvgTrendChart from "@/components/monitor/SvgTrendChart.vue"
import { computed, ref, watch } from "vue"

type LoadPoint = {
	cpuLoad1?: number
	cpuLoad5?: number
	cpuLoad15?: number
	loadUsage?: number
}

type LoadData = {
	date?: string[]
	value?: LoadPoint[]
}

const props = defineProps<{
	data?: LoadData
	rangeDate?: string[]
}>()

const emit = defineEmits(["search"])
const range = ref<[string, string] | null>(
	props.rangeDate && props.rangeDate.length === 2 ? [props.rangeDate[0], props.rangeDate[1]] : null
)

const formatLoad = (value?: number) => Number(value || 0).toFixed(2)
const formatPercent = (value?: number) => `${Number(value || 0).toFixed(2)}%`
const formatAxisValue = (value: number) => (value > 100 ? formatPercent(value) : formatLoad(value))

const summaryCards = computed(() => {
	const values = props.data?.value || []
	const latest = values[values.length - 1] || {}
	const usageValues = values.map(item => Number(item?.loadUsage || 0))
	return [
		{ label: "1 分钟", value: formatLoad(latest.cpuLoad1), desc: "最近 1 分钟平均负载" },
		{ label: "5 分钟", value: formatLoad(latest.cpuLoad5), desc: "最近 5 分钟平均负载" },
		{ label: "15 分钟", value: formatLoad(latest.cpuLoad15), desc: "最近 15 分钟平均负载" },
		{ label: "资源使用率", value: formatPercent(Math.max(0, ...usageValues)), desc: "当前时间段内的最高资源使用率" }
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
		name: "1分钟",
		values: (props.data?.value || []).map(item => Number(item?.cpuLoad1 || 0)),
		color: "#0ea5e9",
		fillFrom: "rgba(14, 165, 233, 0.22)",
		fillTo: "rgba(14, 165, 233, 0.02)",
		valueFormatter: (value: number) => formatLoad(value)
	},
	{
		name: "5分钟",
		values: (props.data?.value || []).map(item => Number(item?.cpuLoad5 || 0)),
		color: "#38bdf8",
		fillFrom: "rgba(56, 189, 248, 0.16)",
		fillTo: "rgba(56, 189, 248, 0.02)",
		valueFormatter: (value: number) => formatLoad(value)
	},
	{
		name: "15分钟",
		values: (props.data?.value || []).map(item => Number(item?.cpuLoad15 || 0)),
		color: "#6366f1",
		fillFrom: "rgba(99, 102, 241, 0.16)",
		fillTo: "rgba(99, 102, 241, 0.02)",
		valueFormatter: (value: number) => formatLoad(value)
	},
	{
		name: "资源使用率",
		values: (props.data?.value || []).map(item => Number(item?.loadUsage || 0)),
		color: "#f59e0b",
		fillFrom: "rgba(245, 158, 11, 0.14)",
		fillTo: "rgba(245, 158, 11, 0.02)",
		valueFormatter: (value: number) => formatPercent(value)
	}
])

function handleRangeChange(value: string[] | null) {
	range.value = value && value.length === 2 ? [value[0], value[1]] : null
	onSearch()
}

const startTime = new Date(range.value[0]).toISOString()
const endTime = new Date(range.value[1]).toISOString()

function onSearch() {
	if (!range.value || range.value.length < 2) {
		return
	}
	const params = {
		startTime,
		endTime,
		param: "load"
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
