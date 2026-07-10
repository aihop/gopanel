<script setup lang="ts">
import { ref, reactive, watch, computed } from "vue"
import { useMessage } from "naive-ui"
import { cronjobCreateAPI, cronjobUpdateAPI } from "@/api/modules/cronjob"
import { databaseServerListAPI, databaseListAPI } from "@/api/modules/database"

const props = defineProps<{
	show: boolean
	job: any | null
}>()

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "saved"): void
}>()

const message = useMessage()
const saving = ref(false)

const typeOptions = [
	{ label: "执行 Shell 脚本", value: "shell" },
	{ label: "数据库备份", value: "db_backup" },
	{ label: "清理面板日志", value: "log_clean" },
	{ label: "SSL 证书续期", value: "ssl_renew" }
]

const periodOptions = [
	{ label: "每小时", value: "hourly" },
	{ label: "每天", value: "daily" },
	{ label: "每周", value: "weekly" },
	{ label: "每月", value: "monthly" },
	{ label: "自定义 cron 表达式", value: "custom" }
]

const weekOptions = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"].map((label, value) => ({ label, value }))

const logTypeOptions = [
	{ label: "操作日志", value: "operation" },
	{ label: "登录日志", value: "login" },
	{ label: "全部", value: "all" }
]

const form = reactive({
	name: "",
	type: "shell",
	script: "",
	serverId: null as number | null,
	dbName: "",
	retainCopies: 0,
	logType: "operation"
})

const periodType = ref<"hourly" | "daily" | "weekly" | "monthly" | "custom">("daily")
const periodMinute = ref(0)
const periodHour = ref(3)
const periodWeek = ref(1)
const periodDay = ref(1)
const customSpec = ref("0 3 * * *")

const serverOptions = ref<{ label: string; value: number; type: string }[]>([])
const dbNameOptions = ref<{ label: string; value: string }[]>([])

const isEdit = computed(() => !!props.job)

const spec = computed(() => {
	switch (periodType.value) {
		case "hourly":
			return `${periodMinute.value} * * * *`
		case "daily":
			return `${periodMinute.value} ${periodHour.value} * * *`
		case "weekly":
			return `${periodMinute.value} ${periodHour.value} * * ${periodWeek.value}`
		case "monthly":
			return `${periodMinute.value} ${periodHour.value} ${periodDay.value} * *`
		default:
			return customSpec.value.trim()
	}
})

// 尝试把已有的 cron 表达式识别回预设周期的字段，识别不了就退回"自定义"原样展示
const parseSpec = (raw: string) => {
	const parts = raw.trim().split(/\s+/)
	if (parts.length !== 5) {
		periodType.value = "custom"
		customSpec.value = raw
		return
	}
	const [minute, hour, day, month, week] = parts
	if (hour === "*" && day === "*" && month === "*" && week === "*" && /^\d+$/.test(minute)) {
		periodType.value = "hourly"
		periodMinute.value = Number(minute)
		return
	}
	if (/^\d+$/.test(hour) && day === "*" && month === "*" && week === "*" && /^\d+$/.test(minute)) {
		periodType.value = "daily"
		periodHour.value = Number(hour)
		periodMinute.value = Number(minute)
		return
	}
	if (/^\d+$/.test(hour) && day === "*" && month === "*" && /^\d+$/.test(week) && /^\d+$/.test(minute)) {
		periodType.value = "weekly"
		periodHour.value = Number(hour)
		periodMinute.value = Number(minute)
		periodWeek.value = Number(week)
		return
	}
	if (/^\d+$/.test(hour) && /^\d+$/.test(day) && month === "*" && week === "*" && /^\d+$/.test(minute)) {
		periodType.value = "monthly"
		periodHour.value = Number(hour)
		periodMinute.value = Number(minute)
		periodDay.value = Number(day)
		return
	}
	periodType.value = "custom"
	customSpec.value = raw
}

const loadServers = async () => {
	try {
		const res: any = await databaseServerListAPI({})
		const items = Array.isArray(res.data) ? res.data : res.data?.items || []
		serverOptions.value = items.map((s: any) => ({ label: s.name, value: s.id, type: s.type }))
	} catch {
		serverOptions.value = []
	}
}

const loadDbNames = async (serverId: number | null) => {
	if (!serverId) {
		dbNameOptions.value = []
		return
	}
	try {
		const res: any = await databaseListAPI({
			page: 1,
			limit: 500,
			wheres: [{ field: "server_id", rule: "eq", val: serverId.toString() }]
		})
		const items = Array.isArray(res.data?.items) ? res.data.items : []
		dbNameOptions.value = items.map((item: any) => ({ label: item.name, value: item.name }))
	} catch {
		dbNameOptions.value = []
	}
}

watch(
	() => form.serverId,
	value => {
		void loadDbNames(value)
	}
)

watch(
	() => props.show,
	async visible => {
		if (!visible) return
		await loadServers()
		if (props.job) {
			form.name = props.job.name
			form.type = props.job.type
			form.script = props.job.script || ""
			form.serverId = props.job.serverID || null
			form.dbName = props.job.dbName || ""
			form.retainCopies = props.job.retainCopies || 0
			form.logType = props.job.logType || "operation"
			parseSpec(props.job.spec || "0 3 * * *")
			if (form.serverId) await loadDbNames(form.serverId)
		} else {
			form.name = ""
			form.type = "shell"
			form.script = ""
			form.serverId = null
			form.dbName = ""
			form.retainCopies = 0
			form.logType = "operation"
			periodType.value = "daily"
			periodMinute.value = 0
			periodHour.value = 3
			periodWeek.value = 1
			periodDay.value = 1
			customSpec.value = "0 3 * * *"
		}
	}
)

const selectedServerType = computed(() => serverOptions.value.find(s => s.value === form.serverId)?.type || "")

const close = () => emit("update:show", false)

const handleSubmit = async () => {
	if (!form.name.trim()) {
		message.warning("请填写任务名称")
		return
	}
	if (form.type === "shell" && !form.script.trim()) {
		message.warning("请填写脚本内容")
		return
	}
	if (form.type === "db_backup" && (!form.serverId || !form.dbName)) {
		message.warning("请选择数据库服务器和数据库")
		return
	}
	if (form.type === "log_clean" && !form.logType) {
		message.warning("请选择要清理的日志类型")
		return
	}
	if (!spec.value || spec.value.trim().split(/\s+/).length !== 5) {
		message.warning("cron 表达式格式不正确，应为 5 段，例如：0 3 * * *")
		return
	}

	const payload = {
		name: form.name,
		type: form.type,
		spec: spec.value,
		script: form.script,
		serverID: form.serverId || 0,
		dbType: selectedServerType.value,
		dbName: form.dbName,
		retainCopies: form.retainCopies,
		logType: form.logType
	}

	saving.value = true
	try {
		if (isEdit.value) {
			await cronjobUpdateAPI({ id: props.job.id, ...payload })
		} else {
			await cronjobCreateAPI(payload)
		}
		message.success("保存成功")
		emit("saved")
	} catch (err: any) {
		message.error(err?.message || "保存失败")
	} finally {
		saving.value = false
	}
}
</script>

<template>
	<n-modal
		:show="show"
		preset="card"
		:title="isEdit ? '编辑计划任务' : '新建计划任务'"
		style="width: 640px"
		:bordered="false"
		:segmented="false"
		@update:show="(val: boolean) => !val && close()"
	>
		<n-form label-placement="left" label-width="100">
			<n-form-item label="任务名称">
				<n-input v-model:value="form.name" placeholder="请输入任务名称" />
			</n-form-item>
			<n-form-item label="任务类型">
				<n-select v-model:value="form.type" :options="typeOptions" :disabled="isEdit" />
			</n-form-item>

			<n-form-item label="执行周期">
				<n-flex vertical style="width: 100%">
					<n-select v-model:value="periodType" :options="periodOptions" />
					<n-flex v-if="periodType !== 'custom'" align="center">
						<n-select
							v-if="periodType === 'weekly'"
							v-model:value="periodWeek"
							:options="weekOptions"
							style="width: 100px"
						/>
						<n-input-number
							v-if="periodType === 'monthly'"
							v-model:value="periodDay"
							:min="1"
							:max="28"
							style="width: 100px"
						>
							<template #suffix>日</template>
						</n-input-number>
						<n-input-number
							v-if="periodType !== 'hourly'"
							v-model:value="periodHour"
							:min="0"
							:max="23"
							style="width: 100px"
						>
							<template #suffix>时</template>
						</n-input-number>
						<n-input-number v-model:value="periodMinute" :min="0" :max="59" style="width: 100px">
							<template #suffix>分</template>
						</n-input-number>
					</n-flex>
					<n-input v-else v-model:value="customSpec" placeholder="例如：0 3 * * *" />
					<div class="text-xs text-gray-400">最终 cron 表达式：{{ spec }}</div>
				</n-flex>
			</n-form-item>

			<n-form-item v-if="form.type === 'shell'" label="脚本内容">
				<n-input
					v-model:value="form.script"
					type="textarea"
					:rows="8"
					style="font-family: monospace"
					placeholder="#!/bin/bash&#10;echo hello"
				/>
			</n-form-item>

			<template v-if="form.type === 'db_backup'">
				<n-form-item label="数据库服务器">
					<n-select
						v-model:value="form.serverId"
						:options="serverOptions"
						placeholder="请选择数据库服务器"
					/>
				</n-form-item>
				<n-form-item label="数据库">
					<n-select
						v-model:value="form.dbName"
						:options="dbNameOptions"
						:disabled="!form.serverId"
						placeholder="请选择数据库"
					/>
				</n-form-item>
				<n-form-item label="保留份数">
					<n-input-number v-model:value="form.retainCopies" :min="0" style="width: 100%" />
					<template #feedback>0 表示不自动清理旧备份</template>
				</n-form-item>
			</template>

			<n-form-item v-if="form.type === 'log_clean'" label="日志类型">
				<n-select v-model:value="form.logType" :options="logTypeOptions" />
			</n-form-item>

			<n-alert v-if="form.type === 'ssl_renew'" type="info">
				自动续签所有开启了"自动续签"且临近到期（&lt;7 天）的证书，无需额外配置。
			</n-alert>
		</n-form>
		<template #footer>
			<n-flex justify="end">
				<n-button @click="close">取消</n-button>
				<n-button type="primary" :loading="saving" @click="handleSubmit">保存</n-button>
			</n-flex>
		</template>
	</n-modal>
</template>
