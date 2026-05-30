<template>
	<n-drawer v-model:show="open" :mask-closable="false" width="40%" :destroy-on-close="true">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="t('file.downloadRemote')" :back="handleClose" />
			</template>

			<!-- 表单模式（URL 输入） -->
			<div v-if="!showProgress" style="padding: 16px">
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

			<!-- 进度模式 -->
			<div v-else style="padding: 16px; text-align: center;">
				<div style="margin-bottom: 16px;">
					<span style="font-size: 14px; color: #64748b;">{{ statusText }}</span>
				</div>
				<div style="margin-bottom: 8px;">
					<n-progress
						:percentage="Math.round(progressPercent)"
						:indicator-placement="'inside'"
						:height="28"
						:border-radius="6"
						:color="progressColor"
						:rail-color="'#e2e8f0'"
						:status="progressStatus"
					/>
				</div>
				<div v-if="downloadedSize" style="margin-bottom: 16px; font-size: 12px; color: #94a3b8;">
					{{ downloadedSize }} / {{ totalSize }}
				</div>
				<div style="font-size: 12px; color: #64748b; margin-bottom: 16px;">
					{{ progressPercent.toFixed(1) }}%
				</div>
			</div>

			<template #footer>
				<div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px">
					<n-button @click="handleClose" :disabled="downloading">
						{{ showProgress ? t("commons.button.close") : t("commons.button.cancel") }}
					</n-button>
					<n-button v-if="!showProgress" type="primary" @click="submit" :loading="loading">
						{{ t("commons.button.confirm") }}
					</n-button>
					<n-button v-if="downloading" type="error" @click="cancelDownload" :loading="cancelling">
						{{ t("commons.button.stop") || "终止下载" }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onUnmounted } from "vue"
import { useI18n } from "vue-i18n"
import { NDrawer, NForm, NFormItem, NInput, NCheckbox, NButton, NProgress } from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { WgetFileStream, WgetCancel } from "@/api/modules/file"
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
let currentKey = ""

// 进度
const showProgress = ref(false)
const downloading = ref(false)
const cancelling = ref(false)
const progressPercent = ref(0)
const downloadedSize = ref("")
const totalSize = ref("")
const statusText = ref("")
const progressStatus = ref<"default" | "success" | "error" | "warning">("default")
let eventSource: EventSource | null = null

const progressColor = computed(() => {
	if (progressStatus.value === "error") return "#ef4444"
	if (progressStatus.value === "success") return "#22c55e"
	if (progressStatus.value === "warning") return "#f59e0b"
	return "#2080f0"
})

const stopLogStream = () => {
	if (eventSource) {
		eventSource.close()
		eventSource = null
	}
}

const formatBytes = (bytes: number): string => {
	if (bytes === 0) return "0 B"
	const units = ["B", "KB", "MB", "GB"]
	const k = 1024
	const i = Math.floor(Math.log(bytes) / Math.log(k))
	return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + units[i]
}

const startProgressStream = (sseKey: string) => {
	stopLogStream()
	showProgress.value = true
	downloading.value = true
	progressPercent.value = 0
	progressStatus.value = "default"
	statusText.value = t("commons.loading") || "下载中..."
	currentKey = sseKey

	const apiUrl = (window as any).__VITE_API_URL__ || "/api"
	const safeToken = encodeURIComponent(authStore.getAuth() || authStore.auth || "")
	eventSource = new EventSource(`${apiUrl}/file/wget/logs?key=${encodeURIComponent(sseKey)}&token=${safeToken}`)

	// progress 事件：更新进度百分比
	eventSource.addEventListener("progress", event => {
		const val = parseFloat((event as MessageEvent).data)
		if (!isNaN(val)) {
			progressPercent.value = val
		}
	})

	eventSource.addEventListener("status", event => {
		const data = (event as MessageEvent).data
		if (data === "success") {
			progressStatus.value = "success"
			progressPercent.value = 100
			statusText.value = t("file.downloadSuccess")
			downloading.value = false
		} else if (data === "failed") {
			progressStatus.value = "error"
			statusText.value = t("file.downloadFailed")
			downloading.value = false
		} else if (data === "cancelled") {
			progressStatus.value = "warning"
			statusText.value = "下载已取消"
			downloading.value = false
		}
	})

	// log 事件：更新状态描述文本
	eventSource.onmessage = event => {
		if (event.data === "ping" || event.data === ":" || event.data === "") return
		// 只取最后一条作为状态描述
		if (event.data.indexOf("已提交") === -1) {
			statusText.value = event.data
		}
	}

	eventSource.addEventListener("eof", () => {
		stopLogStream()
		if (progressStatus.value === "success") {
			MsgSuccess(t("file.downloadSuccess"))
		} else if (progressStatus.value === "error") {
			MsgError(t("file.downloadFailed"))
		}
		emit("close")
	})

	eventSource.onerror = () => {
		stopLogStream()
		if (progressStatus.value === "success") {
			MsgSuccess(t("file.downloadSuccess"))
		} else if (progressStatus.value === "warning") {
			// 已取消，不需要额外提示
		} else {
			progressStatus.value = "error"
			statusText.value = "连接已断开"
		}
		downloading.value = false
	}
}

const cancelDownload = async () => {
	if (!currentKey) return
	cancelling.value = true
	try {
		await WgetCancel({ key: currentKey })
		statusText.value = "正在终止..."
	} catch (e: any) {
		MsgError(e?.msg || "取消失败")
	} finally {
		cancelling.value = false
	}
}

const handleClose = () => {
	stopLogStream()
	currentKey = ""
	form.url = ""
	form.name = ""
	form.ignoreCertificate = false
	showProgress.value = false
	downloading.value = false
	progressPercent.value = 0
	progressStatus.value = "default"
	downloadedSize.value = ""
	totalSize.value = ""
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
		const key = res?.data?.key || ""
		if (key) {
			startProgressStream(key)
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
	showProgress.value = false
	progressPercent.value = 0
	currentKey = ""
	open.value = true
}

defineExpose({ acceptParams })

onUnmounted(() => {
	stopLogStream()
})
</script>
