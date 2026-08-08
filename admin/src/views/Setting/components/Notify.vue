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
	enableCode: true
})

const tlsOptions = [
	{ label: "STARTTLS（587 / 25）", value: "starttls" },
	{ label: "SSL 隐式加密（465）", value: "ssl" },
	{ label: "不加密（仅限内网中继）", value: "none" }
]

// 端口和加密方式对不上是配 SMTP 最常见的错误，表现是连接一直卡到超时
const portMismatch = computed(() => {
	if (form.value.smtpPort === 465 && form.value.smtpTlsMode !== "ssl") {
		return "465 端口通常需要选「SSL 隐式加密」，选错会一直卡到超时"
	}
	if ((form.value.smtpPort === 587 || form.value.smtpPort === 25) && form.value.smtpTlsMode === "ssl") {
		return "587 / 25 端口通常是 STARTTLS，选 SSL 会连不上"
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
		message.success("已保存")
	} catch (error: any) {
	} finally {
		saving.value = false
	}
}

const handleTest = async () => {
	testing.value = true
	try {
		await testNotifyMail({ ...form.value, password: password.value })
		message.success("测试邮件已发送，请查收（也看一下垃圾箱）")
	} catch (error: any) {
	} finally {
		testing.value = false
	}
}

const handleEvaluate = async () => {
	try {
		await evaluateAlerts()
		message.success("已执行一轮评估")
		void fetchEvents()
	} catch (error: any) {
	}
}

const typeLabel: Record<string, string> = {
	disk: "磁盘",
	container: "容器异常",
	offline: "节点离线",
	unauthorized: "令牌失效",
	cert: "证书到期"
}

const statusMeta: Record<string, { label: string; type: "default" | "warning" | "error" | "success" }> = {
	pending: { label: "观察中", type: "default" },
	firing: { label: "告警中", type: "error" },
	resolved: { label: "已恢复", type: "success" }
}

const eventColumns: DataTableColumns<Notify.Event> = [
	{ title: "对象", key: "sourceName", width: 160, ellipsis: { tooltip: true } },
	{
		title: "类型",
		key: "type",
		width: 100,
		render: (row) => h(NTag, { size: "small" }, { default: () => typeLabel[row.type] || row.type })
	},
	{
		title: "状态",
		key: "status",
		width: 100,
		render: (row) => {
			const meta = statusMeta[row.status] || statusMeta.pending
			return h(NTag, { size: "small", type: meta.type }, { default: () => meta.label })
		}
	},
	{ title: "详情", key: "detail", minWidth: 240, ellipsis: { tooltip: true } },
	{
		title: "最近更新",
		key: "updatedAt",
		width: 170,
		render: (row) => formatTime(row.updatedAt || "")
	}
]

onMounted(() => {
	void fetchConfig()
	void fetchEvents()
})
</script>

<template>
	<div class="pt-2">
		<n-card title="邮件通知" class="mb-4">
			<template #header-extra>
				<n-switch v-model:value="form.enabled">
					<template #checked>已启用</template>
					<template #unchecked>已停用</template>
				</n-switch>
			</template>

			<n-alert type="info" :show-icon="false" class="mb-4 text-xs">
				磁盘占用、容器异常、节点离线达到阈值时发邮件。同一问题只在首次触发时发一封，
				持续未恢复按静默期间隔提醒，恢复后再发一封，不会每分钟刷屏。
			</n-alert>

			<n-form label-placement="left" :label-width="110" size="small">
				<div class="grid grid-cols-1 gap-x-6 md:grid-cols-2">
					<n-form-item label="SMTP 服务器">
						<n-input v-model:value="form.smtpHost" placeholder="smtp.qq.com" />
					</n-form-item>
					<n-form-item label="端口">
						<n-input-number v-model:value="form.smtpPort" class="w-full" :min="1" :max="65535" />
					</n-form-item>
					<n-form-item label="加密方式">
						<n-select v-model:value="form.smtpTlsMode" :options="tlsOptions" />
					</n-form-item>
					<n-form-item label="登录账号">
						<n-input v-model:value="form.smtpUser" placeholder="you@qq.com" />
					</n-form-item>
					<n-form-item label="密码 / 授权码">
						<n-input
							v-model:value="password"
							type="password"
							show-password-on="click"
							:placeholder="hasPassword ? '已保存，留空则不修改' : 'QQ/163 需填授权码，不是登录密码'"
						/>
					</n-form-item>
					<n-form-item label="发件人">
						<n-input v-model:value="form.smtpFrom" placeholder="留空则用登录账号" />
					</n-form-item>
				</div>

				<n-alert v-if="portMismatch" type="warning" :show-icon="true" class="mb-3 text-xs">
					{{ portMismatch }}
				</n-alert>

				<n-form-item label="收件人">
					<n-input
						v-model:value="form.receivers"
						type="textarea"
						:rows="2"
						placeholder="多个收件人用逗号、分号或换行分隔"
					/>
				</n-form-item>

				<div class="grid grid-cols-1 gap-x-6 md:grid-cols-2">
					<n-form-item label="连续命中">
						<n-input-number v-model:value="form.debounceTimes" class="w-full" :min="1" :max="10">
							<template #suffix>次才告警</template>
						</n-input-number>
					</n-form-item>
					<n-form-item label="重复提醒间隔">
						<n-input-number v-model:value="form.silenceHours" class="w-full" :min="0" :max="72">
							<template #suffix>小时（0=只发一次）</template>
						</n-input-number>
					</n-form-item>
				</div>

				<n-form-item label="通知内容">
					<n-space :wrap="true">
						<n-checkbox v-model:checked="form.enableDisk">磁盘占用过高</n-checkbox>
						<n-checkbox v-model:checked="form.enableContainer">容器异常</n-checkbox>
						<n-checkbox v-model:checked="form.enableOffline">节点离线 / 令牌失效</n-checkbox>
						<n-checkbox v-model:checked="form.enableCert">证书即将到期</n-checkbox>
						<n-checkbox v-model:checked="form.enableCode">{{ t("notify.codeTasks") }}</n-checkbox>
						<n-checkbox v-model:checked="form.notifyResolved">恢复时也通知</n-checkbox>
					</n-space>
				</n-form-item>
			</n-form>

			<n-space>
				<n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
				<n-button :loading="testing" @click="handleTest">发送测试邮件</n-button>
				<n-button quaternary @click="handleEvaluate">立即评估一次</n-button>
			</n-space>
			<div class="mt-2 text-xs text-slate-400">
				测试邮件不受上面的开关和阈值影响，只验证 SMTP 是否配置正确。
			</div>
		</n-card>

		<n-card title="最近告警事件">
			<template #header-extra>
				<n-button size="tiny" quaternary :loading="loading" @click="fetchEvents">刷新</n-button>
			</template>
			<n-data-table :columns="eventColumns" :data="events" :bordered="false" :max-height="320" />
			<div v-if="!events.length" class="py-6 text-center text-sm text-slate-400">
				暂无告警事件
			</div>
		</n-card>
	</div>
</template>
