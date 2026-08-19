<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { TagProps } from "naive-ui"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{ status: string; compact?: boolean }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const knownStatuses = [
	"active",
	"queued",
	"pending_approval",
	"running",
	"delivering",
	"completed",
	"failed",
	"cancelled",
]
const normalizedStatus = computed(() => (knownStatuses.includes(props.status) ? props.status : "unknown"))
const tagType = computed<TagProps["type"]>(() => {
	if (normalizedStatus.value === "completed") return "success"
	if (normalizedStatus.value === "failed") return "error"
	if (["pending_approval", "cancelled"].includes(normalizedStatus.value)) return "warning"
	if (["queued", "running", "delivering"].includes(normalizedStatus.value)) return "info"
	return "default"
})
const isActive = computed(() => ["queued", "running", "delivering"].includes(normalizedStatus.value))
const isRunning = computed(() => ["running", "delivering"].includes(normalizedStatus.value))
const compactColor = computed(() => {
	if (normalizedStatus.value === "completed") return "text-emerald-500"
	if (normalizedStatus.value === "failed") return "text-red-500"
	if (["pending_approval", "cancelled"].includes(normalizedStatus.value)) return "text-amber-500"
	if (["queued", "running", "delivering"].includes(normalizedStatus.value)) return "text-blue-500"
	return "text-slate-400"
})
</script>

<template>
	<n-tooltip v-if="compact">
		<template #trigger>
			<span
				class="relative inline-flex h-4 w-4 shrink-0 items-center justify-center"
				:class="compactColor"
				:aria-label="t(`code.taskStatus_${normalizedStatus}`)"
			>
				<span
					v-if="isRunning"
					class="absolute inset-0 animate-spin rounded-full border border-current border-r-transparent motion-reduce:animate-none"
				/>
				<span
					class="h-2 w-2 rounded-full bg-current"
					:class="isRunning ? 'animate-pulse motion-reduce:animate-none' : ''"
				/>
			</span>
		</template>
		{{ t(`code.taskStatus_${normalizedStatus}`) }}
	</n-tooltip>
	<n-tag v-else size="small" :type="tagType" round :bordered="false">
		<span
			class="relative mr-1.5 inline-flex shrink-0 items-center justify-center"
			:class="isRunning ? 'h-3.5 w-3.5' : 'h-1.5 w-1.5'"
		>
			<span
				v-if="isRunning"
				class="absolute inset-0 animate-spin rounded-full border border-current border-r-transparent motion-reduce:animate-none"
			/>
			<span
				class="inline-block h-1.5 w-1.5 rounded-full bg-current"
				:class="isActive ? 'animate-pulse motion-reduce:animate-none' : ''"
			/>
		</span>
		{{ t(`code.taskStatus_${normalizedStatus}`) }}
	</n-tag>
</template>
