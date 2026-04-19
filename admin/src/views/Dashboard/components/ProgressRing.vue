<template>
  <div class="flex flex-col items-center gap-3">
    <div
      class="relative"
      :style="{ width: `${size}px`, height: `${size}px` }"
    >
      <svg
        class="h-full w-full -rotate-90"
        :viewBox="`0 0 ${size} ${size}`"
      >
        <circle
          :cx="center"
          :cy="center"
          :r="radius"
          class="fill-none stroke-slate-100"
          :stroke-width="strokeWidth"
        />
        <circle
          :cx="center"
          :cy="center"
          :r="radius"
          class="fill-none transition-all duration-500 ease-out"
          :class="strokeClass"
          stroke-linecap="round"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="dashOffset"
          :stroke-width="strokeWidth"
        />
      </svg>
      <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
        <div class="text-xs font-medium tracking-wide text-slate-500">{{ title }}</div>
        <div class="text-2xl font-semibold fg-base-100">{{ displayValue }}</div>
      </div>
    </div>
    <div
      v-if="subtitle"
      class="text-center text-sm text-slate-500"
    >
      {{ subtitle }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue"

const props = withDefaults(
	defineProps<{
		title: string
		value: number
		subtitle?: string
		size?: number
		strokeWidth?: number
	}>(),
	{
		size: 160,
		strokeWidth: 12,
		subtitle: ""
	}
)

const safeValue = computed(() => {
	if (Number.isNaN(props.value)) return 0
	return Math.max(0, Math.min(100, props.value))
})

const center = computed(() => props.size / 2)
const radius = computed(() => center.value - props.strokeWidth / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
const dashOffset = computed(() => circumference.value * (1 - safeValue.value / 100))

const strokeClass = computed(() => {
	if (safeValue.value >= 85) return "stroke-rose-500"
	if (safeValue.value >= 65) return "stroke-amber-500"
	return "stroke-emerald-500"
})

const displayValue = computed(() => `${safeValue.value.toFixed(1).replace(/\.0$/, "")}%`)
</script>
