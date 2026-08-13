<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import type { Flow } from "@/api/interface/flow"
import { getFlowRun, getFlowRunPage } from "@/api/modules/flow"
import { flowMessages } from "./flowMessages"

const { t } = useI18n({ messages: flowMessages })
const message = useMessage()
const loading = ref(false)
const loadError = ref(false)
const runs = ref<Flow.Run[]>([])
const detail = ref<Flow.Run | null>(null)
const detailVisible = ref(false)
const detailLoading = ref(false)

function runType(status: Flow.RunStatus) {
	if (status === "failed") return "error"
	if (status === "waiting_deployment") return "success"
	if (status === "running") return "info"
	return "warning"
}

function stageType(status: Flow.StageRun["status"]) {
	if (status === "failed") return "error"
	if (status === "success") return "success"
	if (status === "running") return "info"
	return "warning"
}

function resourceLabel(type: string) {
	return type === "release" ? t("flow.release") : t("flow.pipelineRecord")
}

function flowTime(value?: string) {
	if (!value) return "-"
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function runSourceLabel(run: Flow.Run) {
	if (run.sourceType === "code_delivery") {
		return run.sourceTaskTitle || t("flow.codeDeliveryJob", { id: run.codeDeliveryJobId })
	}
	return run.sourceCommit.slice(0, 12)
}

async function loadRuns(silent = false) {
	if (!silent) loading.value = true
	loadError.value = false
	try {
		const response = await getFlowRunPage({ page: 1, limit: 20 })
		runs.value = response.data.items || []
	} catch {
		loadError.value = true
		if (!silent) message.error(t("flow.runLoadFailed"))
	} finally {
		if (!silent) loading.value = false
	}
}

async function openDetail(run: Flow.Run) {
	detailVisible.value = true
	detailLoading.value = true
	detail.value = run
	try {
		const response = await getFlowRun(run.id)
		detail.value = response.data
	} catch {
		message.error(t("flow.runDetailFailed"))
	} finally {
		detailLoading.value = false
	}
}

function refresh() {
	void loadRuns()
}

defineExpose({ refresh })
onMounted(() => loadRuns())
useIntervalFn(() => {
	if (runs.value.some(item => item.status === "queued" || item.status === "running")) void loadRuns(true)
}, 5000)
</script>

<template>
	<section class="rounded-3xl border border-slate-200 bg-base-100 p-6 shadow-sm">
		<div class="flex items-center justify-between gap-4">
			<div><h2 class="text-lg font-semibold fg-base-100">{{ t("flow.recentRuns") }}</h2><p class="mt-1 text-xs text-slate-500">{{ t("flow.recentRunsDescription") }}</p></div>
			<n-button quaternary circle :loading="loading" :aria-label="t('flow.refreshRuns')" @click="loadRuns()"><template #icon><Icon name="mdi:refresh" :size="18" /></template></n-button>
		</div>
		<n-alert v-if="loadError" class="mt-4" type="error" :title="t('flow.runLoadFailed')"><n-button class="mt-3" size="small" @click="loadRuns()">{{ t("flow.retry") }}</n-button></n-alert>
		<n-spin :show="loading">
			<n-empty v-if="!loading && !runs.length" class="py-10" :description="t('flow.runEmpty')" />
			<div v-else class="mt-5 space-y-3">
				<button v-for="run in runs" :key="run.id" type="button" class="flex w-full flex-col gap-3 rounded-2xl border border-slate-200 p-4 text-left transition hover:border-blue-400 sm:flex-row sm:items-center sm:justify-between" @click="openDetail(run)">
					<div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><span class="font-semibold fg-base-100">v{{ run.version }}</span><n-tag size="small" :type="runType(run.status)">{{ t(`flow.runStatus.${run.status}`) }}</n-tag><span class="text-xs text-slate-400">#{{ run.id }}</span></div><div class="mt-2 truncate text-xs text-slate-500">{{ run.flowName }} · {{ runSourceLabel(run) }} · {{ flowTime(run.createdAt) }}</div><div v-if="run.errorSummary" class="mt-2 line-clamp-2 text-xs text-red-500">{{ run.errorSummary }}</div></div>
					<div class="flex shrink-0 items-center gap-4 text-xs text-slate-500"><span>{{ t(`flow.runStage.${run.currentStage}`) }}</span><span v-if="run.pipelineRecordId">{{ t("flow.pipelineRecordId", { id: run.pipelineRecordId }) }}</span><span v-if="run.releaseId">{{ t("flow.releaseId", { id: run.releaseId }) }}</span><Icon name="mdi:chevron-right" :size="18" /></div>
				</button>
			</div>
		</n-spin>

		<n-drawer v-model:show="detailVisible" :width="620">
			<n-drawer-content :title="detail ? t('flow.runDetailTitle', { id: detail.id }) : t('flow.runDetail')" closable>
				<n-spin :show="detailLoading">
					<div v-if="detail" class="space-y-5">
						<n-descriptions bordered :column="1" size="small">
							<n-descriptions-item :label="t('flow.runVersion')">{{ detail.version }}</n-descriptions-item>
							<n-descriptions-item v-if="detail.sourceType === 'git'" :label="t('flow.sourceCommit')"><span class="break-all font-mono text-xs">{{ detail.sourceCommit }}</span></n-descriptions-item>
							<n-descriptions-item v-else :label="t('flow.codeDeliverySource')">{{ detail.sourceTaskTitle || t("flow.codeDeliveryJob", { id: detail.codeDeliveryJobId }) }}</n-descriptions-item>
							<n-descriptions-item v-if="detail.sourceDigest" :label="t('flow.sourceDigest')"><span class="break-all font-mono text-xs">{{ detail.sourceDigest }}</span></n-descriptions-item>
							<n-descriptions-item :label="t('flow.runStatusLabel')"><n-tag size="small" :type="runType(detail.status)">{{ t(`flow.runStatus.${detail.status}`) }}</n-tag></n-descriptions-item>
							<n-descriptions-item :label="t('flow.pipelineRecord')">{{ detail.pipelineRecordId || "-" }}</n-descriptions-item>
							<n-descriptions-item :label="t('flow.release')">{{ detail.releaseId || "-" }}</n-descriptions-item>
							<n-descriptions-item :label="t('flow.artifactDigest')"><span class="break-all font-mono text-xs">{{ detail.artifactDigest || "-" }}</span></n-descriptions-item>
						</n-descriptions>
						<div v-if="detail.sourceRepositories?.length">
							<h3 class="mb-3 font-semibold fg-base-100">{{ t("flow.sourceRepositories") }}</h3>
							<div class="space-y-2">
								<div v-for="repository in detail.sourceRepositories" :key="repository.workspacePath" class="rounded-xl border border-slate-200 p-3">
									<div class="flex items-center justify-between gap-3 text-sm"><span class="truncate font-medium fg-base-100">{{ repository.name }}</span><span class="shrink-0 text-xs text-slate-500">{{ repository.targetBranch }}</span></div>
									<div class="mt-2 break-all font-mono text-xs text-slate-500">{{ repository.commit }}</div>
								</div>
							</div>
						</div>
						<n-alert v-if="detail.errorSummary" type="error" :title="detail.failureCode">{{ detail.errorSummary }}</n-alert>
						<div><h3 class="mb-3 font-semibold fg-base-100">{{ t("flow.stageRecords") }}</h3><div class="space-y-3"><div v-for="stage in detail.stages || []" :key="stage.id" class="rounded-2xl border border-slate-200 p-4"><div class="flex items-center justify-between gap-3"><span class="font-medium fg-base-100">{{ t(`flow.runStage.${stage.stage}`) }}</span><n-tag size="small" :type="stageType(stage.status)">{{ t(`flow.stageStatus.${stage.status}`) }}</n-tag></div><div v-if="stage.errorDetail" class="mt-2 text-xs text-red-500">{{ stage.errorDetail }}</div><div v-if="stage.resourceId" class="mt-2 text-xs text-slate-400">{{ resourceLabel(stage.resourceType) }} #{{ stage.resourceId }}</div></div></div></div>
					</div>
				</n-spin>
			</n-drawer-content>
		</n-drawer>
	</section>
</template>
