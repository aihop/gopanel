<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

export type CodeWorkspaceMode = "editor" | "changes" | "terminal"

defineProps<{ value: CodeWorkspaceMode }>()
const emit = defineEmits<{ (event: "update:value", value: CodeWorkspaceMode): void }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
</script>

<template>
	<n-radio-group :value="value" size="small" @update:value="emit('update:value', $event)">
		<n-radio-button value="editor">{{ t("code.editorMode") }}</n-radio-button>
		<n-radio-button value="changes">{{ t("code.changesMode") }}</n-radio-button>
		<n-radio-button value="terminal">{{ t("code.terminalMode") }}</n-radio-button>
	</n-radio-group>
</template>
