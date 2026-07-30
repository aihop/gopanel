<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getCodeSession, updateCodeSessionApprovalPolicy } from "@/api/modules/code"
import type { CodeApprovalPolicy } from "@/api/interface/code"
import { newCodeSessionMessages } from "../newCodeSessionMessages"

const props = defineProps<{ sessionId: number }>()
const { t } = useI18n({ messages: newCodeSessionMessages })
const message = useMessage()
const approvalPolicy = ref<CodeApprovalPolicy | null>(null)
const executorId = ref("")
const loading = ref(false)
const saving = ref(false)
const loadError = ref(false)
let loadSequence = 0

const options = computed(() => {
	const values: CodeApprovalPolicy[] = executorId.value === "codex"
		? ["manual", "safe_auto", "full_auto"]
		: approvalPolicy.value && approvalPolicy.value !== "full_auto"
			? [approvalPolicy.value, "full_auto"]
			: ["full_auto"]
	return values.map(value => ({
		label: t(`code.approvalPolicy_${value}`),
		value,
		disabled: executorId.value !== "codex" && value !== "full_auto"
	}))
})

const loadPolicy = async () => {
	const sequence = ++loadSequence
	const sessionId = props.sessionId
	loading.value = true
	saving.value = false
	loadError.value = false
	approvalPolicy.value = null
	try {
		const response = await getCodeSession(sessionId)
		if (sequence !== loadSequence || sessionId !== props.sessionId) return
		executorId.value = response.data.session.agentName || ""
		approvalPolicy.value = response.data.session.approvalPolicy || "safe_auto"
	} catch {
		if (sequence !== loadSequence || sessionId !== props.sessionId) return
		loadError.value = true
		message.error(t("code.approvalPolicyLoadFailed"))
	} finally {
		if (sequence === loadSequence) loading.value = false
	}
}

const updatePolicy = async (value: CodeApprovalPolicy) => {
	const sessionId = props.sessionId
	const previousPolicy = approvalPolicy.value
	approvalPolicy.value = value
	saving.value = true
	try {
		const response = await updateCodeSessionApprovalPolicy(sessionId, value)
		if (sessionId !== props.sessionId) return
		approvalPolicy.value = response.data.approvalPolicy
		message.success(t("code.approvalPolicyUpdated"))
	} catch {
		if (sessionId !== props.sessionId) return
		approvalPolicy.value = previousPolicy
		message.error(t("code.approvalPolicyUpdateFailed"))
	} finally {
		if (sessionId === props.sessionId) saving.value = false
	}
}

watch(() => props.sessionId, loadPolicy, { immediate: true })
</script>

<template>
	<div class="flex items-center gap-2">
		<span class="hidden text-xs text-slate-500 xl:inline">{{ t("code.approvalPolicy") }}</span>
		<n-button v-if="loadError" size="small" secondary @click="loadPolicy">
			{{ t("code.retry") }}
		</n-button>
		<n-skeleton v-else-if="loading || approvalPolicy === null" text style="width: 132px" />
		<n-tooltip v-else>
			<template #trigger>
				<n-select
					:value="approvalPolicy"
					:options="options"
					:loading="saving"
					:disabled="saving"
					size="small"
					style="width: 160px"
					@update:value="updatePolicy"
				/>
			</template>
			{{
				executorId !== "codex"
					? t("code.executorFullAutoOnly")
					: approvalPolicy === "full_auto"
					? t("code.fullAutoWarning")
					: t("code.approvalPolicyAppliesNext")
			}}
		</n-tooltip>
	</div>
</template>
