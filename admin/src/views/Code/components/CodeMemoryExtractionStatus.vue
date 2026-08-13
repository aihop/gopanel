<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeMemoryExtractionStatus } from "@/api/interface/codeMemories"
import { codeMemoryMessages } from "../codeMemoryMessages"

const props = defineProps<{
	status: CodeMemoryExtractionStatus | null
	loading: boolean
	loadFailed: boolean
	extracting: boolean
	ready: boolean
	settingLoading: boolean
}>()
const emit = defineEmits<{ refresh: []; extract: [] }>()
const { t } = useI18n({ messages: codeMemoryMessages })

const active = computed(() => props.status?.status === "queued" || props.status?.status === "running")
const statusType = computed(() => {
	if (props.status?.status === "success") return "success"
	if (props.status?.status === "failed") return "error"
	if (props.status?.status === "queued" || props.status?.status === "running") return "info"
	return "warning"
})
const changedCount = computed(() => {
	if (!props.status) return 0
	return props.status.added + props.status.merged + props.status.replaced + props.status.archived
})
const reasonText = computed(() => {
	if (!props.status?.reason) return ""
	const knownReasons = ["disabled", "empty_transcript", "low_signal", "insufficient_growth", "extraction_failed"]
	return knownReasons.includes(props.status.reason)
		? t(`code.memoryExtractionReason.${props.status.reason}`)
		: props.status.reason
})

function formatTime(value?: string) {
	if (!value) return ""
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<template>
	<section class="rounded-xl border border-slate-200/80 p-3 dark:border-[var(--border-color)]">
		<div class="flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<span class="text-xs font-medium text-slate-600 dark:text-slate-300">
					{{ t("code.memoryExtraction") }}
				</span>
				<n-tag v-if="status" size="tiny" :type="statusType" :bordered="false">
					{{ t(`code.memoryExtractionStatus.${status.status}`) }}
				</n-tag>
			</div>
			<n-button
				size="tiny"
				:type="status?.status === 'failed' ? 'error' : 'primary'"
				secondary
				:loading="extracting || active"
				:disabled="!ready || settingLoading || loading || active"
				@click="emit('extract')"
			>
				{{ status?.status === "failed" ? t("code.retry") : t("code.memoryExtractNow") }}
			</n-button>
		</div>
		<div v-if="loading && !status" class="flex h-12 items-center justify-center"><n-spin size="small" /></div>
		<div
			v-else-if="loadFailed && !status"
			class="mt-2 flex items-center justify-between gap-2 text-xs text-red-500"
		>
			<span>{{ t("code.memoryStatusLoadFailed") }}</span>
			<n-button text type="primary" size="tiny" @click="emit('refresh')">{{ t("code.retry") }}</n-button>
		</div>
		<p v-else-if="!status || status.status === 'idle'" class="mt-2 text-[11px] text-slate-400">
			{{ t("code.memoryExtractionIdleHint") }}
		</p>
		<div v-else class="mt-2 space-y-1 text-[11px] text-slate-400">
			<p v-if="reasonText" class="break-words" :class="status.status === 'failed' ? 'text-red-500' : ''">
				{{ reasonText }}
			</p>
			<p v-if="status.status === 'success'">
				{{
					t("code.memoryExtractionResult", {
						count: changedCount,
						added: status.added,
						merged: status.merged,
						archived: status.archived
					})
				}}
			</p>
			<p v-if="status.completedAt">
				{{ t("code.memoryExtractionCompletedAt", { time: formatTime(status.completedAt) }) }}
			</p>
		</div>
		<p v-if="!settingLoading && !ready" class="mt-2 text-[11px] text-amber-500">
			{{ t("code.memoryExtractionNeedsSetting") }}
		</p>
	</section>
</template>
