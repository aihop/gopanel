<template>
  <div
    class="relative"
    ref="containerRef"
  >
    <!-- Y 轴固定悬浮层 -->
    <div class="pointer-events-none absolute bottom-0 left-0 top-0 z-10 w-[54px] border-r border-slate-100 bg-white/80 backdrop-blur-sm dark:border-slate-800 dark:bg-slate-900/80">
      <div
        v-for="tick in yTicks"
        :key="`tick-html-${tick}`"
        class="absolute right-[10px] text-[11px] text-slate-500"
        :style="{ top: `${getY(tick) - 6}px` }"
      >
        {{ formatY(tick) }}
      </div>
    </div>

    <!-- 图表滚动区 -->
    <div
      ref="scrollContainerRef"
      class="relative overflow-x-auto overflow-y-hidden custom-scrollbar"
      :style="{ height: `${height}px` }"
      @mouseleave="activeIndex = null"
      @mousemove="handleMove"
      @scroll="handleScroll"
    >
      <svg
        class="h-full"
        :style="{ width: `${viewWidth}px`, minWidth: '100%' }"
        :viewBox="`0 0 ${viewWidth} ${viewHeight}`"
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient
            v-for="series in normalizedSeries"
            :id="series.gradientId"
            :key="series.gradientId"
            x1="0"
            x2="0"
            y1="0"
            y2="1"
          >
            <stop
              offset="0%"
              :stop-color="series.fillFrom"
            />
            <stop
              offset="100%"
              :stop-color="series.fillTo"
            />
          </linearGradient>
        </defs>

        <g>
          <line
            v-for="tick in yTicks"
            :key="`grid-${tick}`"
            :x1="padding.left"
            :x2="viewWidth - padding.right"
            :y1="getY(tick)"
            :y2="getY(tick)"
            stroke="rgba(148, 163, 184, 0.16)"
            stroke-width="1"
          />
        </g>

        <g v-if="labelIndices.length > 0">
          <text
            v-for="index in labelIndices"
            :key="`label-${index}`"
            :x="getX(index)"
            :y="viewHeight - 12"
            text-anchor="middle"
            fill="#64748b"
            font-size="11"
          >
            {{ xLabels[index] }}
          </text>
        </g>

        <g
          v-for="series in normalizedSeries"
          :key="series.name"
        >
          <path
            v-if="series.areaPath"
            :d="series.areaPath"
            :fill="`url(#${series.gradientId})`"
            opacity="1"
          />
          <path
            :d="series.linePath"
            fill="none"
            :stroke="series.color"
            stroke-width="3"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </g>

        <g v-if="activeIndex !== null && activeIndex >= 0 && activeIndex < pointCount">
          <line
            :x1="getX(activeIndex)"
            :x2="getX(activeIndex)"
            :y1="padding.top"
            :y2="viewHeight - padding.bottom"
            stroke="rgba(59, 130, 246, 0.28)"
            stroke-dasharray="4 4"
          />
          <g
            v-for="series in normalizedSeries"
            :key="`point-${series.name}`"
          >
            <circle
              :cx="getX(activeIndex)"
              :cy="getY(series.values[activeIndex] || 0)"
              r="5"
              fill="white"
              :stroke="series.color"
              stroke-width="3"
            />
          </g>
        </g>
      </svg>

      <div
        v-if="activeIndex !== null && tooltipItems.length > 0"
        class="pointer-events-none absolute z-20 min-w-[180px] rounded-2xl bg-slate-900/95 px-4 py-3 text-xs text-slate-100 shadow-[0_16px_36px_rgba(15,23,42,0.3)]"
        :style="tooltipStyle"
      >
        <div class="mb-2 font-medium text-slate-200">{{ tooltipTitle }}</div>
        <div
          v-for="item in tooltipItems"
          :key="item.name"
          class="flex items-center justify-between gap-4 py-1"
        >
          <div class="flex items-center gap-2">
            <span
              class="h-2.5 w-2.5 rounded-full"
              :style="{ backgroundColor: item.color }"
            ></span>
            <span>{{ item.name }}</span>
          </div>
          <span class="font-medium">{{ item.value }}</span>
        </div>
      </div>

      <div
        v-if="pointCount === 0"
        class="absolute inset-0 flex items-center justify-center rounded-[20px] border border-dashed border-slate-200 bg-white/50 text-sm text-slate-400"
      >
        {{ emptyText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"
import { useElementSize } from "@vueuse/core"

type SeriesItem = {
	name: string
	values: number[]
	color: string
	fillFrom?: string
	fillTo?: string
	valueFormatter?: (value: number) => string
}

const props = withDefaults(
	defineProps<{
		xLabels: string[]
		tooltipLabels?: string[]
		series: SeriesItem[]
		yFormatter?: (value: number) => string
		emptyText?: string
		height?: number
	}>(),
	{
		tooltipLabels: () => [],
		emptyText: "暂无监控数据",
		height: 420
	}
)

const containerRef = ref<HTMLElement | null>(null)
const scrollContainerRef = ref<HTMLElement | null>(null)
const { width: containerWidth } = useElementSize(containerRef)
const activeIndex = ref<number | null>(null)
const scrollLeft = ref(0)

const viewHeight = computed(() => props.height)
const padding = { top: 20, right: 18, bottom: 40, left: 54 }
const chartIdSeed = `svg-chart-${Math.random().toString(36).slice(2, 10)}`

const pointCount = computed(() => {
	const count = props.xLabels.length
	return count > 0 ? count : Math.max(...props.series.map(item => item.values.length), 0)
})

// 每个数据点最小占用宽度（支持水平滚动）
const minPointWidth = 40
const viewWidth = computed(() => {
	const minW = containerWidth.value || 960
	return Math.max(minW, pointCount.value * minPointWidth + padding.left + padding.right)
})

const allValues = computed(() =>
	props.series.reduce<number[]>((result, item) => {
		item.values.forEach(value => {
			if (Number.isFinite(value)) {
				result.push(value)
			}
		})
		return result
	}, [])
)
const minValue = computed(() => {
	if (allValues.value.length === 0) return 0
	const min = Math.min(...allValues.value)
	return min >= 0 ? 0 : min
})
const maxValue = computed(() => {
	if (allValues.value.length === 0) return 100
	const max = Math.max(...allValues.value)
	const base = max <= 0 ? 1 : max
	return base * 1.1
})
const valueRange = computed(() => {
	const range = maxValue.value - minValue.value
	return range === 0 ? 1 : range
})

const yTicks = computed(() => {
	const count = 4
	return Array.from(
		{ length: count + 1 },
		(_, index) => minValue.value + (valueRange.value / count) * index
	).reverse()
})

const labelIndices = computed(() => {
	if (pointCount.value <= 0) return []
	const maxLabels = Math.max(6, Math.floor(viewWidth.value / 120)) // 动态计算可见的标签数量，防止重叠
	if (pointCount.value <= maxLabels) {
		return Array.from({ length: pointCount.value }, (_, index) => index)
	}
	const step = Math.max(1, Math.floor((pointCount.value - 1) / (maxLabels - 1)))
	const result = Array.from({ length: maxLabels }, (_, index) => Math.min(index * step, pointCount.value - 1))
	result[result.length - 1] = pointCount.value - 1
	return Array.from(new Set(result))
})

const normalizedSeries = computed(() =>
	props.series.map((series, index) => {
		const gradientId = `${chartIdSeed}-${index}`
		return {
			...series,
			gradientId,
			fillFrom: series.fillFrom || `${series.color}33`,
			fillTo: series.fillTo || `${series.color}05`,
			linePath: buildLinePath(series.values),
			areaPath: buildAreaPath(series.values)
		}
	})
)

const tooltipTitle = computed(() => {
	if (activeIndex.value === null) return ""
	return props.tooltipLabels[activeIndex.value] || props.xLabels[activeIndex.value] || ""
})

const tooltipItems = computed(() => {
	if (activeIndex.value === null) return []
	return normalizedSeries.value.map(series => {
		const value = series.values[activeIndex.value as number] || 0
		return {
			name: series.name,
			color: series.color,
			value: series.valueFormatter ? series.valueFormatter(value) : formatY(value)
		}
	})
})

const tooltipStyle = computed(() => {
	if (activeIndex.value === null) return {}
	const absoluteX = getX(activeIndex.value)
	const visibleX = absoluteX - scrollLeft.value
	const clientW = scrollContainerRef.value?.clientWidth || containerWidth.value || 960
	const alignRight = visibleX > clientW / 2

	if (alignRight) {
		return {
			top: "16px",
			right: `${viewWidth.value - absoluteX + 8}px`
		}
	} else {
		return {
			top: "16px",
			left: `${absoluteX + 8}px`
		}
	}
})

function getX(index: number) {
	if (pointCount.value <= 1) {
		return padding.left
	}
	const width = viewWidth.value - padding.left - padding.right
	return padding.left + (width / (pointCount.value - 1)) * index
}

function getY(value: number) {
	const height = viewHeight.value - padding.top - padding.bottom
	return padding.top + ((maxValue.value - value) / valueRange.value) * height
}

function formatY(value: number) {
	return props.yFormatter ? props.yFormatter(value) : `${Number(value).toFixed(2)}`
}

function buildLinePath(values: number[]) {
	if (values.length === 0) return ""
	return values.map((value, index) => `${index === 0 ? "M" : "L"} ${getX(index)} ${getY(value)}`).join(" ")
}

function buildAreaPath(values: number[]) {
	if (values.length === 0) return ""
	const topPath = buildLinePath(values)
	const lastX = getX(values.length - 1)
	const firstX = getX(0)
	const bottomY = viewHeight.value - padding.bottom
	return `${topPath} L ${lastX} ${bottomY} L ${firstX} ${bottomY} Z`
}

function handleScroll(event: Event) {
	scrollLeft.value = (event.target as HTMLElement).scrollLeft
}

function handleMove(event: MouseEvent) {
	if (!scrollContainerRef.value || pointCount.value === 0) return
	const rect = scrollContainerRef.value.getBoundingClientRect()
	const absoluteX = event.clientX - rect.left + scrollContainerRef.value.scrollLeft
	const usableWidth = viewWidth.value - padding.left - padding.right
	const startX = padding.left
	const progress = Math.min(Math.max(absoluteX - startX, 0), usableWidth)
	const ratio = usableWidth <= 0 ? 0 : progress / usableWidth
	activeIndex.value = Math.min(pointCount.value - 1, Math.max(0, Math.round(ratio * (pointCount.value - 1))))
}
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
	height: 8px;
}
.custom-scrollbar::-webkit-scrollbar-track {
	background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
	background: rgba(148, 163, 184, 0.2);
	border-radius: 4px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
	background: rgba(148, 163, 184, 0.4);
}
</style>
