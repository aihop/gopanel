<script setup lang="ts">
import { computed } from "vue"
import { codeProjectColor } from "../projectColor"

const props = defineProps<{
	projectId: number
	name: string
}>()

const color = computed(() => codeProjectColor(props.projectId))
</script>

<template>
	<span
		class="inline-flex min-w-0 items-center gap-1.5 rounded-full border px-2 py-0.5"
		:style="{ backgroundColor: `${color}14`, borderColor: `${color}2b` }"
	>
		<span
			class="h-2 w-2 shrink-0 rounded-full"
			aria-hidden="true"
			:style="{ backgroundColor: color, boxShadow: `0 0 0 3px ${color}1f` }"
		/>
		<span class="min-w-0 truncate">{{ name }}</span>
	</span>
</template>
