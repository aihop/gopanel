<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { Flow } from "@/api/interface/flow"
import { usePipelineLogStream } from "@/composables/usePipelineLogStream"
import { flowMessages } from "./flowMessages"

const props = defineProps<{ recordId: number; runStatus: Flow.RunStatus }>()
const { t } = useI18n({ messages: flowMessages })
const message = useMessage()
const terminal = ref<HTMLElement | null>(null)
const { logs, connecting, connected, streamError, connect, close } = usePipelineLogStream({
	canReconnect: () => props.runStatus === "queued" || props.runStatus === "running",
	onFinished: scrollToBottom,
	onLog: scrollToBottom,
	truncatedMessage: t("flow.terminalTruncated")
})

const terminalStatus = computed(() => {
	if (!props.recordId) return t("flow.terminalWaiting")
	if (connecting.value) return t("flow.terminalConnecting")
	if (connected.value) return t("flow.terminalLive")
	if (streamError.value) return t("flow.terminalDisconnected")
	return t("flow.terminalFinished")
})

function scrollToBottom() {
	nextTick(() => {
		if (terminal.value) terminal.value.scrollTop = terminal.value.scrollHeight
	})
}

async function copyLogs() {
	if (!logs.value.length) return
	try {
		await navigator.clipboard.writeText(logs.value.join("\n"))
		message.success(t("flow.terminalCopied"))
	} catch {
		message.error(t("flow.terminalCopyFailed"))
	}
}

watch(() => props.recordId, recordId => connect(recordId), { immediate: true })
watch(() => props.runStatus, status => {
	if (status !== "queued" && status !== "running" && streamError.value) close()
})
</script>

<template>
	<section>
		<div class="mb-3 flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<h3 class="font-semibold fg-base-100">{{ t("flow.executionTerminal") }}</h3>
				<n-tag size="small" :type="connected ? 'success' : streamError ? 'error' : 'default'">{{ terminalStatus }}</n-tag>
			</div>
			<div class="flex items-center gap-2">
				<n-button v-if="streamError && recordId" size="tiny" quaternary @click="connect(recordId)">{{ t("flow.terminalReconnect") }}</n-button>
				<n-button size="tiny" quaternary :disabled="!logs.length" @click="copyLogs">{{ t("flow.terminalCopy") }}</n-button>
			</div>
		</div>
		<div ref="terminal" class="h-[360px] overflow-y-auto rounded-xl bg-[#0f1117] p-4 font-mono text-xs leading-5 text-slate-300 shadow-inner">
			<div v-if="!recordId" class="text-slate-500">{{ t("flow.terminalWaitingDescription") }}</div>
			<div v-else-if="!logs.length && !streamError" class="text-slate-500">{{ t("flow.terminalConnectingDescription") }}</div>
			<div v-else-if="!logs.length && streamError" class="text-red-400">{{ t("flow.terminalLoadFailed") }}</div>
			<div v-for="(line, index) in logs" :key="`${index}-${line}`" class="whitespace-pre-wrap break-words">{{ line }}</div>
		</div>
	</section>
</template>
