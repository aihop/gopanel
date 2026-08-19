<script setup lang="ts">
import type { AITask, CodeApproval } from "@/api/interface/code"
import { approveCodeInstruction, getCodeSessionState } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import { useMessage } from "naive-ui"
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import TaskStatusBadge from "./TaskStatusBadge.vue"

const props = defineProps<{ task: AITask; compact?: boolean }>()
const emit = defineEmits<{ approved: [] }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const message = useMessage()
const approval = ref<CodeApproval | null>(null)
const loading = ref(false)
const loadError = ref(false)
const approving = ref(false)

async function loadApproval() {
	approval.value = null
	loadError.value = false
	if (props.task.status !== "pending_approval" || !props.task.sessionId) return
	loading.value = true
	try {
		const response = await getCodeSessionState(props.task.sessionId)
		if (response.data.pendingApproval?.taskId === props.task.id) approval.value = response.data.pendingApproval
	} catch {
		loadError.value = true
	} finally {
		loading.value = false
	}
}

async function approve() {
	if (!approval.value || approving.value) return
	approving.value = true
	try {
		const response = await approveCodeInstruction(approval.value.id)
		if (response.code !== 0) throw new Error(response.message)
		approval.value = null
		message.success(t("code.quickContinueSuccess"))
		emit("approved")
	} catch (error) {
		message.error(error instanceof Error && error.message ? error.message : t("code.quickContinueFailed"))
	} finally {
		approving.value = false
	}
}

watch(() => `${props.task.status}:${props.task.sessionId}:${props.task.id}`, () => void loadApproval(), { immediate: true })
</script>

<template>
	<div class="flex items-center gap-1.5">
		<TaskStatusBadge :status="task.status" :compact="compact" />
		<n-tooltip v-if="approval" placement="top" style="max-width: min(420px, 80vw)">
			<template #trigger>
				<n-button text type="warning" size="tiny" :loading="approving" @click.stop="approve">
					<template #icon><Icon name="mdi:check-circle-outline" :size="14" /></template>
					{{ t("code.quickContinue") }}
				</n-button>
			</template>
			<div class="whitespace-pre-wrap break-words">
				<div class="font-semibold">{{ approval.title }}</div>
				<div class="mt-1 opacity-80">{{ approval.content }}</div>
			</div>
		</n-tooltip>
		<n-tooltip v-else-if="loadError" placement="top">
			<template #trigger>
				<n-button text type="error" size="tiny" @click.stop="loadApproval">
					<template #icon><Icon name="mdi:refresh" :size="14" /></template>
					{{ t("code.retryApprovalLoad") }}
				</n-button>
			</template>
			{{ t("code.taskApprovalLoadFailed") }}
		</n-tooltip>
		<n-spin v-else-if="loading" :size="12" />
	</div>
</template>
