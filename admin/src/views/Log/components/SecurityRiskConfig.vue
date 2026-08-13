<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { SecurityMonitoringConfig } from "@/api/interface/securityMonitoring"
import type { AIProviderAccount } from "@/api/interface/aiAccounts"
import { getAIProviderAccounts } from "@/api/modules/code"
import {
	getSecurityMonitoringConfig,
	saveSecurityMonitoringConfig
} from "@/api/modules/securityMonitoring"

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (event: "update:show", value: boolean): void }>()
const { t } = useI18n()
const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const error = ref("")
const accounts = ref<AIProviderAccount[]>([])
const form = ref<SecurityMonitoringConfig>({
	enabled: true,
	websiteEnabled: true,
	sshEnabled: true,
	panelEnabled: true,
	aiEnabled: false,
	aiProviderAccountId: 0,
	aiIntervalMinutes: 15,
	aiDailyTokenBudget: 50000,
	maxBatchBytes: 2097152,
	maxBatchLines: 10000,
	requestPerMinute: 120,
	notFoundPerMinute: 30,
	serverErrorPerMinute: 20,
	loginFailurePerMinute: 10,
	sshFailurePerMinute: 10,
	debounceTimes: 2,
	resolveAfterMinutes: 10
})

async function loadConfig() {
	loading.value = true
	error.value = ""
	try {
		const [configResponse, accountResponse] = await Promise.all([
			getSecurityMonitoringConfig(), getAIProviderAccounts()
		])
		form.value = { ...form.value, ...configResponse.data }
		accounts.value = (accountResponse.data || []).filter(account => account.enabled && account.useForSecurityAnalysis)
	} catch {
		error.value = t("securityMonitoring.configLoadFailed")
		message.error(error.value)
	} finally {
		loading.value = false
	}
}

const accountOptions = computed(() => accounts.value.map(account => ({
	label: `${account.name} · ${account.model}`,
	value: account.id
})))
const selectedAccountAvailable = computed(() => accounts.value.some(account => account.id === form.value.aiProviderAccountId))

async function saveConfig() {
	if (form.value.aiEnabled && !selectedAccountAvailable.value) {
		message.error(t("securityMonitoring.aiProviderRequired"))
		return
	}
	saving.value = true
	try {
		await saveSecurityMonitoringConfig(form.value)
		message.success(t("securityMonitoring.saved"))
		emit("update:show", false)
	} catch {
		message.error(t("securityMonitoring.saveFailed"))
	} finally {
		saving.value = false
	}
}

onMounted(() => void loadConfig())
</script>

<template>
	<n-modal
		:show="props.show"
		preset="card"
		:title="t('securityMonitoring.settings')"
		style="width: 720px"
		@update:show="value => emit('update:show', value)"
	>
		<n-spin :show="loading">
			<n-alert v-if="error" type="error" class="mb-4">{{ error }}</n-alert>
			<n-form label-placement="left" :label-width="180" size="small">
				<n-form-item :label="t('securityMonitoring.monitoringEnabled')">
					<n-switch v-model:value="form.enabled" />
				</n-form-item>
				<n-form-item :label="t('securityMonitoring.sources')">
					<n-space :wrap="true">
						<n-checkbox v-model:checked="form.websiteEnabled">{{ t("securityMonitoring.source.website") }}</n-checkbox>
						<n-checkbox v-model:checked="form.sshEnabled">{{ t("securityMonitoring.source.ssh") }}</n-checkbox>
						<n-checkbox v-model:checked="form.panelEnabled">{{ t("securityMonitoring.source.panel") }}</n-checkbox>
					</n-space>
				</n-form-item>
				<n-divider>{{ t("securityMonitoring.ruleThresholds") }}</n-divider>
				<div class="grid grid-cols-1 gap-x-5 md:grid-cols-2">
					<n-form-item :label="t('securityMonitoring.requestThreshold')">
						<n-input-number v-model:value="form.requestPerMinute" :min="1" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.notFoundThreshold')">
						<n-input-number v-model:value="form.notFoundPerMinute" :min="1" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.serverErrorThreshold')">
						<n-input-number v-model:value="form.serverErrorPerMinute" :min="1" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.panelFailureThreshold')">
						<n-input-number v-model:value="form.loginFailurePerMinute" :min="1" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.sshFailureThreshold')">
						<n-input-number v-model:value="form.sshFailurePerMinute" :min="1" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.debounceTimes')">
						<n-input-number v-model:value="form.debounceTimes" :min="1" :max="10" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.resolveMinutes')">
						<n-input-number v-model:value="form.resolveAfterMinutes" :min="1" :max="1440" class="w-full" />
					</n-form-item>
				</div>
				<n-divider>{{ t("securityMonitoring.aiInspection") }}</n-divider>
				<n-form-item :label="t('securityMonitoring.aiEnabled')">
					<n-switch v-model:value="form.aiEnabled" />
				</n-form-item>
				<n-form-item :label="t('securityMonitoring.aiProvider')">
					<n-select
						v-model:value="form.aiProviderAccountId"
						:options="accountOptions"
						:disabled="!form.aiEnabled"
						:placeholder="t('securityMonitoring.selectAiProvider')"
					/>
					<template #feedback>{{ t("securityMonitoring.aiProviderHint") }}</template>
				</n-form-item>
				<n-alert v-if="form.aiEnabled && !accounts.length" type="warning" class="mb-4">
					{{ t("securityMonitoring.noAiProviders") }}
				</n-alert>
				<n-alert type="info" class="mb-4">{{ t("securityMonitoring.aiBoundary") }}</n-alert>
				<div class="grid grid-cols-1 gap-x-5 md:grid-cols-2">
					<n-form-item :label="t('securityMonitoring.aiInterval')">
						<n-input-number v-model:value="form.aiIntervalMinutes" :min="5" :max="1440" class="w-full" />
					</n-form-item>
					<n-form-item :label="t('securityMonitoring.aiBudget')">
						<n-input-number v-model:value="form.aiDailyTokenBudget" :min="0" class="w-full" />
					</n-form-item>
				</div>
			</n-form>
		</n-spin>
		<template #footer>
			<div class="flex justify-end gap-2">
				<n-button @click="emit('update:show', false)">{{ t("securityMonitoring.cancel") }}</n-button>
				<n-button type="primary" :loading="saving" @click="saveConfig">{{ t("securityMonitoring.save") }}</n-button>
			</div>
		</template>
	</n-modal>
</template>
