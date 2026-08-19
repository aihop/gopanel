<script setup lang="ts">
import { useMediaQuery } from "@vueuse/core"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodexRuntimeState } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { codeTerminalMessages } from "../codeTerminalMessages"
import CodeTerminalProgressSummary from "./CodeTerminalProgressSummary.vue"

const props = defineProps<{
	runtimeState: CodexRuntimeState | null
	runtimeError: boolean
	nativeProtocol: boolean
	hasControl: boolean
	reconnecting: boolean
	connectionFailed: boolean
	terminalInactive: boolean
	reserveTopRightActions?: boolean
}>()
defineEmits<{ reconnect: []; resume: []; takeControl: [] }>()
const { t } = useI18n({ messages: codeTerminalMessages })
const isDesktop = useMediaQuery("(min-width: 768px)")

const runtimeTagType = computed(() => {
	if (props.runtimeState?.responseState === "failed") return "error"
	if (props.runtimeState?.responseState === "needsInput") return "warning"
	if (props.runtimeState?.responseState === "completed") return "success"
	return "info"
})

const formatTokens = (count: number) => new Intl.NumberFormat().format(count)
const cacheHitRate = computed(() => {
	if (!props.runtimeState?.inputTokens) return null
	return new Intl.NumberFormat(undefined, {
		style: "percent",
		maximumFractionDigits: 1,
	}).format(Math.min(props.runtimeState.cachedInputTokens / props.runtimeState.inputTokens, 1))
})
</script>

<template>
	<!--
		终端顶部的状态条：原生执行器运行状态 + 连接/控制权操作 + token 计数。
		从 CodeTerminal 拆出来是为了守住 500 行门禁 —— 它是纯展示，
		所有状态由父组件传入，自己不碰 xterm 也不碰 WebSocket。
	-->
	<div
		class="flex min-h-12 items-center justify-between gap-3 border-b border-slate-700 bg-slate-900 px-4 py-2 text-slate-300"
		:style="{ paddingRight: reserveTopRightActions ? '5rem' : undefined }"
	>
		<div class="flex min-w-0 flex-1 items-center gap-3 overflow-hidden">
			<n-tag v-if="runtimeState" :type="runtimeTagType" size="small" round :bordered="false">
				{{ t(`code.codexState_${runtimeState.responseState}`) }}
			</n-tag>
			<span v-else class="text-xs text-slate-400">
				{{ t(runtimeError ? "code.codexRuntimeUnavailable" : "code.codexRuntimeStarting") }}
			</span>
			<span v-if="runtimeState?.awaitingApproval" class="truncate text-xs font-medium text-amber-300">
				{{ t("code.codexApprovalHint") }}
			</span>
			<CodeTerminalProgressSummary
				v-else-if="runtimeState?.progress && (runtimeState.progress.totalSteps || runtimeState.progress.changedFiles)"
				:progress="runtimeState.progress"
			/>
			<span v-else-if="runtimeState?.lastAssistantPreview" class="truncate text-xs text-slate-400">
				{{ runtimeState.lastAssistantPreview }}
			</span>
		</div>
		<div class="flex shrink-0 items-center gap-3 text-xs text-slate-400">
			<n-button v-if="terminalInactive" size="tiny" type="primary" @click="$emit('resume')">
				{{ t("code.resumeTerminalSession") }}
			</n-button>
			<n-button
				v-else-if="connectionFailed && !reconnecting"
				size="tiny"
				type="warning"
				@click="$emit('reconnect')"
			>
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
				<n-popover
					v-if="runtimeState.totalTokens"
					:trigger="isDesktop ? 'hover' : 'click'"
					placement="bottom-end"
					style="width: 280px"
				>
					<template #trigger>
						<n-button
							quaternary
							circle
							size="tiny"
							:title="t('code.tokenUsageDetails')"
							:aria-label="t('code.tokenUsageDetails')"
						>
							<template #icon>
								<Icon name="mdi:chart-donut-variant" :size="16" />
							</template>
						</n-button>
					</template>
					<div class="space-y-3 p-1">
						<div class="flex items-center justify-between gap-4">
							<span class="text-sm font-medium text-[var(--n-text-color)]">
								{{ t("code.tokenUsageDetails") }}
							</span>
							<span class="text-sm font-semibold text-[var(--n-text-color)]">
								{{ formatTokens(runtimeState.totalTokens) }}
							</span>
						</div>
						<div class="grid grid-cols-2 gap-x-5 gap-y-2 text-xs">
							<span class="text-[var(--n-text-color-3)]">{{ t("code.inputTokens") }}</span>
							<span class="text-right text-[var(--n-text-color-2)]">
								{{ formatTokens(runtimeState.inputTokens) }}
							</span>
							<span class="text-[var(--n-text-color-3)]">{{ t("code.outputTokens") }}</span>
							<span class="text-right text-[var(--n-text-color-2)]">
								{{ formatTokens(runtimeState.outputTokens) }}
							</span>
							<span class="text-[var(--n-text-color-3)]">{{ t("code.cachedInputTokens") }}</span>
							<span class="text-right text-[var(--n-text-color-2)]">
								{{ formatTokens(runtimeState.cachedInputTokens) }}
							</span>
							<span class="text-[var(--n-text-color-3)]">{{ t("code.reasoningTokens") }}</span>
							<span class="text-right text-[var(--n-text-color-2)]">
								{{ formatTokens(runtimeState.reasoningTokens) }}
							</span>
							<template v-if="cacheHitRate">
								<span class="text-[var(--n-text-color-3)]">{{ t("code.cacheHitRate") }}</span>
								<span class="text-right text-[var(--n-text-color-2)]">{{ cacheHitRate }}</span>
							</template>
						</div>
					</div>
				</n-popover>
			</template>
		</div>
	</div>
</template>
