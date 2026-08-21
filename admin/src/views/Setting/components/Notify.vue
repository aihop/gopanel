<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue"
import {
	NAlert,
	NButton,
	NCard,
	NCheckbox,
	NDataTable,
	NForm,
	NFormItem,
	NInput,
	NInputNumber,
	NSelect,
	NSpace,
	NSwitch,
	NTag,
	useMessage,
	type DataTableColumns
} from "naive-ui"
import {
	evaluateAlerts,
	getNotifyConfig,
	getNotifyEvents,
	saveNotifyConfig,
	testNotifyMail
} from "@/api/modules/notify_alert"
import type { Notify } from "@/api/interface/notify"
import { formatTime } from "@/utils/date"
import { useI18n } from "vue-i18n"
import { notifyMessages } from "@/i18n/locales/notify"

const message = useMessage()
const { t } = useI18n({ messages: notifyMessages })
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const events = ref<Notify.Event[]>([])

// 密码单独存：后端只回"是否已设置"，明文永远不出后端。
// 留空提交 = 沿用已保存的密码，不然用户改个端口就把密码清没了。
const password = ref("")
const hasPassword = ref(false)

const form = ref<Notify.Config>({
	enabled: false,
	smtpHost: "",
	smtpPort: 587,
	smtpUser: "",
	smtpFrom: "",
	smtpTlsMode: "starttls",
	receivers: "",
	debounceTimes: 2,
	silenceHours: 6,
	notifyResolved: true,
	enableDisk: true,
	enableContainer: true,
	enableOffline: true,
	enableCert: false,
	enableCode: true,
	enableSecurity: true,
	enableSecurityLowMedium: false
})

const tlsOptions = [
	{ label: t("setting.tlsStarttls"), value: "starttls" },
	{ label: t("setting.tlsSsl"), value: "ssl" },
	{ label: t("setting.tlsNone"), value: "none" }
]

// 端口和加密方式对不上是配 SMTP 最常见的错误，表现是连接一直卡到超时
const portMismatch = computed(() => {
	if (form.value.smtpPort === 465 && form.value.smtpTlsMode !== "ssl") {
		return t("setting.portMismatch465")
	}
	if ((form.value.smtpPort === 587 || form.value.smtpPort === 25) && form.value.smtpTlsMode === "ssl") {
		return t("setting.portMismatch587")
	}
	return ""
})

const fetchConfig = async () => {
	loading.value = true
	try {
		const res = await getNotifyConfig()
		form.value = { ...form.value, ...res.data }
		hasPassword.value = !!res.data.hasPassword
	} catch (error: any) {
		void 0
	} finally {
		loading.value = false
	}
}

const fetchEvents = async () => {
	try {
		const res = await getNotifyEvents(1, 20)
		events.value = res.data.items || []
	} catch (_error) {
		// 事件列表拉不到不影响配置，静默即可
	}
}

const handleSave = async () => {
	saving.value = true
	try {
		const res = await saveNotifyConfig({ ...form.value, password: password.value })
		form.value = { ...form.value, ...res.data }
		hasPassword.value = !!res.data.hasPassword
		password.value = ""
		message.success(t("setting.saved"))
	} catch (error: any) {
		void 0
	} finally {
		saving.value = false
	}
}

const handleTest = async () => {
	testing.value = true
	try {
		await testNotifyMail({ ...form.value, password: password.value })
		message.success(t("setting.testMailSent"))
	} catch (error: any) {
		void 0
	} finally {
		testing.value = false
	}
}

const handleEvaluate = async () => {
	try {
		await evaluateAlerts()
		message.success(t("setting.evaluateDone"))
		void fetchEvents()
	} catch (error: any) {
		void 0
	}
}

const typeLabel: Record<string, string> = {
	disk: t("setting.notifyTypeDisk"),
	container: t("setting.notifyTypeContainer"),
	offline: t("setting.notifyTypeOffline"),
	unauthorized: t("setting.notifyTypeUnauthorized"),
	cert: t("setting.notifyTypeCert")
}

const statusMeta: Record<string, { label: string; type: "default" | "warning" | "error" | "success" }> = {
	pending: { label: t("setting.notifyStatusPending"), type: "default" },
	firing: { label: t("setting.notifyStatusFiring"), type: "error" },
	resolved: { label: t("setting.notifyStatusResolved"), type: "success" }
}

const eventColumns: DataTableColumns<Notify.Event> = [
	{ title: t("setting.notifyColTarget"), key: "sourceName", width: 160, ellipsis: { tooltip: true } },
	{
		title: t("setting.notifyColType"),
		key: "type",
		width: 100,
		render: row => h(NTag, { size: "small" }, { default: () => typeLabel[row.type] || row.type })
	},
	{
		title: t("setting.notifyColStatus"),
		key: "status",
		width: 100,
		render: row => {
			const meta = statusMeta[row.status] || statusMeta.pending
			return h(NTag, { size: "small", type: meta.type }, { default: () => meta.label })
		}
	},
	{ title: t("setting.notifyColDetail"), key: "detail", minWidth: 240, ellipsis: { tooltip: true } },
	{
		title: t("setting.notifyColUpdatedAt"),
		key: "updatedAt",
		width: 170,
		render: row => formatTime(row.updatedAt || "")
	}
]

onMounted(() => {
	void fetchConfig()
	void fetchEvents()
})
</script>

<template>
	<div class="pt-2">
		<n-card :title="t('setting.mailNotify')" class="mb-4">
			<template #header-extra>
				<n-switch v-model:value="form.enabled">
					<template #checked>{{ t("setting.enabled") }}</template>
					<template #unchecked>{{ t("setting.disabled") }}</template>
				</n-switch>
			</template>

			<n-alert type="info" :show-icon="false" class="mb-4 text-xs">
				{{ t("setting.notifyAlertDesc") }}
			</n-alert>

			<n-form label-placement="left" :label-width="110" size="small">
				<div class="grid grid-cols-1 gap-x-6 md:grid-cols-2">
					<n-form-item :label="t('setting.smtpServer')">
						<n-input v-model:value="form.smtpHost" placeholder="smtp.qq.com" />
					</n-form-item>
					<n-form-item :label="t('setting.port')">
						<n-input-number v-model:value="form.smtpPort" class="w-full" :min="1" :max="65535" />
					</n-form-item>
					<n-form-item :label="t('setting.encryptionMode')">
						<n-select v-model:value="form.smtpTlsMode" :options="tlsOptions" />
					</n-form-item>
					<n-form-item :label="t('setting.loginAccount')">
						<n-input v-model:value="form.smtpUser" placeholder="you@qq.com" />
					</n-form-item>
					<n-form-item :label="t('setting.passwordOrAuthCode')">
						<n-input
							v-model:value="password"
							type="password"
							show-password-on="click"
							:placeholder="
								hasPassword ? t('setting.passwordSavedHint') : t('setting.passwordAuthCodeHint')
							"
						/>
					</n-form-item>
					<n-form-item :label="t('setting.sender')">
						<n-input v-model:value="form.smtpFrom" :placeholder="t('setting.senderPlaceholder')" />
					</n-form-item>
				</div>

				<n-alert v-if="portMismatch" type="warning" :show-icon="true" class="mb-3 text-xs">
					{{ portMismatch }}
				</n-alert>

				<n-form-item :label="t('setting.recipients')">
					<n-input
						v-model:value="form.receivers"
						type="textarea"
						:rows="2"
						:placeholder="t('setting.recipientsPlaceholder')"
					/>
				</n-form-item>

				<div class="grid grid-cols-1 gap-x-6 md:grid-cols-2">
					<n-form-item :label="t('setting.consecutiveHits')">
						<n-input-number v-model:value="form.debounceTimes" class="w-full" :min="1" :max="10">
							<template #suffix>{{ t("setting.timesToAlert") }}</template>
						</n-input-number>
					</n-form-item>
					<n-form-item :label="t('setting.repeatInterval')">
						<n-input-number v-model:value="form.silenceHours" class="w-full" :min="0" :max="72">
							<template #suffix>{{ t("setting.hoursOnlyOnce") }}</template>
						</n-input-number>
					</n-form-item>
				</div>

				<n-form-item :label="t('setting.notifyContent')">
					<n-space :wrap="true">
						<n-checkbox v-model:checked="form.enableDisk">{{ t("setting.notifyDiskHigh") }}</n-checkbox>
						<n-checkbox v-model:checked="form.enableContainer">
							{{ t("setting.notifyContainerError") }}
						</n-checkbox>
						<n-checkbox v-model:checked="form.enableOffline">
							{{ t("setting.notifyOfflineOrToken") }}
						</n-checkbox>
						<n-checkbox v-model:checked="form.enableCert">{{ t("setting.notifyCertExpiring") }}</n-checkbox>
						<n-checkbox v-model:checked="form.enableCode">{{ t("notify.codeTasks") }}</n-checkbox>
						<n-checkbox v-model:checked="form.enableSecurity">
							{{ t("securityMonitoring.emailNotifications") }}
						</n-checkbox>
						<n-checkbox v-model:checked="form.enableSecurityLowMedium">
							{{ t("securityMonitoring.emailLowMedium") }}
						</n-checkbox>
						<n-checkbox v-model:checked="form.notifyResolved">{{ t("setting.notifyResolved") }}</n-checkbox>
					</n-space>
				</n-form-item>
			</n-form>

			<n-space>
				<n-button type="primary" :loading="saving" @click="handleSave">{{ t("commons.button.save") }}</n-button>
				<n-button :loading="testing" @click="handleTest">{{ t("setting.sendTestMail") }}</n-button>
				<n-button quaternary @click="handleEvaluate">{{ t("setting.evaluateNow") }}</n-button>
			</n-space>
			<div class="mt-2 text-xs text-slate-400">
				{{ t("setting.testMailHint") }}
			</div>
		</n-card>

		<n-card :title="t('setting.recentAlertEvents')">
			<template #header-extra>
				<n-button size="tiny" quaternary :loading="loading" @click="fetchEvents">
					{{ t("commons.button.refresh") }}
				</n-button>
			</template>
			<n-data-table :columns="eventColumns" :data="events" :bordered="false" :max-height="320" />
			<div v-if="!events.length" class="py-6 text-center text-sm text-slate-400">
				{{ t("setting.noAlertEvents") }}
			</div>
		</n-card>
	</div>
</template>
