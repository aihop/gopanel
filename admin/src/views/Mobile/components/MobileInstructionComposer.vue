<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import { createMobileInstruction } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"

const props = defineProps<{ sessionId: number; disabled?: boolean }>()
const emit = defineEmits<{ sent: [] }>()
const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const content = ref("")
const sending = ref(false)

async function sendInstruction() {
	const instruction = content.value.trim()
	if (!instruction || sending.value || props.disabled) return
	sending.value = true
	try {
		await createMobileInstruction(props.sessionId, instruction)
		content.value = ""
		message.success(t("mobile.instructionQueued"))
		emit("sent")
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.instructionFailed"))
	} finally {
		sending.value = false
	}
}
</script>

<template>
	<div class="shrink-0 border-t border-white/10 bg-slate-950/95 px-2 pb-[max(8px,env(safe-area-inset-bottom))] pt-2 backdrop-blur">
		<div class="flex items-end gap-2 rounded-2xl border border-white/10 bg-white/5 p-2">
			<n-input v-model:value="content" type="textarea" :autosize="{ minRows: 1, maxRows: 4 }" :disabled="disabled" :placeholder="t('mobile.instructionPlaceholder')" :bordered="false" class="min-w-0 flex-1 text-white" @keydown.meta.enter.prevent="sendInstruction" @keydown.ctrl.enter.prevent="sendInstruction" />
			<n-button type="primary" circle :loading="sending" :disabled="disabled || !content.trim()" :title="t('mobile.sendInstruction')" @click="sendInstruction">
				<template #icon><Icon name="mdi:arrow-up" :size="20" /></template>
			</n-button>
		</div>
		<div class="mt-1 px-1 text-[10px] text-slate-500">{{ t("mobile.instructionHint") }}</div>
	</div>
</template>
