<script setup lang="ts">
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import { mobileTerminalMessages } from "../mobileTerminalMessages"

defineProps<{ show: boolean }>()
const emit = defineEmits<{ copy: []; clear: [] }>()
const selectionMessages = {
	zh: { mobile: { ...mobileMessages.zh.mobile, ...mobileTerminalMessages.zh.mobile } },
	en: { mobile: { ...mobileMessages.en.mobile, ...mobileTerminalMessages.en.mobile } }
}
const { t } = useI18n({ messages: selectionMessages })
</script>

<template>
	<div
		v-if="show"
		class="absolute left-1/2 top-3 z-20 flex -translate-x-1/2 items-center overflow-hidden rounded-xl border border-blue-300/30 bg-slate-800/95 text-sm text-white shadow-xl backdrop-blur"
		role="toolbar"
		:aria-label="t('mobile.copyTerminalSelection')"
		@touchstart.stop
	>
		<button type="button" class="flex h-10 items-center gap-2 px-3 active:bg-white/15" @click="emit('copy')">
			<Icon name="mdi:content-copy" :size="17" />
			<span>{{ t("mobile.copyTerminalSelection") }}</span>
		</button>
		<button
			type="button"
			class="flex h-10 w-10 items-center justify-center border-l border-white/10 active:bg-white/15"
			:aria-label="t('mobile.cancel')"
			@click="emit('clear')"
		>
			<Icon name="mdi:close" :size="18" />
		</button>
	</div>
</template>
