<template>
	<div class="space-y-4">
		<n-alert v-if="error" type="error" :show-icon="true">{{ error }}</n-alert>
		<div v-if="loading" class="flex min-h-48 items-center justify-center"><n-spin /></div>
		<template v-else>
			<n-empty v-if="probes.length === 0" :description="t('websiteDiagnostic.probeEmpty')" class="py-8" />
			<n-card v-for="(probe, index) in probes" :key="probe.id || index" size="small" class="mb-3">
				<div class="grid gap-3 sm:grid-cols-2">
					<n-form-item :label="t('websiteDiagnostic.probeName')"><n-input v-model:value="probe.name" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probePath')"><n-input v-model:value="probe.path" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeMethod')"><n-select v-model:value="probe.method" :options="methodOptions" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeStatus')"><n-input-number v-model:value="probe.expectedStatus" :min="100" :max="599" class="w-full" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeCode')"><n-input v-model:value="probe.expectedCode" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeFields')"><n-input v-model:value="probe.requiredFields" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeTimeout')"><n-input-number v-model:value="probe.timeoutMs" :min="100" :max="30000" class="w-full" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeInterval')"><n-input-number v-model:value="probe.intervalSeconds" :min="10" :max="86400" class="w-full" /></n-form-item>
					<n-form-item :label="t('websiteDiagnostic.probeFailures')"><n-input-number v-model:value="probe.failureThreshold" :min="1" :max="100" class="w-full" /></n-form-item>
				</div>
				<div class="flex items-center justify-between gap-2"><n-switch v-model:value="probe.enabled" /><div class="flex gap-2"><n-button v-if="probe.id" :loading="runningId === probe.id" @click="run(probe)">{{ t("websiteDiagnostic.probeRun") }}</n-button><n-button type="error" @click="probes.splice(index, 1)">{{ t("websiteDiagnostic.probeDelete") }}</n-button></div></div>
				<div v-if="probe.lastRunAt" class="mt-2 text-xs text-slate-500">{{ probe.lastStatus }} · {{ probe.lastMessage || formatTime(probe.lastRunAt) }}</div>
			</n-card>
			<div class="flex justify-between"><n-button @click="add">{{ t("websiteDiagnostic.probeAdd") }}</n-button><n-button type="primary" :loading="saving" @click="save">{{ t("websiteDiagnostic.save") }}</n-button></div>
		</template>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { WebsiteProbe } from "@/api/interface/websiteDiagnostic"
import { listWebsiteDiagnosticProbesAPI, runWebsiteDiagnosticProbeAPI, saveWebsiteDiagnosticProbesAPI } from "@/api/modules/website"
import { formatTime } from "@/utils/date"
import { websiteDiagnosticMessages } from "../websiteDiagnosticMessages"

const props = defineProps<{ websiteId: number }>()
const { t } = useI18n({ messages: websiteDiagnosticMessages })
const message = useMessage()
const loading = ref(false), saving = ref(false), runningId = ref(0), error = ref("")
const probes = ref<WebsiteProbe[]>([])
const methodOptions = [{ label: "GET", value: "GET" }, { label: "HEAD", value: "HEAD" }]
const errorText = (value: unknown, fallback: string) => value instanceof Error && value.message ? value.message : fallback
function emptyProbe(): WebsiteProbe { return { id: 0, websiteId: props.websiteId, name: "", enabled: true, method: "GET", path: "/health", expectedStatus: 200, expectedCode: "", requiredFields: "", timeoutMs: 5000, intervalSeconds: 60, failureThreshold: 3, failureCount: 0, lastStatus: "", lastMessage: "" } }
function add() { probes.value.push(emptyProbe()) }
async function load() { loading.value = true; error.value = ""; try { probes.value = (await listWebsiteDiagnosticProbesAPI(props.websiteId)).data || [] } catch (value) { error.value = errorText(value, t("websiteDiagnostic.probeLoadFailed")); message.error(error.value) } finally { loading.value = false } }
async function save() { saving.value = true; try { probes.value = (await saveWebsiteDiagnosticProbesAPI(props.websiteId, probes.value)).data || []; message.success(t("websiteDiagnostic.probeSaved")) } catch (value) { message.error(errorText(value, t("websiteDiagnostic.probeSaveFailed"))) } finally { saving.value = false } }
async function run(probe: WebsiteProbe) { runningId.value = probe.id; try { Object.assign(probe, (await runWebsiteDiagnosticProbeAPI(props.websiteId, probe.id)).data); message.success(t("websiteDiagnostic.probeFinished")) } catch (value) { message.error(errorText(value, t("websiteDiagnostic.probeRunFailed"))) } finally { runningId.value = 0 } }
onMounted(load)
</script>
