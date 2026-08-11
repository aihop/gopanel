<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
	title: string
	subtitle: string
	content: string
	truncated: boolean
	loading: boolean
	emptyDescription: string
	diffEmptyDescription: string
	truncatedDescription: string
	openFileLabel: string
	canOpenFile: boolean
}>()

defineEmits<{ (event: "open-file"): void }>()

const diffLines = computed(() => props.content.split("\n"))
const diffLineClass = (line: string) => {
	if (line.startsWith("+++") || line.startsWith("---")) return "text-slate-400"
	if (line.startsWith("+")) return "bg-emerald-500/10 text-emerald-300"
	if (line.startsWith("-")) return "bg-rose-500/10 text-rose-300"
	if (line.startsWith("@@")) return "text-sky-300"
	return "text-slate-300"
}
</script>

<template>
	<section class="flex min-w-0 flex-1 flex-col bg-[#0f172a] text-slate-100">
		<div v-if="title" class="flex h-12 shrink-0 items-center justify-between gap-3 border-b border-slate-700 px-4">
			<div class="min-w-0">
				<div class="truncate text-sm font-semibold">{{ title }}</div>
				<div class="text-[11px] text-slate-400">{{ subtitle }}</div>
			</div>
			<n-button v-if="canOpenFile" size="small" secondary @click="$emit('open-file')">
				{{ openFileLabel }}
			</n-button>
		</div>
		<n-spin :show="loading" class="min-h-0 flex-1">
			<div v-if="!title" class="flex h-full items-center justify-center">
				<n-empty :description="emptyDescription" />
			</div>
			<div v-else-if="!loading && !content" class="flex h-full items-center justify-center">
				<n-empty :description="diffEmptyDescription" />
			</div>
			<div v-else class="flex h-full min-h-0 flex-col">
				<n-alert v-if="truncated" type="warning" :show-icon="false" class="m-3 mb-0">
					{{ truncatedDescription }}
				</n-alert>
				<pre class="min-h-0 flex-1 overflow-auto p-4 font-mono text-xs leading-5"><span
					v-for="(line, index) in diffLines"
					:key="index"
					class="block min-w-max px-1"
					:class="diffLineClass(line)"
				>{{ line || " " }}</span></pre>
			</div>
		</n-spin>
	</section>
</template>

<style scoped>
:deep(.n-spin-container),
:deep(.n-spin-content) {
	height: 100%;
}
</style>
