<template>
  <div class="rounded-[28px] border border-blue-100/80 bg-base-100 shadow-[0_18px_48px_rgba(15,23,42,0.08)]">
    <div class="p-8">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-2">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Host Monitor</div>
          <div class="text-2xl font-semibold fg-base-100">磁盘 I/O 监控</div>
          <div class="text-sm leading-7 text-slate-500">
            把读取、写入、次数和延迟统一收纳到更圆润的卡片中，和 Dashboard 风格保持一致。
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
            <div class="text-sm font-medium text-slate-600">I/O 趋势</div>
            <div class="mt-1 text-xs text-slate-400">结合带宽、次数和延迟查看磁盘活跃程度</div>
          </div>
          <div class="flex items-center gap-3 text-xs text-slate-400">
            <span class="rounded-full bg-blue-50 px-3 py-1 font-medium text-blue-600">读取 / 写入</span>
            <span class="rounded-full bg-slate-100 px-3 py-1 font-medium text-slate-500">次数 / 延迟</span>
          </div>
        </div>
        <SvgTrendChart
          :x-labels="chartLabels"
          :tooltip-labels="chartLabels"
          :series="chartSeries"
          :y-formatter="formatAxisValue"
          empty-text="暂无 I/O 监控数据"
        />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import SvgTrendChart from "@/components/monitor/SvgTrendChart.vue"
import { computed, ref, watch } from "vue"

type IoPoint = {
	read?: number
	write?: number
	count?: number
	time?: number
}

type IoData = {
	date?: string[]
	value?: IoPoint[]
}

const props = defineProps<{
	data?: IoData
	rangeDate?: string[]
}>()

const emit = defineEmits(["search"])
const range = ref<[string, string] | null>(
	props.rangeDate && props.rangeDate.length === 2 ? [props.rangeDate[0], props.rangeDate[1]] : null
)

const formatRate = (value?: number) => {
  const bytes = Number(value || 0)
  if (bytes === 0) return '0.00 KB/s'
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB/s`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB/s`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB/s`
}
const formatCount = (value?: number) => `${Number(value || 0).toFixed(2)} 次/s`
const formatLatency = (value?: number) => `${Number(value || 0).toFixed(2)} ms`
const formatAxisValue = (value: number) => {
	if (value > 5000) return formatRate(value)
	if (value > 100) return formatCount(value)
	return Number(value).toFixed(2)
}

const summaryCards = computed(() => {
	const values = props.data?.value || []
	const latest = values[values.length - 1] || {}
	return [
		{ label: "当前读取", value: formatRate(latest.read), desc: "最近一个采样点的读取吞吐" },
		{ label: "当前写入", value: formatRate(latest.write), desc: "最近一个采样点的写入吞吐" },
		{ label: "读写次数", value: formatCount(latest.count), desc: "最近一个采样点的读写频率" },
		{ label: "读写延迟", value: formatLatency(latest.time), desc: "最近一个采样点的平均延迟" }
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
		name: "读取",
		values: (props.data?.value || []).map(item => Number(item?.read || 0)),
		color: "#0ea5e9",
		fillFrom: "rgba(14, 165, 233, 0.22)",
		fillTo: "rgba(14, 165, 233, 0.02)",
		valueFormatter: formatRate
	},
	{
		name: "写入",
		values: (props.data?.value || []).map(item => Number(item?.write || 0)),
		color: "#6366f1",
		fillFrom: "rgba(99, 102, 241, 0.18)",
		fillTo: "rgba(99, 102, 241, 0.02)",
		valueFormatter: formatRate
	},
	{
		name: "读写次数",
		values: (props.data?.value || []).map(item => Number(item?.count || 0)),
		color: "#10b981",
		fillFrom: "rgba(16, 185, 129, 0.14)",
		fillTo: "rgba(16, 185, 129, 0.02)",
		valueFormatter: formatCount
	},
	{
		name: "读写延迟",
		values: (props.data?.value || []).map(item => Number(item?.time || 0)),
		color: "#f59e0b",
		fillFrom: "rgba(245, 158, 11, 0.14)",
		fillTo: "rgba(245, 158, 11, 0.02)",
		valueFormatter: formatLatency
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
		param: "io"
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
