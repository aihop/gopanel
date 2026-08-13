<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { Flow } from "@/api/interface/flow"
import { createFlowRun, getFlowCodeBaselineSource, getFlowCodeDeliverySources } from "@/api/modules/flow"
import { flowMessages } from "./flowMessages"

const props = defineProps<{ show: boolean; flow: Flow.Item | null }>()
const emit = defineEmits<{ "update:show": [value: boolean]; success: [run: Flow.Run] }>()
const { t } = useI18n({ messages: flowMessages })
const message = useMessage()
const saving = ref(false)
const loadingSources = ref(false)
const sourceLoadError = ref(false)
const sources = ref<Flow.CodeDeliverySource[]>([])
const baseline = ref<Flow.CodeBaselineSource | null>(null)
const form = reactive({ codeDeliveryJobId: null as number | null, sourceCommit: "", version: "" })
const commitPattern = /^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$/
const isCodeFlow = computed(() => props.flow?.pipelineSourceType === "code")
const selectedSource = computed(() => sources.value.find(item => item.jobId === form.codeDeliveryJobId) || null)
const useProjectBaseline = computed(() => isCodeFlow.value && !sources.value.length && baseline.value?.available === true)
const selectedRepositories = computed(() => selectedSource.value?.repositories || baseline.value?.repositories || [])
const selectedDigest = computed(() => selectedSource.value?.sourceDigest || baseline.value?.sourceDigest || "")
const sourceOptions = computed(() => sources.value.map(item => ({
	label: t("flow.codeDeliveryOption", {
		task: item.taskTitle || `#${item.taskId}`,
		count: item.repositories.length,
		time: flowTime(item.completedAt)
	}),
	value: item.jobId
})))

function close() {
	emit("update:show", false)
}

async function submit() {
	if (!props.flow) return
	const sourceCommit = form.sourceCommit.trim()
	if (isCodeFlow.value && !form.codeDeliveryJobId && !useProjectBaseline.value) {
		message.warning(t("flow.codeDeliveryRequired"))
		return
	}
	if (!isCodeFlow.value && !commitPattern.test(sourceCommit)) {
		message.warning(t("flow.commitInvalid"))
		return
	}
	saving.value = true
	try {
		const response = await createFlowRun({
			flowId: props.flow.id,
			codeDeliveryJobId: isCodeFlow.value ? form.codeDeliveryJobId || undefined : undefined,
			useProjectBaseline: useProjectBaseline.value || undefined,
			sourceCommit: isCodeFlow.value ? undefined : sourceCommit,
			version: form.version.trim() || undefined
		})
		message.success(t("flow.runCreated", { id: response.data.id, version: response.data.version }))
		close()
		emit("success", response.data)
	} catch {
		message.error(t("flow.runCreateFailed"))
	} finally {
		saving.value = false
	}
}

function flowTime(value?: string) {
	if (!value) return "-"
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

async function loadSources() {
	if (!props.flow || props.flow.pipelineSourceType !== "code") return
	loadingSources.value = true
	sourceLoadError.value = false
	try {
		const deliveriesResponse = await getFlowCodeDeliverySources(props.flow.id)
		sources.value = deliveriesResponse.data || []
		form.codeDeliveryJobId = sources.value[0]?.jobId || null
		if (!sources.value.length) {
			const baselineResponse = await getFlowCodeBaselineSource(props.flow.id)
			baseline.value = baselineResponse.data || null
		}
	} catch {
		sourceLoadError.value = true
		message.error(t("flow.codeDeliveryLoadFailed"))
	} finally {
		loadingSources.value = false
	}
}

watch(() => props.show, visible => {
	if (!visible) return
	Object.assign(form, { codeDeliveryJobId: null, sourceCommit: "", version: "" })
	sources.value = []
	baseline.value = null
	sourceLoadError.value = false
	void loadSources()
})
</script>

<template>
	<n-modal :show="show" preset="card" style="width: min(560px, calc(100vw - 32px))" :title="t('flow.startRunTitle')" @update:show="emit('update:show', $event)">
		<div class="space-y-5">
			<n-alert type="info" :title="flow?.name">{{ t("flow.startRunDescription") }}</n-alert>
			<n-form-item v-if="isCodeFlow && (loadingSources || sources.length)" :label="t('flow.codeDeliverySource')" required>
				<n-select v-model:value="form.codeDeliveryJobId" :options="sourceOptions" :loading="loadingSources" :placeholder="t('flow.codeDeliveryPlaceholder')" />
				<template #feedback>{{ t("flow.codeDeliveryHelper") }}</template>
			</n-form-item>
			<n-alert v-if="isCodeFlow && sourceLoadError" type="error" :title="t('flow.codeDeliveryLoadFailed')">
				<n-button class="mt-3" size="small" @click="loadSources">{{ t("flow.retry") }}</n-button>
			</n-alert>
			<n-empty v-else-if="isCodeFlow && !loadingSources && !sources.length && !baseline?.available" :description="t('flow.codeDeliveryEmpty')" />
			<div v-if="selectedSource || useProjectBaseline" class="rounded-2xl border border-slate-200 p-4">
				<div class="text-sm font-medium fg-base-100">{{ selectedSource ? (selectedSource.taskTitle || `#${selectedSource.taskId}`) : t("flow.projectBaseline") }}</div>
				<div class="mt-1 break-all text-xs text-slate-500">{{ selectedDigest }}</div>
				<n-alert v-if="useProjectBaseline" class="mt-3" type="info" :title="t('flow.projectBaselineTitle')">{{ t("flow.projectBaselineHelper") }}</n-alert>
				<n-alert v-if="useProjectBaseline && baseline?.hasUncommittedChanges" class="mt-3" type="warning" :title="t('flow.baselineDirtyTitle')">{{ t("flow.baselineDirtyDescription") }}</n-alert>
				<div class="mt-3 space-y-2">
					<div v-for="repository in selectedRepositories" :key="repository.workspacePath" class="flex items-center justify-between gap-3 text-xs">
						<span class="truncate">{{ repository.name }} · {{ repository.targetBranch }}</span>
						<span class="shrink-0 font-mono text-slate-500">{{ repository.commit.slice(0, 12) }}</span>
					</div>
				</div>
			</div>
			<n-form-item v-if="!isCodeFlow" :label="t('flow.sourceCommit')" required>
				<n-input v-model:value="form.sourceCommit" :placeholder="t('flow.sourceCommitPlaceholder')" />
			</n-form-item>
			<n-form-item :label="t('flow.runVersion')">
				<n-input v-model:value="form.version" :placeholder="t('flow.runVersionPlaceholder')" />
				<template #feedback>{{ t("flow.runVersionHelper") }}</template>
			</n-form-item>
		</div>
		<template #footer>
			<div class="flex justify-end gap-2">
				<n-button @click="close">{{ t("flow.cancel") }}</n-button>
				<n-button type="primary" :loading="saving" :disabled="isCodeFlow && ((!form.codeDeliveryJobId && !useProjectBaseline) || sourceLoadError)" @click="submit">{{ t("flow.startRunConfirm") }}</n-button>
			</div>
		</template>
	</n-modal>
</template>
