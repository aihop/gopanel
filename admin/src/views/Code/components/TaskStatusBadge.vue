<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { TagProps } from "naive-ui"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{ status: string }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const knownStatuses = ["active", "queued", "pending_approval", "running", "completed", "failed", "cancelled"]
const normalizedStatus = computed(() => (knownStatuses.includes(props.status) ? props.status : "unknown"))
const tagType = computed<TagProps["type"]>(() => {
	if (normalizedStatus.value === "completed") return "success"
	if (normalizedStatus.value === "failed") return "error"
	if (["pending_approval", "cancelled"].includes(normalizedStatus.value)) return "warning"
	if (["queued", "running"].includes(normalizedStatus.value)) return "info"
	return "default"
})
const isActive = computed(() => ["queued", "running"].includes(normalizedStatus.value))
</script>

<template>
	<n-tag size="small" :type="tagType" round :bordered="false">
		<span
			class="mr-1.5 inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-current"
			:class="isActive ? 'animate-pulse' : ''"
		/>
		{{ t(`code.taskStatus_${normalizedStatus}`) }}
	</n-tag>
</template>
