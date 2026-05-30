<template>
	<n-drawer v-model:show="open" :mask-closable="false" width="50%" :destroy-on-close="true">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="t('file.downloadRemote')" :back="handleClose" />
			</template>

			<!-- 表单模式（URL 输入） -->
			<div v-if="!showLog" style="padding: 16px">
				<n-form ref="formRef" :model="form" label-placement="top">
					<n-form-item :label="t('file.remoteUrl')" path="url" required>
						<n-input v-model:value="form.url" :placeholder="t('file.remoteUrlPlaceholder')" />
					</n-form-item>

					<n-form-item :label="t('commons.table.name')" path="name" required>
						<n-input v-model:value="form.name" :placeholder="t('file.downloadNamePlaceholder')" />
					</n-form-item>

					<n-form-item>
						<n-checkbox :checked="form.ignoreCertificate" @update:checked="v => (form.ignoreCertificate = v)">
							{{ t("file.ignoreCertificate") }}
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
import { NDrawer, NForm, NFormItem, NInput, NCheckbox, NButton } from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { WgetFileStream } from "@/api/modules/file"
import { MsgSuccess, MsgError } from "@/utils/message"
import { useAuthStore } from "@/store/auth"

interface WgetProps {
	path: string
}

const { t } = useI18n()
const authStore = useAuthStore()

const formRef = ref<any>(null)
const loading = ref(false)
const open = ref(false)
const emit = defineEmits(["close"])

const form = reactive({
	url: "",
	name: "",
	ignoreCertificate: false
})

let downloadPath = ""

// SSE 日志相关
const showLog = ref(false)
const streamLogs = ref<string[]>([])
const logStatus = ref<"idle" | "connecting" | "streaming" | "success" | "failed">("idle")
const terminalRef = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

const logStatusLabel = computed(() => {
	switch (logStatus.value) {
		case "connecting": return "连接中..."
		case "streaming": return "下载中..."
		case "success": return "下载完成"
		case "failed": return "下载失败"
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
	eventSource = new EventSource(`${apiUrl}/file/wget/logs?key=${encodeURIComponent(sseKey)}&token=${safeToken}`)

	eventSource.onmessage = event => {
		if (event.data === "ping" || event.data === ":" || event.data === "") return
		logStatus.value = logStatus.value === "connecting" ? "streaming" : logStatus.value
		streamLogs.value.push(event.data)
		scrollToBottom()
	}

	eventSource.addEventListener("progress", event => {
		logStatus.value = "streaming"
	})

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
			MsgSuccess(t("file.downloadSuccess"))
		} else if (logStatus.value === "failed") {
			MsgError(t("file.downloadFailed"))
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
	form.url = ""
	form.name = ""
	form.ignoreCertificate = false
	showLog.value = false
	streamLogs.value = []
	logStatus.value = "idle"
	open.value = false
	emit("close", open)
}

const submit = async () => {
	if (!form.url) {
		MsgError(t("commons.msg.invalid"))
		return
	}
	if (!form.name) {
		MsgError(t("commons.msg.invalid"))
		return
	}

	loading.value = true
	try {
		const payload = { url: form.url, path: downloadPath, name: form.name, ignoreCertificate: form.ignoreCertificate }
		const res = await WgetFileStream(payload)
		const key = res?.data?.key || res?.key
		if (key) {
			startLogStream(key)
		} else {
			MsgSuccess(t("file.downloadSuccess"))
			handleClose()
		}
	} catch (e: any) {
		MsgError(e?.msg || e?.message || t("commons.msg.operationFailed"))
		console.error(e)
	} finally {
		loading.value = false
	}
}

const acceptParams = (props: WgetProps) => {
	downloadPath = props.path
	form.url = ""
	form.name = ""
	form.ignoreCertificate = false
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
