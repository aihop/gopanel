<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodexRuntimeState } from "@/api/interface/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const props = defineProps<{
	runtimeState: CodexRuntimeState | null
	runtimeError: boolean
	nativeProtocol: boolean
	hasControl: boolean
	reconnecting: boolean
	connectionFailed: boolean
}>()
defineEmits<{ reconnect: []; takeControl: [] }>()
const { t } = useI18n({ messages: codeProjectMessages })

const runtimeTagType = computed(() => {
	if (props.runtimeState?.responseState === "failed") return "error"
	if (props.runtimeState?.responseState === "needsInput") return "warning"
	if (props.runtimeState?.responseState === "completed") return "success"
	return "info"
})

const formatTokens = (count: number) => new Intl.NumberFormat().format(count)
</script>

<template>
	<!--
		终端顶部的状态条：原生执行器运行状态 + 连接/控制权操作 + token 计数。
		从 CodeTerminal 拆出来是为了守住 500 行门禁 —— 它是纯展示，
		所有状态由父组件传入，自己不碰 xterm 也不碰 WebSocket。
	-->
	<div
		class="flex min-h-12 items-center justify-between gap-3 border-b border-slate-700 bg-slate-900 px-4 py-2 text-slate-300"
	>
		<div class="flex min-w-0 items-center gap-3">
			<n-tag v-if="runtimeState" :type="runtimeTagType" size="small" round :bordered="false">
				{{ t(`code.codexState_${runtimeState.responseState}`) }}
			</n-tag>
			<span v-else class="text-xs text-slate-400">
				{{ t(runtimeError ? "code.codexRuntimeUnavailable" : "code.codexRuntimeStarting") }}
			</span>
			<span v-if="runtimeState?.awaitingApproval" class="truncate text-xs font-medium text-amber-300">
				{{ t("code.codexApprovalHint") }}
			</span>
			<span v-else-if="runtimeState?.lastAssistantPreview" class="truncate text-xs text-slate-400">
				{{ runtimeState.lastAssistantPreview }}
			</span>
		</div>
		<div class="flex shrink-0 items-center gap-3 text-xs text-slate-400">
			<n-button v-if="connectionFailed && !reconnecting" size="tiny" type="warning" @click="$emit('reconnect')">
				{{ t("code.reconnectTerminal") }}
			</n-button>
			<n-tag v-else-if="reconnecting" size="small" type="warning" :bordered="false">
				{{ t("code.terminalReconnecting") }}
			</n-tag>
			<n-tag v-else-if="nativeProtocol && hasControl" size="small" type="success" :bordered="false">
				{{ t("code.terminalControlling") }}
			</n-tag>
			<n-button v-else-if="nativeProtocol" size="tiny" type="warning" @click="$emit('takeControl')">
				{{ t("code.takeTerminalControl") }}
			</n-button>
			<template v-if="runtimeState">
				<span v-if="runtimeState.model">{{ runtimeState.model }}</span>
				<span v-if="runtimeState.totalTokens">
					{{ t("code.codexTokenUsage", { count: formatTokens(runtimeState.totalTokens) }) }}
				</span>
				<span v-if="runtimeState.cachedInputTokens" class="hidden md:inline">
					{{ t("code.codexCachedTokens", { count: formatTokens(runtimeState.cachedInputTokens) }) }}
				</span>
			</template>
		</div>
	</div>
</template>
