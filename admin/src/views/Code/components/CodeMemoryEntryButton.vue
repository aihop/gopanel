<script setup lang="ts">
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { codeMemoryMessages } from "../codeMemoryMessages"

const emit = defineEmits<{ open: [] }>()
const { t } = useI18n({ messages: codeMemoryMessages })
</script>

<template>
	<n-button
		secondary
		style="width: 100%; height: auto; justify-content: flex-start; padding: 10px 12px"
		@click="emit('open')"
	>
		<template #icon><Icon name="mdi:brain" :size="18" /></template>
		<div class="min-w-0 text-left">
			<div class="text-sm font-medium">{{ t("code.memoryEntry") }}</div>
			<div class="mt-0.5 truncate text-xs text-[var(--n-text-color-3)]">{{ t("code.memoryHint") }}</div>
		</div>
	</n-button>
</template>
