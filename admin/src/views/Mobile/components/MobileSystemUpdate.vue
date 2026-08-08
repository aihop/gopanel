<script setup lang="ts">
import { checkMobileSystemUpdate, getMobileSystemVersion, startMobileSystemUpgrade, type MobileUpdateInfo, type MobileVersionInfo } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useDialog, useMessage } from "naive-ui"

type UpdateStatus = "idle" | "connecting" | "updating" | "restarting" | "success" | "failed"

const { locale, t } = useI18n({ messages: mobileMessages })
withDefaults(defineProps<{ showCurrentVersion?: boolean }>(), { showCurrentVersion: false })
const dialog = useDialog()
const message = useMessage()
const versionInfo = ref<MobileVersionInfo | null>(null)
const updateInfo = ref<MobileUpdateInfo | null>(null)
const checking = ref(false)
const checkError = ref("")
const upgrading = ref(false)
const drawerVisible = ref(false)
const logs = ref<string[]>([])
const logStatus = ref<UpdateStatus>("idle")
const logContainer = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

const appBrand = computed(() => import.meta.env.VITE_APP_BRAND || "GoPanel")
const needUpdate = computed(() => {
	if (!versionInfo.value || !updateInfo.value) return false
	if (versionInfo.value.versionCode > 0 && updateInfo.value.latestVersionCode > 0) {
		return versionInfo.value.versionCode < updateInfo.value.latestVersionCode
	}
	return updateInfo.value.needUpdate
})
const statusType = computed(() => {
	if (logStatus.value === "success") return "success"
	if (logStatus.value === "failed") return "error"
	if (logStatus.value === "restarting") return "warning"
	return "info"
})
const statusLabel = computed(() => t(`mobile.updateStatus_${logStatus.value}`))

function appendLog(line: string) {
	if (!line) return
	if (logs.value.length >= 1000) logs.value.splice(0, 200)
	logs.value.push(line)
	void nextTick(() => {
		if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
	})
}

function closeLogStream() {
	eventSource?.close()
	eventSource = null
}

async function loadUpdate(silent = false) {
	checking.value = true
	if (!silent) checkError.value = ""
	try {
		const [version, update] = await Promise.all([
			getMobileSystemVersion(),
			checkMobileSystemUpdate(locale.value, appBrand.value)
		])
		if (!version || !update) throw new Error(t("mobile.updateCheckFailed"))
		versionInfo.value = version
		updateInfo.value = update
		checkError.value = ""
	} catch (error) {
		checkError.value = error instanceof Error ? error.message : t("mobile.updateCheckFailed")
	} finally {
		checking.value = false
	}
}

async function verifyAfterRestart() {
	logStatus.value = "restarting"
	for (let attempt = 0; attempt < 15; attempt += 1) {
		await new Promise(resolve => window.setTimeout(resolve, 2000))
		try {
			const version = await getMobileSystemVersion()
			versionInfo.value = version
			if (version.versionName === updateInfo.value?.latestVersionName) {
				logStatus.value = "success"
				appendLog(t("mobile.updateCompleteLog"))
				await loadUpdate(true)
				return
			}
		} catch (_error) {
		}
	}
	logStatus.value = "failed"
	appendLog(t("mobile.updateVerifyFailed"))
}

function openLogStream(logName: string) {
	closeLogStream()
	logs.value = []
	logStatus.value = "connecting"
	const apiUrl = (window as typeof window & { __VITE_API_URL__?: string }).__VITE_API_URL__ || import.meta.env.VITE_API_URL || "/api"
	eventSource = new EventSource(`${apiUrl}/mobile/app/system/upgrade/logs?log=${encodeURIComponent(logName)}`, {
		withCredentials: true
	})
	eventSource.onmessage = event => {
		if (!event.data || event.data === "ping") return
		logStatus.value = event.data.includes("restart panel") ? "restarting" : "updating"
		appendLog(event.data)
	}
	eventSource.addEventListener("status", event => {
		const status = (event as MessageEvent).data
		if (status === "failed") logStatus.value = "failed"
		else if (status === "success") logStatus.value = "success"
		else if (status === "running") logStatus.value = "updating"
	})
	eventSource.addEventListener("eof", async () => {
		closeLogStream()
		if (logStatus.value === "failed") return
		await verifyAfterRestart()
	})
	eventSource.onerror = async () => {
		closeLogStream()
		appendLog(t("mobile.updateConnectionLost"))
		await verifyAfterRestart()
	}
}

async function startUpgrade() {
	if (!versionInfo.value || !updateInfo.value) return
	upgrading.value = true
	try {
		const result = await startMobileSystemUpgrade(
			versionInfo.value.versionName,
			updateInfo.value.latestVersionName,
			locale.value
		)
		drawerVisible.value = true
		openLogStream(result.log)
		message.success(t("mobile.updateStarted"))
	} catch (error) {
		// 错误提示由请求拦截器统一处理
	} finally {
		upgrading.value = false
	}
}

function confirmUpgrade() {
	dialog.warning({
		title: t("mobile.updateConfirmTitle"),
		content: t("mobile.updateConfirm", { version: updateInfo.value?.latestVersionName || "-" }),
		positiveText: t("mobile.updateNow"),
		negativeText: t("mobile.cancel"),
		onPositiveClick: startUpgrade
	})
}

onMounted(() => void loadUpdate())
onBeforeUnmount(closeLogStream)
</script>

<template>
	<section v-if="showCurrentVersion" class="mb-4 rounded-2xl bg-white p-4 shadow-sm">
		<div class="flex items-center justify-between gap-3">
			<div>
				<div class="text-sm font-medium text-slate-900">{{ t("mobile.updateLogs") }}</div>
				<div class="mt-1 text-xs text-slate-500">{{ t("setting.currentVersion") }} {{ versionInfo?.versionName || "-" }}</div>
			</div>
			<n-button size="small" secondary :loading="checking" @click="loadUpdate()">
				{{ t("mobile.refresh") }}
			</n-button>
		</div>
	</section>

	<n-alert v-if="checkError" type="error" :show-icon="false" class="mb-4 rounded-2xl">
		<div class="flex items-center justify-between gap-3">
			<span class="text-sm">{{ t("mobile.updateCheckFailed") }}</span>
			<n-button size="small" secondary :loading="checking" @click="loadUpdate()">{{ t("mobile.retry") }}</n-button>
		</div>
	</n-alert>

	<section v-else-if="needUpdate" class="mb-4 overflow-hidden rounded-2xl bg-gradient-to-br from-blue-600 to-indigo-700 p-4 text-white shadow-sm">
		<div class="flex items-start gap-3">
			<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white/15">
				<Icon name="carbon:cloud-download" :size="23" />
			</div>
			<div class="min-w-0 flex-1">
				<div class="font-semibold">{{ updateInfo?.title || t("mobile.updateAvailable") }}</div>
				<p class="mt-1 line-clamp-2 text-sm text-blue-100">
					{{ updateInfo?.description || t("mobile.updateAvailableHint") }}
				</p>
				<div class="mt-3 flex items-center justify-between gap-3">
					<span class="truncate text-xs text-blue-100">
						{{ versionInfo?.versionName || "-" }} → {{ updateInfo?.latestVersionName || "-" }}
					</span>
					<n-button size="small" type="primary" color="#ffffff" text-color="#1d4ed8" :loading="upgrading" @click="confirmUpgrade">
						{{ t("mobile.updateNow") }}
					</n-button>
				</div>
			</div>
		</div>
	</section>

	<n-drawer v-model:show="drawerVisible" placement="bottom" height="min(620px, 82dvh)" :mask-closable="!eventSource">
		<n-drawer-content :title="t('mobile.updateLogs')" :closable="!eventSource" body-content-style="padding: 16px;">
			<div class="flex h-full flex-col gap-3">
				<div class="flex items-center justify-between gap-3 rounded-xl bg-slate-100 px-3 py-2">
					<span class="text-sm text-slate-600">{{ t("mobile.updateStatusHint") }}</span>
					<n-tag :type="statusType" :bordered="false" round>{{ statusLabel }}</n-tag>
				</div>
				<div ref="logContainer" class="min-h-0 flex-1 overflow-y-auto rounded-xl bg-[#0b1020] p-3 font-mono text-xs leading-5 text-slate-200">
					<div v-if="logs.length === 0" class="text-slate-500">{{ t("mobile.updateWaitingLogs") }}</div>
					<div v-for="(line, index) in logs" :key="index" class="whitespace-pre-wrap break-words">{{ line }}</div>
				</div>
			</div>
		</n-drawer-content>
	</n-drawer>
</template>
