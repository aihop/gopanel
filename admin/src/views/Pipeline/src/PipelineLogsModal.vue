<script setup lang="ts">
import { ref, watch, nextTick, computed } from "vue"
import { NModal, NButton, NPopconfirm, useMessage, NAlert, NSpace } from "naive-ui"
import { getPipelineRecords, stopPipeline } from "@/api/modules/pipeline"
import { appsRepairPodmanSubuidAPI } from "@/api/modules/apps"
import type { Pipeline } from "@/api/interface/pipeline"
import { usePipelineLogStream } from "@/composables/usePipelineLogStream"
import { buildRuntimeDetailText } from "@/utils/runtime"
import { t } from "@/i18n"

const props = defineProps<{ show: boolean; recordId: number; pipelineId?: number | null }>()
const emit = defineEmits(["update:show", "finished", "retry"])

const terminalRef = ref<HTMLElement | null>(null)
const message = useMessage()

const isRunning = ref(false)
const isStopping = ref(false)
const runnerResult = ref<{ hostPort: number; containerId: string } | null>(null)
const currentRecord = ref<Pipeline.ResRecord | null>(null)

const repairTipVisible = ref(false)
const repairTipTitle = ref("")
const repairTipMessage = ref("")
const repairTipAction = ref("")
const repairTipOutput = ref("")
const repairing = ref(false)
const repairSuccess = ref(false)

const scrollToBottom = () => {
	nextTick(() => {
		if (terminalRef.value) {
			terminalRef.value.scrollTop = terminalRef.value.scrollHeight
		}
	})
}

const {
	logs,
	connect: connectLogStream,
	close: closeLogStream
} = usePipelineLogStream({
	canReconnect: () => false,
	onDisconnected: () => {
		isRunning.value = false
		logs.value.push(t("pipeline.disconnected"))
		emit("finished")
		scrollToBottom()
	},
	onFinished: () => {
		isRunning.value = false
		logs.value.push(t("pipeline.executionEnded"))
		emit("finished")
		scrollToBottom()
	},
	onLog: line => {
		const runnerMatch = line.match(/Runner 容器已启动：containerId=([^,\s]+), hostPort=(\d+)/)
		if (runnerMatch) {
			runnerResult.value = {
				containerId: runnerMatch[1],
				hostPort: Number(runnerMatch[2])
			}
		}
		if (line.includes("insufficient UIDs or GIDs")) {
			repairTipVisible.value = true
			repairTipTitle.value = t("pipeline.repairTitle")
			repairTipMessage.value = t("pipeline.repairMessage")
			repairTipAction.value = "subuid"
		}
		scrollToBottom()
	},
	truncatedMessage: t("pipeline.truncatedLog")
})

const handleRepair = async () => {
	if (repairing.value) return
	repairing.value = true
	repairTipOutput.value = ""
	try {
		let res: any
		if (repairTipAction.value === "subuid") {
			res = await appsRepairPodmanSubuidAPI()
		}

		if (res && res.code === 0) {
			repairTipOutput.value = res.data?.output || t("pipeline.repairExecuted")
			repairSuccess.value = true
			message.success(t("pipeline.repairSuccess"))
		} else {
			message.error(res?.msg || t("pipeline.repairFailed"))
		}
	} catch (e: any) {
		message.error(e?.message || t("pipeline.repairFailed"))
	} finally {
		repairing.value = false
	}
}

const handleRetry = () => {
	emit("retry")
}

const copyRunnerAddress = async () => {
	if (!runnerResult.value) return
	try {
		await navigator.clipboard.writeText(`127.0.0.1:${runnerResult.value.hostPort}`)
		message.success(t("pipeline.runnerAddressCopied"))
	} catch (error: any) {
		message.error(error?.message || t("pipeline.copyFailed"))
	}
}

const runnerRuntimeText = computed(() => {
	const row = currentRecord.value
	if (!row?.runnerContainerId) return ""
	return buildRuntimeDetailText(row, {
		kindFallback: "Runtime",
		userFallback: t("pipeline.imageDefault"),
		runtimePrefix: "",
		runUserPrefix: t("pipeline.runUserPrefix")
	})
})

const fetchCurrentRecord = async () => {
	if (!props.pipelineId || !props.recordId) {
		currentRecord.value = null
		return
	}
	try {
		const res = await getPipelineRecords({
			pipelineId: props.pipelineId,
			page: 1,
			limit: 100
		})
		const items = Array.isArray(res.data?.items) ? res.data.items : []
		currentRecord.value =
			items.find((item: Pipeline.ResRecord) => Number(item.id) === Number(props.recordId)) || null
	} catch (error) {
		currentRecord.value = null
	}
}

const startLogs = () => {
	isRunning.value = true
	isStopping.value = false
	runnerResult.value = null
	fetchCurrentRecord()

	repairTipVisible.value = false
	repairSuccess.value = false
	repairTipTitle.value = ""
	repairTipMessage.value = ""
	repairTipAction.value = ""
	repairTipOutput.value = ""

	connectLogStream(props.recordId)
}

const stopLogs = () => {
	closeLogStream()
	isRunning.value = false
}

const handleStopPipeline = async () => {
	isStopping.value = true
	try {
		await stopPipeline({ id: props.recordId })
		message.success(t("pipeline.stopCommandSent"))
	} catch (error: any) {
		message.error(error?.message || t("pipeline.stopFailed"))
		isStopping.value = false
	}
}

watch(
	() => props.show,
	newVal => {
		if (newVal) {
			startLogs()
		} else {
			stopLogs()
		}
	},
	{ immediate: true }
)
</script>

<template>
	<n-modal
		:show="show"
		preset="card"
		:title="t('pipeline.pipelineLogs')"
		style="width: 800px"
		class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(0,0,0,0.5)] sm:w-[90%]"
		@update:show="val => emit('update:show', val)"
	>
		<n-alert v-if="runnerResult" type="success" :title="t('pipeline.runnerStarted')" class="mb-4">
			<div class="text-sm">
				{{ t("pipeline.runnerInstanceDescription") }}
				<span class="font-mono text-emerald-700">127.0.0.1:{{ runnerResult.hostPort }}</span>
				<span class="ml-2 text-slate-500">
					{{ t("pipeline.containerId") }}{{ runnerResult.containerId.slice(0, 12) }}
				</span>
				<span v-if="runnerRuntimeText" class="ml-2 text-slate-500">{{ runnerRuntimeText }}</span>
				<n-button size="tiny" class="ml-3" type="primary" quaternary @click="copyRunnerAddress">
					{{ t("pipeline.copyAddress") }}
				</n-button>
			</div>
		</n-alert>

		<n-alert
			v-if="repairTipVisible"
			type="warning"
			:title="repairTipTitle"
			closable
			class="mb-4"
			@close="repairTipVisible = false"
		>
			<div class="text-sm">
				<div v-if="repairTipMessage">{{ repairTipMessage }}</div>
				<div
					v-if="repairTipOutput"
					class="mt-2 whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700"
				>
					{{ repairTipOutput }}
				</div>
				<n-space class="mt-3">
					<n-button
						v-if="!repairSuccess"
						size="small"
						type="primary"
						:loading="repairing"
						@click="handleRepair"
					>
						{{ t("pipeline.oneClickRepair") }}
					</n-button>
					<n-button v-else size="small" type="success" @click="handleRetry">
						{{ t("pipeline.continueExecution") }}
					</n-button>
				</n-space>
			</div>
		</n-alert>

		<div
			ref="terminalRef"
			class="h-[500px] overflow-y-auto rounded-lg bg-[#0F0F0F] p-4 font-mono text-sm leading-relaxed text-gray-300 shadow-inner"
		>
			<div v-for="(log, idx) in logs" :key="idx" class="whitespace-pre-wrap break-words">
				{{ log }}
			</div>
			<div v-if="logs.length === 0" class="italic text-gray-500">
				{{ t("pipeline.connectingLogs") }}
			</div>
		</div>

		<div class="mt-4 flex items-center justify-between">
			<div>
				<n-popconfirm
					v-if="isRunning"
					@positive-click="handleStopPipeline"
					:negative-text="t('commons.button.cancel')"
					:positive-text="t('pipeline.confirmStop')"
				>
					<template #trigger>
						<n-button type="error" :loading="isStopping" ghost>
							{{ t("pipeline.forceStopExecution") }}
						</n-button>
					</template>
					{{ t("pipeline.forceStopConfirm") }}
				</n-popconfirm>
			</div>
			<n-button type="primary" ghost @click="emit('update:show', false)">
				{{ t("commons.button.close") }}
			</n-button>
		</div>
	</n-modal>
</template>

<style scoped>
/* Custom scrollbar for terminal */
::-webkit-scrollbar {
	width: 8px;
	height: 8px;
}
::-webkit-scrollbar-track {
	background: #1e1e1e;
	border-radius: 4px;
}
::-webkit-scrollbar-thumb {
	background: #424242;
	border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
	background: #555555;
}
</style>
