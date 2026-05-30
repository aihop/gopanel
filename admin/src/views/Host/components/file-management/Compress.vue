<template>
	<n-drawer v-model:show="open" :mask-closable="false" width="50%" :destroy-on-close="true">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="title" :back="handleClose" />
			</template>

			<!-- 表单模式（压缩参数配置） -->
			<div v-if="!showLog" style="padding: 16px">
				<n-form ref="fileFormRef" :model="form" label-placement="top">
					<n-form-item :label="t('file.compressType')" path="type">
						<n-select v-model:value="form.type" :options="options" />
					</n-form-item>

					<n-form-item :label="t('commons.table.name')" path="name">
						<div class="flex items-center gap-2">
							<n-input v-model:value="form.name" class="flex-1" />
							<span class="text-sm text-slate-500">{{ extension }}</span>
						</div>
					</n-form-item>

					<n-form-item :label="t('file.compressDst')" path="dst">
						<n-input v-model:value="form.dst">
							<template #prefix></template>
						</n-input>
					</n-form-item>

					<n-form-item v-if="form.type === 'tar.gz'">
						<n-input v-model:value="form.secret" :placeholder="t('setting.compressPassword')" />
					</n-form-item>

					<n-form-item>
						<n-checkbox :checked="form.replace" @update:checked="value => (form.replace = value)">
							{{ t("file.replace") }}
						</n-checkbox>
					</n-form-item>
				</n-form>
			</div>

			<!-- 日志模式（SSE 实时进度） -->
			<div v-else style="padding: 16px">
				<div style="display:flex; justify-content: space-between; align-items:center; margin-bottom: 12px;">
					<div style="font-size: 12px; color: #64748b;">{{ logStatusLabel }}</div>
				</div>
				<div
					ref="terminalRef"
					style="height: 50vh; overflow: auto; background: #0b1020; color: #e2e8f0; border-radius: 8px; padding: 12px; font-size: 12px; line-height: 18px;"
				>
					<div v-for="(line, index) in streamLogs" :key="index" style="white-space: pre-wrap; word-break: break-word;">
						{{ line }}
					</div>
				</div>
			</div>

			<template #footer>
				<div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px">
					<n-button @click="handleClose" :disabled="!showLog && loading">
						{{ showLog ? t("commons.button.close") : t("commons.button.cancel") }}
					</n-button>
					<n-button v-if="!showLog" type="primary" @click="submit" :loading="loading">
						{{ t("commons.button.confirm") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, nextTick, onUnmounted } from "vue"
import { useI18n } from "vue-i18n"
import { NDrawer, NForm, NFormItem, NSelect, NInput, NCheckbox, NButton } from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { fileCompressAPI } from "@/api/modules/file"
import { CompressExtension, CompressType } from "@/enums/files"
import { MsgSuccess, MsgError } from "@/utils/message"
import { useAuthStore } from "@/store/auth"
import type { File as FileType } from "@/api/interface/file"
import { i18n } from "@/i18n"

interface CompressProps {
	files: Array<any>
	dst: string
	name: string
	operate: string
}

const { t } = useI18n()
const authStore = useAuthStore()

const fileFormRef = ref<any>(null)
const loading = ref(false)

const form = reactive<FileType.FileCompress>({
	files: [],
	type: "zip",
	dst: "",
	name: "",
	replace: false,
	secret: ""
})

const options = ref<{ label: string; value: string }[]>([])
const open = ref(false)
const title = ref("")
const operate = ref("compress")
const emit = defineEmits(["close"])

// SSE 日志相关
const showLog = ref(false)
const streamLogs = ref<string[]>([])
const logStatus = ref<"idle" | "connecting" | "streaming" | "success" | "failed">("idle")
const terminalRef = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

const extension = computed(() => CompressExtension[form.type as keyof typeof CompressExtension] || "")

const logStatusLabel = computed(() => {
	switch (logStatus.value) {
		case "connecting": return "连接中..."
		case "streaming": return "打包中..."
		case "success": return "打包完成"
		case "failed": return "打包失败"
		default: return ""
	}
})

const scrollToBottom = () => {
	nextTick(() => {
		if (!terminalRef.value) return
		terminalRef.value.scrollTop = terminalRef.value.scrollHeight
	})
}

const stopLogStream = () => {
	if (eventSource) {
		eventSource.close()
		eventSource = null
	}
}

const startLogStream = (sseKey: string) => {
	stopLogStream()
	logStatus.value = "connecting"
	showLog.value = true

	const apiUrl = (window as any).__VITE_API_URL__ || "/api"
	const safeToken = encodeURIComponent(authStore.getAuth() || authStore.auth || "")
	eventSource = new EventSource(`${apiUrl}/file/compress/logs?key=${encodeURIComponent(sseKey)}&token=${safeToken}`)

	eventSource.onmessage = event => {
		if (event.data === "ping" || event.data === ":" || event.data === "") return
		logStatus.value = logStatus.value === "connecting" ? "streaming" : logStatus.value
		streamLogs.value.push(event.data)
		scrollToBottom()
	}

	eventSource.addEventListener("status", event => {
		const data = (event as MessageEvent).data
		if (data === "success") {
			logStatus.value = "success"
		} else if (data === "failed") {
			logStatus.value = "failed"
		} else {
			logStatus.value = "streaming"
		}
	})

	eventSource.addEventListener("eof", () => {
		stopLogStream()
		if (logStatus.value === "success") {
			MsgSuccess(t("file.compressSuccess"))
		} else if (logStatus.value === "failed") {
			MsgError(t("file.compressFailed"))
		}
		emit("close")
	})

	eventSource.onerror = () => {
		streamLogs.value.push("连接已断开或发生错误")
		scrollToBottom()
		stopLogStream()
		logStatus.value = logStatus.value === "success" ? "success" : "failed"
	}
}

const handleClose = () => {
	stopLogStream()
	// reset form
	form.files = []
	form.type = "zip"
	form.dst = ""
	form.name = ""
	form.replace = false
	form.secret = ""
	showLog.value = false
	streamLogs.value = []
	logStatus.value = "idle"
	open.value = false
	emit("close", open)
}

const validate = () => {
	if (!form.type) {
		MsgError(t("commons.msg.invalid"))
		return false
	}
	if (!form.dst) {
		MsgError(t("commons.msg.invalid"))
		return false
	}
	if (!form.name) {
		MsgError(t("commons.msg.invalid"))
		return false
	}
	return true
}

const submit = async () => {
	if (!validate()) return
	const payload = { ...form, name: form.name + extension.value }
	loading.value = true
	try {
		const res = await fileCompressAPI(payload as FileType.FileCompress)
		const key = res?.data?.key || ""
		if (key) {
			startLogStream(key)
		} else {
			MsgSuccess(t("file.compressSuccess"))
			handleClose()
		}
	} catch (e: any) {
		MsgError(e?.msg || e?.message || t("commons.msg.operationFailed"))
		console.error(e)
	} finally {
		loading.value = false
	}
}

const acceptParams = (props: CompressProps) => {
	form.files = props.files
	form.dst = props.dst
	form.name = props.name
	operate.value = props.operate

	options.value = Object.keys(CompressType)
		.map(k => CompressType[k as keyof typeof CompressType])
		.filter(Boolean)
		.map(v => ({ label: v, value: v }))

	title.value = t("file." + props.operate)
	showLog.value = false
	streamLogs.value = []
	logStatus.value = "idle"
	open.value = true
}

defineExpose({ acceptParams })

onUnmounted(() => {
	stopLogStream()
})
</script>
