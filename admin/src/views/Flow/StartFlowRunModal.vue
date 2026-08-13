<script setup lang="ts">
import { reactive, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { Flow } from "@/api/interface/flow"
import { createFlowRun } from "@/api/modules/flow"
import { flowMessages } from "./flowMessages"

const props = defineProps<{ show: boolean; flow: Flow.Item | null }>()
const emit = defineEmits<{ "update:show": [value: boolean]; success: [run: Flow.Run] }>()
const { t } = useI18n({ messages: flowMessages })
const message = useMessage()
const saving = ref(false)
const form = reactive({ sourceCommit: "", version: "" })
const commitPattern = /^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$/

function close() {
	emit("update:show", false)
}

async function submit() {
	if (!props.flow) return
	const sourceCommit = form.sourceCommit.trim()
	if (!commitPattern.test(sourceCommit)) {
		message.warning(t("flow.commitInvalid"))
		return
	}
	saving.value = true
	try {
		const response = await createFlowRun({
			flowId: props.flow.id,
			sourceCommit,
			version: form.version.trim() || undefined
		})
		message.success(t("flow.runCreated", { id: response.data.id, version: response.data.version }))
		close()
		emit("success", response.data)
	} catch {
		message.error(t("flow.runCreateFailed"))
	} finally {
		saving.value = false
	}
}

watch(() => props.show, visible => {
	if (visible) Object.assign(form, { sourceCommit: "", version: "" })
})
</script>

<template>
	<n-modal :show="show" preset="card" style="width: min(560px, calc(100vw - 32px))" :title="t('flow.startRunTitle')" @update:show="emit('update:show', $event)">
		<div class="space-y-5">
			<n-alert type="info" :title="flow?.name">{{ t("flow.startRunDescription") }}</n-alert>
			<n-form-item :label="t('flow.sourceCommit')" required>
				<n-input v-model:value="form.sourceCommit" :placeholder="t('flow.sourceCommitPlaceholder')" />
			</n-form-item>
			<n-form-item :label="t('flow.runVersion')">
				<n-input v-model:value="form.version" :placeholder="t('flow.runVersionPlaceholder')" />
				<template #feedback>{{ t("flow.runVersionHelper") }}</template>
			</n-form-item>
		</div>
		<template #footer>
			<div class="flex justify-end gap-2">
				<n-button @click="close">{{ t("flow.cancel") }}</n-button>
				<n-button type="primary" :loading="saving" @click="submit">{{ t("flow.startRunConfirm") }}</n-button>
			</div>
		</template>
	</n-modal>
</template>
