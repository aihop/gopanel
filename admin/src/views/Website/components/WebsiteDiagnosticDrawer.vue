<template>
	<n-drawer v-model:show="visible" placement="right" style="width: min(720px, 100vw)" :mask-closable="false">
		<n-drawer-content closable :native-scrollbar="false">
			<template #header>
				<div class="flex items-center gap-3">
					<span class="text-base font-semibold">{{ t("websiteDiagnostic.title") }}</span>
					<n-tag v-if="website.primaryDomain" round :bordered="false" type="primary">
						{{ website.primaryDomain }}
					</n-tag>
				</div>
			</template>

			<div v-if="loading" class="flex min-h-64 flex-col items-center justify-center gap-3 text-slate-500">
				<n-spin size="large" />
				<span>{{ t("websiteDiagnostic.loading") }}</span>
			</div>

			<n-result v-else-if="loadError" status="error" :title="t('websiteDiagnostic.loadFailed')" :description="loadError">
				<template #footer>
					<n-button type="primary" @click="load">{{ t("websiteDiagnostic.retry") }}</n-button>
				</template>
			</n-result>

			<div v-else class="space-y-6">
				<n-alert type="info" :show-icon="true">{{ t("websiteDiagnostic.description") }}</n-alert>
				<n-tabs v-model:value="activeTab" type="segment">
					<n-tab name="issues">{{ t("websiteDiagnostic.issuesTab") }}</n-tab>
					<n-tab name="settings">{{ t("websiteDiagnostic.settingsTab") }}</n-tab>
					<n-tab name="probes">{{ t("websiteDiagnostic.probesTab") }}</n-tab>
				</n-tabs>

				<website-diagnostic-issues
					v-if="activeTab === 'issues' && website.id"
					:website-id="website.id"
					:code-project-id="form.codeProjectId"
					@changed="emit('confirm')"
				/>

				<website-diagnostic-probes
					v-else-if="activeTab === 'probes' && website.id"
					:website-id="website.id"
				/>

				<template v-else>

				<section class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
					<div class="flex items-start justify-between gap-6">
						<div>
							<div class="font-semibold text-slate-700">{{ t("websiteDiagnostic.enabled") }}</div>
							<div class="mt-1 text-xs leading-5 text-slate-500">{{ t("websiteDiagnostic.enabledHint") }}</div>
						</div>
						<n-switch v-model:value="form.enabled" />
					</div>
				</section>

				<setting-section :title="t('websiteDiagnostic.sources')">
					<div class="grid gap-3 sm:grid-cols-2">
						<switch-card
							v-for="item in sourceOptions"
							:key="item.key"
							:label="item.label"
							:model-value="form[item.key]"
							@update:model-value="form[item.key] = $event"
						/>
					</div>
				</setting-section>

				<setting-section :title="t('websiteDiagnostic.contents')">
					<div class="grid gap-3 sm:grid-cols-2">
						<switch-card
							v-for="item in contentOptions"
							:key="item.key"
							:label="item.label"
							:model-value="form[item.key]"
							@update:model-value="form[item.key] = $event"
						/>
					</div>
				</setting-section>

				<setting-section :title="t('websiteDiagnostic.thresholds')">
					<div class="grid gap-x-4 sm:grid-cols-2">
						<n-form-item :label="t('websiteDiagnostic.slowRequestThresholdMs')">
							<n-input-number v-model:value="form.slowRequestThresholdMs" :min="100" :max="120000" class="w-full" />
						</n-form-item>
						<n-form-item :label="t('websiteDiagnostic.triggerCount')">
							<n-input-number v-model:value="form.triggerCount" :min="1" :max="10000" class="w-full" />
						</n-form-item>
						<n-form-item :label="t('websiteDiagnostic.triggerWindowMinutes')">
							<n-input-number v-model:value="form.triggerWindowMinutes" :min="1" :max="1440" class="w-full" />
						</n-form-item>
						<n-form-item :label="t('websiteDiagnostic.retentionDays')">
							<n-input-number v-model:value="form.retentionDays" :min="1" :max="365" class="w-full" />
						</n-form-item>
					</div>
				</setting-section>

				<setting-section :title="t('websiteDiagnostic.codeLink')">
					<n-alert v-if="auxiliaryError" class="mb-4" type="warning" :show-icon="true">{{ auxiliaryError }}</n-alert>
					<n-form-item :label="t('websiteDiagnostic.project')">
						<n-select
							v-model:value="form.codeProjectId"
							:options="projectOptions"
							:placeholder="t('websiteDiagnostic.projectPlaceholder')"
							:loading="auxiliaryLoading"
							clearable
						/>
					</n-form-item>
					<n-empty v-if="!auxiliaryLoading && projectOptions.length === 0" size="small" :description="t('websiteDiagnostic.projectEmpty')" class="py-3" />
					<div class="grid gap-x-4 sm:grid-cols-2">
						<n-form-item :label="t('websiteDiagnostic.executor')">
							<n-select v-model:value="form.defaultExecutorId" :options="executorOptions" :loading="auxiliaryLoading" />
						</n-form-item>
						<n-form-item :label="t('websiteDiagnostic.approvalPolicy')">
							<n-select v-model:value="form.approvalPolicy" :options="approvalOptions" />
						</n-form-item>
					</div>
					<div class="flex items-start justify-between gap-6 rounded-xl bg-slate-50 p-4">
						<div>
							<div class="text-sm font-medium text-slate-700">{{ t("websiteDiagnostic.autoAnalysis") }}</div>
							<div class="mt-1 text-xs leading-5 text-slate-500">{{ t("websiteDiagnostic.autoAnalysisHint") }}</div>
						</div>
						<n-switch v-model:value="form.autoAnalysis" :disabled="!form.codeProjectId" />
					</div>
				</setting-section>

				<n-alert v-if="showTrackingDir" type="default" :show-icon="true">
					<div class="font-medium">{{ t("websiteDiagnostic.trackingDir") }}</div>
					<div class="mt-1 break-all font-mono text-xs">{{ form.trackingDir }}</div>
					<div class="mt-1 text-xs text-slate-500">{{ t("websiteDiagnostic.trackingDirHint") }}</div>
				</n-alert>
				<n-card v-if="form.backendHook || form.browserHook" size="small" :title="t('websiteDiagnostic.hookIntegration')">
					<div class="space-y-3 text-xs text-slate-600">
						<div>{{ t("websiteDiagnostic.hookIntegrationHint") }}</div>
						<div class="break-all font-mono">POST {{ form.remoteEndpoint }}</div>
						<div>{{ t("websiteDiagnostic.hookSignatureHint") }}</div>
						<n-alert v-if="generatedSecret" type="warning" :show-icon="true">
							<div>{{ t("websiteDiagnostic.hookSecretOnce") }}</div>
							<div class="mt-1 break-all font-mono">{{ generatedSecret }}</div>
						</n-alert>
						<n-button :loading="secretLoading" @click="rotateHookSecret">{{ form.hookSecretConfigured ? t("websiteDiagnostic.rotateHookSecret") : t("websiteDiagnostic.generateHookSecret") }}</n-button>
					</div>
				</n-card>
				</template>
			</div>

			<template v-if="activeTab === 'settings'" #footer>
				<div class="flex justify-end gap-3">
					<n-button @click="visible = false">{{ t("websiteDiagnostic.cancel") }}</n-button>
					<n-button type="primary" :loading="saving" :disabled="loading || !!loadError" @click="save">
						{{ t("websiteDiagnostic.save") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, ref, watch } from "vue"
import { NCard, NSwitch, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { Website } from "@/api/interface/website"
import type { WebsiteDiagnosticSetting } from "@/api/interface/websiteDiagnostic"
import type { AIProject, CodeExecutor } from "@/api/interface/code"
import { getAIProjects, getCodeExecutors } from "@/api/modules/code"
import { getWebsiteDiagnosticSettingAPI, rotateWebsiteDiagnosticHookSecretAPI, saveWebsiteDiagnosticSettingAPI } from "@/api/modules/website"
import { websiteDiagnosticMessages } from "../websiteDiagnosticMessages"
import WebsiteDiagnosticIssues from "./WebsiteDiagnosticIssues.vue"
import WebsiteDiagnosticProbes from "./WebsiteDiagnosticProbes.vue"

type BooleanSettingKey =
	| "caddyMonitoring" | "activeProbes" | "backendHook" | "browserHook"
	| "monitorHttp4xx" | "monitorHttp5xx" | "monitorUpstreamErrors" | "monitorSlowRequests"
	| "monitorBusinessErrors" | "monitorBrowserErrors" | "monitorResourceErrors"

const emit = defineEmits<{ (event: "confirm"): void }>()
const { t } = useI18n({ messages: websiteDiagnosticMessages })
const message = useMessage()
const visible = ref(false)
const loading = ref(false)
const saving = ref(false)
const auxiliaryLoading = ref(false)
const loadError = ref("")
const auxiliaryError = ref("")
const activeTab = ref("issues")
const secretLoading = ref(false)
const generatedSecret = ref("")
const website = reactive<Partial<Website.WebsiteDTO>>({})
const projects = ref<AIProject[]>([])
const executors = ref<CodeExecutor[]>([])
const form = reactive<WebsiteDiagnosticSetting>(emptySetting())

const SettingSection = defineComponent({
	props: { title: { type: String, required: true } },
	setup(props, { slots }) {
		return () => h(NCard, { bordered: true, contentStyle: "padding: 20px" }, {
			header: () => h("span", { class: "text-sm font-semibold" }, props.title),
			default: () => slots.default?.()
		})
	}
})

const SwitchCard = defineComponent({
	props: { label: { type: String, required: true }, modelValue: { type: Boolean, required: true } },
	emits: ["update:modelValue"],
	setup(props, { emit: emitValue }) {
		return () => h("div", { class: "flex items-center justify-between gap-4 rounded-xl border border-slate-100 bg-slate-50 px-4 py-3" }, [
			h("span", { class: "text-sm text-slate-700" }, props.label),
			h(NSwitch, { value: props.modelValue, "onUpdate:value": (value: boolean) => emitValue("update:modelValue", value) })
		])
	}
})

function emptySetting(): WebsiteDiagnosticSetting {
	return {
		websiteId: 0, codeProjectId: 0, enabled: false, caddyMonitoring: true, activeProbes: false,
		backendHook: false, browserHook: false, autoAnalysis: false, monitorHttp4xx: true,
		monitorHttp5xx: true, monitorUpstreamErrors: true, monitorSlowRequests: true,
		monitorBusinessErrors: true, monitorBrowserErrors: true, monitorResourceErrors: true,
		slowRequestThresholdMs: 1500, triggerCount: 5, triggerWindowMinutes: 10, retentionDays: 7,
		defaultExecutorId: "codex", approvalPolicy: "safe_auto", configured: false, trackingDir: "",
		hookSecretConfigured: false, remoteEndpoint: ""
	}
}

const sourceOptions = computed(() => ([
	{ key: "caddyMonitoring" as const, label: t("websiteDiagnostic.caddyMonitoring") },
	{ key: "activeProbes" as const, label: t("websiteDiagnostic.activeProbes") },
	{ key: "backendHook" as const, label: t("websiteDiagnostic.backendHook") },
	{ key: "browserHook" as const, label: t("websiteDiagnostic.browserHook") }
]))
const contentOptions = computed(() => ([
	{ key: "monitorHttp4xx" as const, label: t("websiteDiagnostic.monitorHttp4xx") },
	{ key: "monitorHttp5xx" as const, label: t("websiteDiagnostic.monitorHttp5xx") },
	{ key: "monitorUpstreamErrors" as const, label: t("websiteDiagnostic.monitorUpstreamErrors") },
	{ key: "monitorSlowRequests" as const, label: t("websiteDiagnostic.monitorSlowRequests") },
	{ key: "monitorBusinessErrors" as const, label: t("websiteDiagnostic.monitorBusinessErrors") },
	{ key: "monitorBrowserErrors" as const, label: t("websiteDiagnostic.monitorBrowserErrors") },
	{ key: "monitorResourceErrors" as const, label: t("websiteDiagnostic.monitorResourceErrors") }
]))
const projectOptions = computed(() => projects.value.map(project => ({ label: project.name, value: project.id })))
const selectedExecutor = computed(() => executors.value.find(executor => executor.id === form.defaultExecutorId))
const executorOptions = computed(() => executors.value
	.filter(executor => executor.id !== "terminal")
	.map(executor => ({ label: executor.name, value: executor.id, disabled: !executor.available })))
const approvalOptions = computed(() => {
	const labels = {
		manual: t("websiteDiagnostic.policyManual"),
		safe_auto: t("websiteDiagnostic.policySafeAuto"),
		full_auto: t("websiteDiagnostic.policyFullAuto")
	}
	const policies = selectedExecutor.value?.approvalPolicies || ["manual", "safe_auto", "full_auto"]
	return policies.map(value => ({ label: labels[value], value }))
})
const showTrackingDir = computed(() => (form.backendHook || form.browserHook) && !!form.trackingDir)

function errorText(error: unknown, fallback: string) {
	return error instanceof Error && error.message ? error.message : fallback
}

async function loadAuxiliary() {
	auxiliaryLoading.value = true
	auxiliaryError.value = ""
	const [projectResult, executorResult] = await Promise.allSettled([
		getAIProjects({ page: 1, limit: 100 }),
		getCodeExecutors()
	])
	projects.value = projectResult.status === "fulfilled" ? projectResult.value.data.items || [] : []
	executors.value = executorResult.status === "fulfilled" ? executorResult.value.data || [] : []
	const failures = [projectResult, executorResult].filter(result => result.status === "rejected") as PromiseRejectedResult[]
	if (failures.length) auxiliaryError.value = errorText(failures[0].reason, t("websiteDiagnostic.loadFailed"))
	auxiliaryLoading.value = false
}

async function load() {
	if (!website.id) return
	loading.value = true
	loadError.value = ""
	void loadAuxiliary()
	try {
		const response = await getWebsiteDiagnosticSettingAPI(website.id)
		Object.assign(form, emptySetting(), response.data)
	} catch (error) {
		loadError.value = errorText(error, t("websiteDiagnostic.loadFailed"))
	} finally {
		loading.value = false
	}
}

function open(row: Website.WebsiteDTO) {
	Object.assign(website, row)
	activeTab.value = "issues"
	generatedSecret.value = ""
	visible.value = true
	void load()
}

async function rotateHookSecret() {
	if (!website.id) return
	secretLoading.value = true
	try {
		generatedSecret.value = (await rotateWebsiteDiagnosticHookSecretAPI(website.id)).data.secret
		form.hookSecretConfigured = true
		message.success(t("websiteDiagnostic.hookSecretGenerated"))
	} catch (error) {
		message.error(errorText(error, t("websiteDiagnostic.hookSecretFailed")))
	} finally {
		secretLoading.value = false
	}
}

function validate() {
	const sources: BooleanSettingKey[] = ["caddyMonitoring", "activeProbes", "backendHook", "browserHook"]
	const contents: BooleanSettingKey[] = [
		"monitorHttp4xx", "monitorHttp5xx", "monitorUpstreamErrors", "monitorSlowRequests",
		"monitorBusinessErrors", "monitorBrowserErrors", "monitorResourceErrors"
	]
	if (form.enabled && !sources.some(key => form[key])) return t("websiteDiagnostic.selectSource")
	if (form.enabled && !contents.some(key => form[key])) return t("websiteDiagnostic.selectContent")
	if (form.autoAnalysis && !form.codeProjectId) return t("websiteDiagnostic.selectProject")
	return ""
}

async function save() {
	if (!website.id) return
	const validationError = validate()
	if (validationError) {
		message.warning(validationError)
		return
	}
	saving.value = true
	try {
		const payload = {
			codeProjectId: form.codeProjectId, enabled: form.enabled,
			caddyMonitoring: form.caddyMonitoring, activeProbes: form.activeProbes,
			backendHook: form.backendHook, browserHook: form.browserHook, autoAnalysis: form.autoAnalysis,
			monitorHttp4xx: form.monitorHttp4xx, monitorHttp5xx: form.monitorHttp5xx,
			monitorUpstreamErrors: form.monitorUpstreamErrors, monitorSlowRequests: form.monitorSlowRequests,
			monitorBusinessErrors: form.monitorBusinessErrors, monitorBrowserErrors: form.monitorBrowserErrors,
			monitorResourceErrors: form.monitorResourceErrors, slowRequestThresholdMs: form.slowRequestThresholdMs,
			triggerCount: form.triggerCount, triggerWindowMinutes: form.triggerWindowMinutes,
			retentionDays: form.retentionDays, defaultExecutorId: form.defaultExecutorId,
			approvalPolicy: form.approvalPolicy
		}
		const response = await saveWebsiteDiagnosticSettingAPI(website.id, payload)
		Object.assign(form, response.data)
		message.success(t("websiteDiagnostic.saved"))
		emit("confirm")
	} catch (error) {
		message.error(errorText(error, t("websiteDiagnostic.saveFailed")))
	} finally {
		saving.value = false
	}
}

watch(() => form.codeProjectId, value => {
	if (!value) form.autoAnalysis = false
})

watch(() => form.defaultExecutorId, () => {
	if (!approvalOptions.value.some(option => option.value === form.approvalPolicy)) {
		form.approvalPolicy = approvalOptions.value[0]?.value || "full_auto"
	}
})

defineExpose({ open })
</script>
