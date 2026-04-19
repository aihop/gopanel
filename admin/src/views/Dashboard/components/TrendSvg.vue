<template>
  <div class="w-full rounded-2xl border border-slate-100 bg-slate-50/70 p-4">
    <div class="flex items-center justify-between">
      <div>
        <div class="text-sm font-medium text-slate-600">{{ title }}</div>
        <div class="mt-1 text-2xl font-semibold fg-base-100">{{ metric }}</div>
      </div>
      <div
        v-if="badge"
        class="rounded-full bg-base-100 px-3 py-1 text-xs font-medium text-slate-500 shadow-sm"
      >
        {{ badge }}
      </div>
    </div>
    <div class="mt-4">
      <svg
        class="h-[260px] w-full"
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
      >
        <line
          v-for="grid in grids"
          :key="grid"
          x1="0"
          :x2="100"
          :y1="grid"
          :y2="grid"
          stroke="rgb(226 232 240)"
          stroke-dasharray="3 3"
        />
        <polygon
          :points="areaPoints(primaryNormalizedPoints)"
          class="fill-sky-100/30"
        />
        <polyline
          :points="linePoints(primaryNormalizedPoints)"
          fill="none"
          stroke="rgb(14 165 233)"
          stroke-width="0.8"
          stroke-linejoin="round"
          stroke-linecap="round"
        />
        <polyline
          v-if="secondaryNormalizedPoints.length"
          :points="linePoints(secondaryNormalizedPoints)"
          fill="none"
          stroke="rgb(99 102 241)"
          stroke-width="0.8"
          stroke-linejoin="round"
          stroke-linecap="round"
        />
        <circle
          v-for="point in primaryNormalizedPoints"
          :key="point.key"
          :cx="point.x"
          :cy="point.y"
          r="0.6"
          fill="rgb(2 132 199)"
        />
        <circle
          v-for="point in secondaryNormalizedPoints"
          :key="`secondary-${point.key}`"
          :cx="point.x"
          :cy="point.y"
          r="0.6"
          fill="rgb(79 70 229)"
        />
      </svg>
    </div>
    <div
      v-if="primaryLabel || secondaryLabel"
      class="mt-3 flex flex-wrap items-center gap-4 text-xs text-slate-500"
    >
      <div
        v-if="primaryLabel"
        class="flex items-center gap-2"
      >
        <span class="h-2 w-2 rounded-full bg-sky-500"></span>
        <span>{{ primaryLabel }}</span>
      </div>
      <div
        v-if="secondaryLabel"
        class="flex items-center gap-2"
      >
        <span class="h-2 w-2 rounded-full bg-indigo-500"></span>
        <span>{{ secondaryLabel }}</span>
      </div>
    </div>
    <div class="mt-3 flex items-center justify-between text-xs text-slate-400">
      <span>{{ startLabel }}</span>
      <span>{{ middleLabel }}</span>
      <span>{{ endLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue"

const props = withDefaults(
	defineProps<{
		title: string
		metric: string
		points: number[]
		secondaryPoints?: number[]
		labels?: string[]
		badge?: string
		primaryLabel?: string
		secondaryLabel?: string
	}>(),
	{
		secondaryPoints: () => [],
		labels: () => [],
		badge: "",
		primaryLabel: "",
		secondaryLabel: ""
	}
)

const grids = [20, 40, 60, 80]

const safePoints = computed(() =>
	(props.points.length ? props.points : [0, 0]).map(point => (Number.isFinite(point) ? point : 0))
)
const safeSecondaryPoints = computed(() =>
	(props.secondaryPoints.length ? props.secondaryPoints : []).map(point => (Number.isFinite(point) ? point : 0))
)

const valueRange = computed(() => {
	const values = [...safePoints.value, ...safeSecondaryPoints.value]
	const max = Math.max(...values)
	const min = Math.min(...values)
	return {
		max,
		min,
		range: max - min || 1
	}
})

function normalize(values: number[]) {
	const { min, range } = valueRange.value
	return values.map((value, index) => {
		const x = values.length === 1 ? 50 : (index / (values.length - 1)) * 100
		const y = 90 - ((value - min) / range) * 70
		return {
			key: `${index}-${value}`,
			x: Number(x.toFixed(2)),
			y: Number(y.toFixed(2))
		}
	})
}

const primaryNormalizedPoints = computed(() => normalize(safePoints.value))
const secondaryNormalizedPoints = computed(() => normalize(safeSecondaryPoints.value))

function linePoints(points: Array<{ x: number; y: number }>) {
	return points.map(point => `${point.x},${point.y}`).join(" ")
}

function areaPoints(points: Array<{ x: number; y: number }>) {
	return `0,90 ${linePoints(points)} 100,90`
}

const startLabel = computed(() => props.labels[0] || "--")
const middleLabel = computed(() => props.labels[Math.floor(props.labels.length / 2)] || "--")
const endLabel = computed(() => props.labels[props.labels.length - 1] || "--")
</script>
