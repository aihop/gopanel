<script setup lang="ts">
import { useI18n } from "vue-i18n"
import HostTerminalPanel from "@/views/Host/terminal/HostTerminalPanel.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

defineProps<{ sessionId: number | null; opening: boolean }>()
const emit = defineEmits<{ closed: []; reopen: [] }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
</script>

<template>
	<div
		class="ai-workspace-terminal-panel min-h-0 flex-1 overflow-hidden rounded-[20px] border border-slate-700 bg-[#0b1020] shadow-lg"
	>
		<HostTerminalPanel
			v-if="sessionId !== null"
			:key="sessionId"
			:session-id="sessionId"
			@closed="emit('closed')"
			@status="emit('closed')"
		/>
		<n-result
			v-else
			status="info"
			:title="t('code.projectTerminalEnded')"
			class="flex h-full items-center justify-center bg-white"
		>
			<template #footer>
				<n-button type="primary" :loading="opening" @click="emit('reopen')">
					{{ t("code.reopenTerminal") }}
				</n-button>
			</template>
		</n-result>
	</div>
</template>
