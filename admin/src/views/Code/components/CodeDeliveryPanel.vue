<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { getCodeQualityChecks, getCodeSessionState, runCodeQualityCheck } from "@/api/modules/code"
import type {
	CodePreview,
	CodeQualityCheck,
	CodeQualityCheckResult,
	CodeQualityStatus,
	CodeSessionState
} from "@/api/interface/code"
import { codeDeliveryMessages } from "../codeDeliveryMessages"

const props = defineProps<{ sessionId: number | null }>()
const emit = defineEmits<{ (event: "open-file", file: { path: string; extension: string }): void }>()
const { t } = useI18n({ messages: codeDeliveryMessages })
const message = useMessage()
const dialog = useDialog()
const show = ref(false)
const loading = ref(false)
const stateError = ref(false)
const qualityError = ref(false)
const state = ref<CodeSessionState | null>(null)
const checks = ref<CodeQualityCheck[]>([])
const runningCheckId = ref("")
const showOutput = ref(false)
const selectedResult = ref<CodeQualityCheckResult | null>(null)
const selectedCheck = ref<CodeQualityCheck | null>(null)
let statePending = false

const buttonType = computed(() => {
	if (state.value?.errorSummary || checks.value.some(check => check.lastResult && check.lastResult.status !== "passed")) return "error"
	if (state.value?.pendingApproval) return "warning"
	if (state.value?.currentStage === "completed" || state.value?.currentStage === "preview_ready") return "success"
	return "default"
})

const stageLabel = computed(() => {
	const knownStages = [
		"idle", "task_ready", "instruction_queued", "awaiting_approval", "executing", "interactive",
		"completed", "preview_ready", "failed", "cancelled", "approval_rejected", "quality_check"
	]
	const stage = state.value?.currentStage || "idle"
	return knownStages.includes(stage) ? t(`code.deliveryStage_${stage}`) : t("code.deliveryStage_unknown")
})

const stageType = computed(() => {
	const stage = state.value?.currentStage || "idle"
	if (stage === "failed") return "error"
	if (["awaiting_approval", "approval_rejected", "cancelled"].includes(stage)) return "warning"
	if (["completed", "preview_ready"].includes(stage)) return "success"
	if (["executing", "instruction_queued", "interactive", "quality_check"].includes(stage)) return "info"
	return "default"
})

const loadState = async (notify = false) => {
	if (!props.sessionId || statePending) return
	statePending = true
	try {
		const response = await getCodeSessionState(props.sessionId)
		state.value = response.data
		stateError.value = false
	} catch (error) {
		stateError.value = true
		if (notify) message.error(error instanceof Error ? error.message : t("code.deliveryLoadFailed"))
	} finally {
		statePending = false
	}
}

const loadChecks = async (notify = false) => {
	if (!props.sessionId) return
	try {
		const response = await getCodeQualityChecks(props.sessionId)
		checks.value = response.data.items || []
		qualityError.value = false
	} catch (error) {
		qualityError.value = true
		if (notify) message.error(error instanceof Error ? error.message : t("code.qualityLoadFailed"))
	}
}

const loadDelivery = async (notify = false) => {
	if (!props.sessionId) return
	loading.value = true
	await Promise.all([loadState(notify), loadChecks(notify)])
	loading.value = false
}

const confirmRun = (check: CodeQualityCheck) => {
	if (!props.sessionId || runningCheckId.value) {
		if (runningCheckId.value) message.warning(t("code.qualityRunning"))
		return
	}
	dialog.warning({
		title: t("code.runQualityTitle"),
		content: t("code.runQualityConfirm", { command: check.command, workDir: check.workDir }),
		positiveText: t("code.confirm"),
		negativeText: t("code.cancel"),
		onPositiveClick: () => runCheck(check)
	})
}

const runCheck = async (check: CodeQualityCheck) => {
	if (!props.sessionId) return
	runningCheckId.value = check.id
	try {
		const response = await runCodeQualityCheck(props.sessionId, check.id)
		check.lastResult = response.data
		if (response.data.status === "passed") message.success(t("code.qualityPassed"))
		else message.error(t("code.qualityFailed"))
		await Promise.all([loadState(), loadChecks()])
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.qualityRunFailed"))
	} finally {
		runningCheckId.value = ""
	}
}

const openResult = (check: CodeQualityCheck) => {
	if (!check.lastResult) return
	selectedCheck.value = check
	selectedResult.value = check.lastResult
	showOutput.value = true
}

const openChangedFile = (path: string) => {
	const filename = path.split(/[\\/]/).pop() || ""
	emit("open-file", { path, extension: filename.includes(".") ? filename.split(".").pop() || "" : "" })
	show.value = false
}

const previewCanOpen = (preview: CodePreview) => preview.status === "ready"
const qualityType = (status?: CodeQualityStatus) => {
	if (status === "passed") return "success"
	if (status === "failed" || status === "timed_out") return "error"
	return "default"
}
const durationLabel = (milliseconds: number) => milliseconds < 1000
	? `${milliseconds} ms`
	: `${(milliseconds / 1000).toFixed(milliseconds < 10000 ? 1 : 0)} s`

watch(() => props.sessionId, () => {
	state.value = null
	checks.value = []
	stateError.value = false
	qualityError.value = false
	if (show.value) void loadDelivery()
})
watch(show, value => {
	if (value) void loadDelivery()
})
useIntervalFn(() => {
	if (show.value && !runningCheckId.value) void loadState()
}, 4000)
</script>

<template>
	<n-button size="small" :type="buttonType" secondary @click="show = true">
		<template #icon><Icon name="mdi:clipboard-check-outline" :size="17" /></template>
		{{ t("code.deliveryPanel") }}
	</n-button>
	<n-drawer v-model:show="show" placement="right" style="width: min(640px, 100vw)">
		<n-drawer-content :title="t('code.deliveryPanel')" closable body-content-style="padding: 16px;">
			<div class="mb-3 flex justify-end"><n-button quaternary size="small" :loading="loading" @click="loadDelivery(true)"><template #icon><Icon name="mdi:refresh" :size="18" /></template>{{ t("code.refreshDelivery") }}</n-button></div>
			<n-alert v-if="stateError" type="error" :title="t('code.deliveryLoadFailed')" class="mb-4">
				<n-button size="small" @click="loadState(true)">{{ t("code.retry") }}</n-button>
			</n-alert>
			<n-spin :show="loading && !state">
				<div class="space-y-4">
					<section class="rounded-2xl border border-slate-200 bg-white p-4">
						<div class="flex items-center justify-between gap-3">
							<strong class="text-sm text-slate-800">{{ t("code.currentStage") }}</strong>
							<n-tag :type="stageType" :bordered="false">{{ stageLabel }}</n-tag>
						</div>
						<div class="mt-4 text-sm font-semibold text-slate-800">{{ t("code.resultSummary") }}</div>
						<p class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-slate-600">
							{{ state?.recentOutput || t("code.noResultSummary") }}
						</p>
						<n-alert v-if="state?.errorSummary" type="error" class="mt-4" :title="t('code.errorSummary')">
							{{ state.errorSummary }}
						</n-alert>
					</section>

					<section class="rounded-2xl border border-slate-200 bg-white p-4">
						<strong class="text-sm text-slate-800">{{ t("code.changedFiles") }}</strong>
						<n-empty v-if="!state?.changedFiles.length" size="small" :description="t('code.noChangedFiles')" class="py-6" />
						<div v-else class="mt-3 space-y-1.5">
							<button v-for="file in state.changedFiles" :key="file" type="button" class="flex w-full items-center gap-2 rounded-xl bg-slate-50 px-3 py-2 text-left text-sm text-slate-600 hover:bg-blue-50 hover:text-blue-700" @click="openChangedFile(file)">
								<Icon name="mdi:file-code-outline" :size="17" /><span class="min-w-0 flex-1 truncate">{{ file }}</span><Icon name="mdi:open-in-new" :size="15" />
							</button>
						</div>
					</section>

					<section v-if="state?.previews.length" class="rounded-2xl border border-slate-200 bg-white p-4">
						<strong class="text-sm text-slate-800">{{ t("code.previews") }}</strong>
						<div class="mt-3 space-y-2">
							<a v-for="preview in state.previews" :key="preview.id" :href="previewCanOpen(preview) ? preview.url : undefined" :target="previewCanOpen(preview) ? '_blank' : undefined" rel="noopener noreferrer" class="flex items-center gap-3 rounded-xl bg-slate-50 p-3" :class="previewCanOpen(preview) ? 'hover:bg-blue-50' : 'cursor-not-allowed opacity-60'">
								<Icon :name="previewCanOpen(preview) ? 'mdi:open-in-new' : 'mdi:link-off'" :size="18" /><span class="min-w-0 flex-1 truncate text-sm">{{ preview.title }}</span><n-tag size="small" :bordered="false">{{ preview.status }}</n-tag>
							</a>
						</div>
					</section>

					<section class="rounded-2xl border border-slate-200 bg-white p-4">
						<div class="flex items-start justify-between gap-3"><div><strong class="text-sm text-slate-800">{{ t("code.qualityGate") }}</strong><p class="mt-1 text-xs leading-5 text-slate-500">{{ t("code.qualityGateHint") }}</p></div></div>
						<n-alert v-if="qualityError" type="error" :title="t('code.qualityLoadFailed')" class="mt-3"><n-button size="small" @click="loadChecks(true)">{{ t("code.retry") }}</n-button></n-alert>
						<n-empty v-else-if="checks.length === 0" size="small" :description="t('code.noQualityChecks')" class="py-6" />
						<div v-else class="mt-3 space-y-2">
							<div v-for="check in checks" :key="check.id" class="rounded-xl border border-slate-200 p-3">
								<div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><span class="text-sm font-semibold text-slate-800">{{ t(`code.qualityKind_${check.kind}`) }}</span><n-tag size="small" :type="qualityType(check.lastResult?.status)" :bordered="false">{{ check.lastResult ? t(`code.qualityStatus_${check.lastResult.status}`) : t("code.qualityNeverRun") }}</n-tag></div><div class="mt-1 truncate font-mono text-xs text-slate-500" :title="`${check.workDir}: ${check.command}`">{{ check.workDir }} · {{ check.command }}</div></div><n-button size="small" type="primary" secondary :loading="runningCheckId === check.id" :disabled="Boolean(runningCheckId && runningCheckId !== check.id)" @click="confirmRun(check)">{{ t("code.runQualityCheck") }}</n-button></div>
								<div v-if="check.lastResult" class="mt-2 flex items-center gap-3 text-xs text-slate-400"><span>{{ t("code.duration", { duration: durationLabel(check.lastResult.durationMs) }) }}</span><span>{{ t("code.exitCode", { code: check.lastResult.exitCode }) }}</span><n-button v-if="check.lastResult.output" text type="primary" size="tiny" @click="openResult(check)">{{ t("code.viewOutput") }}</n-button></div>
							</div>
						</div>
					</section>

					<section class="rounded-2xl border border-slate-200 bg-white p-4">
						<strong class="text-sm text-slate-800">{{ t("code.timeline") }}</strong>
						<n-empty v-if="!state?.timelineEvents.length" size="small" :description="t('code.noTimeline')" class="py-6" />
						<div v-else class="mt-4 space-y-4"><div v-for="event in state.timelineEvents" :key="event.id" class="border-l-2 border-blue-200 pl-3"><div class="flex items-center justify-between gap-3"><span class="text-sm font-medium text-slate-800">{{ event.title }}</span><span class="text-xs text-slate-400">{{ new Date(event.createdAt).toLocaleString() }}</span></div><div v-if="event.content" class="mt-1 whitespace-pre-wrap break-words text-xs leading-5 text-slate-500">{{ event.content }}</div></div></div>
					</section>
				</div>
			</n-spin>
		</n-drawer-content>
	</n-drawer>
	<n-modal v-model:show="showOutput" preset="card" style="width: min(900px, 94vw)" :title="`${t('code.qualityOutput')} · ${selectedCheck?.command || ''}`">
		<n-alert v-if="selectedResult?.outputTruncated" type="warning" :show-icon="false" class="mb-3">{{ t("code.outputTruncated") }}</n-alert>
		<n-empty v-if="!selectedResult?.output" :description="t('code.noQualityOutput')" />
		<pre v-else class="max-h-[65vh] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-slate-950 p-4 font-mono text-xs leading-5 text-slate-100">{{ selectedResult.output }}</pre>
	</n-modal>
</template>
