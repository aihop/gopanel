<template>
  <div class="rounded-[28px] border border-blue-100/80 bg-base-100 shadow-[0_18px_48px_rgba(15,23,42,0.08)]">
    <div class="p-8">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-2">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Host Monitor</div>
          <div class="text-2xl font-semibold fg-base-100">网络吞吐监控</div>
          <div class="text-sm leading-7 text-slate-500">
            聚合展示主机上行、下行与峰值变化，视觉风格与当前 Dashboard 保持一致。
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
    <div class="p-8">
      <div class="hidden items-center gap-2 lg:flex">
        <span class="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-600">上行 / 下行</span>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-500">实时趋势</span>
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
            <div class="text-sm font-medium text-slate-600">流量趋势</div>
            <div class="mt-1 text-xs text-slate-400">单位自动换算，支持拖拽缩放查看波动区间</div>
          </div>
          <div class="flex items-center gap-3 text-xs text-slate-400">
            <span class="flex items-center gap-2">
              <span class="h-2.5 w-2.5 rounded-full bg-sky-500"></span>
              上行
            </span>
            <span class="flex items-center gap-2">
              <span class="h-2.5 w-2.5 rounded-full bg-indigo-500"></span>
              下行
            </span>
          </div>
        </div>
        <SvgTrendChart
          :x-labels="chartLabels"
          :tooltip-labels="chartLabels"
          :series="chartSeries"
          :y-formatter="formatRate"
          empty-text="暂无网络监控数据"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import SvgTrendChart from "@/components/monitor/SvgTrendChart.vue"
import { computed, ref, watch } from "vue"

type MonitorPoint = {
	up?: number
	down?: number
}

type MonitorNetworkData = {
	date?: string[]
	value?: MonitorPoint[]
}

const props = defineProps<{
	data?: MonitorNetworkData
	rangeDate?: string[]
}>()

const emit = defineEmits(["search"])
const range = ref<[string, string] | null>(
	props.rangeDate && props.rangeDate.length === 2 ? [props.rangeDate[0], props.rangeDate[1]] : null
)

const formatRate = (value?: number) => {
	const bytes = Number(value || 0)
	if (bytes === 0) return "0.00 KB/s"
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB/s`
	if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB/s`
	return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB/s`
}

const summaryCards = computed(() => {
	const values = props.data?.value || []
	const latest = values[values.length - 1] || {}
	const upValues = values.map((item: any) => Number(item?.up || 0))
	const downValues = values.map((item: any) => Number(item?.down || 0))
	const avg = (items: number[]) => (items.length ? items.reduce((sum, cur) => sum + cur, 0) / items.length : 0)

	return [
		{
			label: "当前上行",
			value: formatRate(Number(latest.up || 0)),
			desc: "最近一个采样点的上传速率"
		},
		{
			label: "当前下行",
			value: formatRate(Number(latest.down || 0)),
			desc: "最近一个采样点的下载速率"
		},
		{
			label: "上行峰值",
			value: formatRate(upValues.length ? Math.max(...upValues) : 0),
			desc: "当前时间段内的最高上传速率"
		},
		{
			label: "下行均值",
			value: formatRate(avg(downValues)),
			desc: "当前时间段内的平均下载速率"
		}
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
		name: "上行",
		values: (props.data?.value || []).map(item => Number(item?.up || 0)),
		color: "#0ea5e9",
		fillFrom: "rgba(14, 165, 233, 0.22)",
		fillTo: "rgba(14, 165, 233, 0.02)",
		valueFormatter: formatRate
	},
	{
		name: "下行",
		values: (props.data?.value || []).map(item => Number(item?.down || 0)),
		color: "#6366f1",
		fillFrom: "rgba(99, 102, 241, 0.18)",
		fillTo: "rgba(99, 102, 241, 0.02)",
		valueFormatter: formatRate
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
		param: "network"
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
