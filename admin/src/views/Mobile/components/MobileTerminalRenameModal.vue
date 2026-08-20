<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { updateMobileSessionTitle } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"
import { mobileTerminalMessages } from "../mobileTerminalMessages"

const props = defineProps<{
	sessionId: number
	taskName: string
}>()
const emit = defineEmits<{ renamed: [] }>()
const { t } = useI18n({
	messages: {
		zh: { mobile: { ...mobileMessages.zh.mobile, ...mobileTerminalMessages.zh.mobile } },
		en: { mobile: { ...mobileMessages.en.mobile, ...mobileTerminalMessages.en.mobile } },
	},
})
const message = useMessage()
const show = ref(false)
const title = ref("")
const loading = ref(false)

function open() {
	title.value = props.taskName
	show.value = true
}

async function submit() {
	const next = title.value.trim()
	if (!next || loading.value) return
	loading.value = true
	try {
		await updateMobileSessionTitle(props.sessionId, next)
		show.value = false
		message.success(t("mobile.sessionRenameSuccess"))
		emit("renamed")
	} catch {
		void 0
	} finally {
		loading.value = false
	}
}

defineExpose({ open })
</script>

<template>
	<n-modal
		v-model:show="show"
		preset="card"
		style="width: min(92vw, 420px)"
		:title="t('mobile.renameSession')"
	>
		<n-input
			v-model:value="title"
			maxlength="255"
			show-count
			autofocus
			:placeholder="t('mobile.sessionNamePlaceholder')"
			@keydown.enter.prevent="submit"
		/>
		<template #footer>
			<div class="flex justify-end gap-2">
				<n-button @click="show = false">{{ t("mobile.cancel") }}</n-button>
				<n-button
					type="primary"
					:loading="loading"
					:disabled="!title.trim()"
					@click="submit"
				>
					{{ t("mobile.renameSession") }}
				</n-button>
			</div>
		</template>
	</n-modal>
</template>
