<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { AIProject } from "@/api/interface/code"
import type { Pipeline } from "@/api/interface/pipeline"
import type { Website } from "@/api/interface/website"
import type { Flow } from "@/api/interface/flow"
import { getAIProjects } from "@/api/modules/code"
import { getPipelinePage } from "@/api/modules/pipeline"
import { websiteListAPI } from "@/api/modules/website"
import { createFlow } from "@/api/modules/flow"
import { flowMessages } from "./flowMessages"

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ "update:show": [value: boolean]; success: [] }>()
const { t } = useI18n({ messages: flowMessages })
const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const loadError = ref(false)
const projects = ref<AIProject[]>([])
const pipelines = ref<Pipeline.ResPipeline[]>([])
const websites = ref<Website.WebsiteDTO[]>([])
const form = reactive({
	name: "",
	projectId: null as number | null,
	pipelineId: null as number | null,
	autoStart: false,
	previewEnabled: true,
	previewWebsiteId: null as number | null,
	productionEnabled: false,
	productionWebsiteId: null as number | null
})

const projectOptions = computed(() => projects.value.map(item => ({ label: item.name, value: item.id })))
const pipelineOptions = computed(() => pipelines.value.map(item => ({ label: item.name, value: item.id })))
const websiteOptions = computed(() => websites.value.map(item => ({
	label: item.alias || item.primaryDomain,
	value: item.id
})))
const missingResources = computed(() => !projects.value.length || !pipelines.value.length || !websites.value.length)

function resetForm() {
	Object.assign(form, {
		name: "", projectId: null, pipelineId: null, autoStart: false,
		previewEnabled: true, previewWebsiteId: null,
		productionEnabled: false, productionWebsiteId: null
	})
}

async function loadOptions() {
	loading.value = true
	loadError.value = false
	try {
		const [projectResponse, pipelineResponse, websiteResponse] = await Promise.all([
			getAIProjects({ page: 1, limit: 100 }),
			getPipelinePage({ page: 1, limit: 100 }),
			websiteListAPI()
		])
		projects.value = projectResponse.data.items || []
		pipelines.value = pipelineResponse.data.items || []
		websites.value = websiteResponse.data.items || []
	} catch {
		loadError.value = true
		message.error(t("flow.optionsLoadFailed"))
	} finally {
		loading.value = false
	}
}

function close() {
	emit("update:show", false)
}

async function submit() {
	if (!form.name.trim() || !form.projectId || !form.pipelineId) {
		message.warning(t("flow.basicRequired"))
		return
	}
	if ((!form.previewEnabled || !form.previewWebsiteId) && (!form.productionEnabled || !form.productionWebsiteId)) {
		message.warning(t("flow.environmentRequired"))
		return
	}
	if (form.previewEnabled && !form.previewWebsiteId) {
		message.warning(t("flow.previewWebsiteRequired"))
		return
	}
	if (form.productionEnabled && !form.productionWebsiteId) {
		message.warning(t("flow.productionWebsiteRequired"))
		return
	}
	const environments: Flow.CreateEnvironment[] = []
	if (form.previewEnabled && form.previewWebsiteId) {
		environments.push({ name: "preview", websiteId: form.previewWebsiteId, autoDeploy: true, approvalRequired: false })
	}
	if (form.productionEnabled && form.productionWebsiteId) {
		environments.push({ name: "production", websiteId: form.productionWebsiteId, autoDeploy: false, approvalRequired: true })
	}
	saving.value = true
	try {
		await createFlow({
			name: form.name.trim(), projectId: form.projectId, pipelineId: form.pipelineId,
			autoStartAfterCodeDelivery: form.autoStart, environments
		})
		message.success(t("flow.createSuccess"))
		close()
		emit("success")
	} catch {
		message.error(t("flow.createFailed"))
	} finally {
		saving.value = false
	}
}

watch(() => props.show, value => {
	if (!value) return
	resetForm()
	void loadOptions()
})
</script>

<template>
	<n-modal :show="show" preset="card" style="width: min(760px, calc(100vw - 32px))" :title="t('flow.createTitle')" @update:show="emit('update:show', $event)">
		<n-spin :show="loading">
			<n-alert v-if="loadError" type="error" :title="t('flow.optionsLoadFailed')" class="mb-5">
				<n-button class="mt-3" size="small" @click="loadOptions">{{ t("flow.retry") }}</n-button>
			</n-alert>
			<div v-else class="space-y-6">
				<div class="rounded-2xl border border-slate-200 p-5">
					<div class="mb-4 flex items-center gap-3"><span class="flex size-7 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white">1</span><span class="font-semibold fg-base-100">{{ t("flow.basicTitle") }}</span></div>
					<div class="grid gap-4 md:grid-cols-2">
						<n-form-item :label="t('flow.name')" required><n-input v-model:value="form.name" :placeholder="t('flow.namePlaceholder')" /></n-form-item>
						<n-form-item :label="t('flow.project')" required><n-select v-model:value="form.projectId" filterable :options="projectOptions" :placeholder="t('flow.projectPlaceholder')" /></n-form-item>
						<n-form-item :label="t('flow.pipeline')" required><n-select v-model:value="form.pipelineId" filterable :options="pipelineOptions" :placeholder="t('flow.pipelinePlaceholder')" /></n-form-item>
						<n-form-item :label="t('flow.autoStart')"><n-switch v-model:value="form.autoStart" /><span class="ml-3 text-xs text-slate-500">{{ t("flow.autoStartHelper") }}</span></n-form-item>
					</div>
				</div>
				<div class="rounded-2xl border border-slate-200 p-5">
					<div class="mb-4 flex items-center gap-3"><span class="flex size-7 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white">2</span><span class="font-semibold fg-base-100">{{ t("flow.environmentTitle") }}</span></div>
					<div class="grid gap-4 md:grid-cols-2">
						<div class="rounded-xl bg-slate-50 p-4 dark:bg-white/5">
							<div class="flex items-center justify-between"><div><div class="font-medium fg-base-100">{{ t("flow.preview") }}</div><div class="mt-1 text-xs text-slate-500">{{ t("flow.previewHelper") }}</div></div><n-switch v-model:value="form.previewEnabled" /></div>
							<n-select v-if="form.previewEnabled" v-model:value="form.previewWebsiteId" class="mt-4" filterable :options="websiteOptions" :placeholder="t('flow.websitePlaceholder')" />
						</div>
						<div class="rounded-xl bg-slate-50 p-4 dark:bg-white/5">
							<div class="flex items-center justify-between"><div><div class="font-medium fg-base-100">{{ t("flow.production") }}</div><div class="mt-1 text-xs text-slate-500">{{ t("flow.productionHelper") }}</div></div><n-switch v-model:value="form.productionEnabled" /></div>
							<n-select v-if="form.productionEnabled" v-model:value="form.productionWebsiteId" class="mt-4" filterable :options="websiteOptions" :placeholder="t('flow.websitePlaceholder')" />
						</div>
					</div>
				</div>
				<n-alert v-if="missingResources" type="warning">{{ t("flow.missingResources") }}</n-alert>
			</div>
		</n-spin>
		<template #footer>
			<div class="flex justify-end gap-2"><n-button @click="close">{{ t("flow.cancel") }}</n-button><n-button type="primary" :loading="saving" :disabled="loading || loadError || missingResources" @click="submit">{{ t("flow.createConfirm") }}</n-button></div>
		</template>
	</n-modal>
</template>
