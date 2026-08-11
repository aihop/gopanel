<template>
  <div>
    <slot
      :versionInfo="versionInfo"
      :updateInfo="updateInfo"
      :checkingUpdate="checkingUpdate"
      :updating="updating"
      :logVisible="logVisible"
      :isReading="isReading"
      :streamLogs="streamLogs"
      :logStatus="logStatus"
      :logStatusLabel="logStatusLabel"
      :logStatusTag="logStatusTag"
      :logStatusText="logStatusText"
      :effectiveNeedUpdate="effectiveNeedUpdate"
      :formatTime="formatTime"
      :fetchCurrentVersion="fetchCurrentVersion"
      :fetchUpdateInfo="fetchUpdateInfo"
      :checkUpdate="checkUpdate"
      :startUpgrade="startUpgrade"
      :setLogVisible="setLogVisible"
      :stopLogStream="stopLogStream"
      :setTerminalRef="setTerminalRef"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from "vue"
import { settingSystemCheckAPI, settingSystemUpgradeAPI, settingSystemVersionAPI } from "@/api/modules/setting"
import { useLocalesStore } from "@/store/i18n"
import { useAuthStore } from "@/store/auth"

type VersionInfo = {
	versionName: string
	versionCode: number
	buildTime: string
	installPath: string
}

type UpdateInfo = {
	needUpdate: boolean
	latestVersionCode: number
	latestVersionName: string
	title?: string
	description?: string
	content?: string
	createAt?: string
	downloadUrl?: string
	curVersion?: string
}

type TagType = "default" | "error" | "success" | "warning" | "info" | "primary"

 
const { pollIntervalMs = 0 } = defineProps<{
	pollIntervalMs?: number
}>()

const versionInfo = ref<VersionInfo>({
	versionName: "",
	versionCode: 0,
	buildTime: "",
	installPath: ""
})

const updateInfo = ref<UpdateInfo>({
	needUpdate: false,
	latestVersionCode: 0,
	latestVersionName: "",
	title: "",
	description: "",
	content: "",
	createAt: "",
	downloadUrl: "",
	curVersion: ""
})

const checkingUpdate = ref(false)
const updating = ref(false)

const localesStore = useLocalesStore()
const lang = computed(() => localesStore.locale)
const appBrand = computed(() => import.meta.env.VITE_APP_BRAND || "GoPanel")

const logVisible = ref(false)
const isReading = ref(false)
const streamLogs = ref<string[]>([])
const logStatus = ref<"idle" | "connecting" | "streaming" | "restarting" | "success" | "failed">("idle")

const currentLogName = ref("")
const pendingTargetVersion = ref("")

let eventSource: EventSource | null = null
const terminalRef = ref<HTMLElement | null>(null)
let pollTimer: number | null = null

const setTerminalRef = (el: HTMLElement | null) => {
	terminalRef.value = el
}

const scrollToBottom = () => {
	nextTick(() => {
		if (!terminalRef.value) return
		terminalRef.value.scrollTop = terminalRef.value.scrollHeight
	})
}

const setLogVisible = (value: boolean) => {
	logVisible.value = value
	if (!value) {
		stopLogStream()
	}
}

const logStatusLabel = computed(() => {
	switch (logStatus.value) {
		case "connecting":
			return "连接中"
		case "streaming":
			return "更新中"
		case "restarting":
			return "重启中"
		case "success":
			return "已完成"
		case "failed":
			return "失败"
		default:
			return "等待中"
	}
})

const logStatusTag = computed<TagType>(() => {
	switch (logStatus.value) {
		case "success":
			return "success"
		case "failed":
			return "error"
		case "restarting":
			return "warning"
		case "streaming":
		case "connecting":
			return "info"
		default:
			return "default"
	}
})

const logStatusText = computed(() => {
	switch (logStatus.value) {
		case "connecting":
			return "正在建立日志连接，请稍候..."
		case "streaming":
			return "正在下载并替换文件，日志将实时刷新"
		case "restarting":
			return "文件已替换完成，正在等待服务重启恢复"
		case "success":
			return "更新完成，版本信息已同步刷新"
		case "failed":
			return "更新未成功完成，请根据日志排查"
		default:
			return "准备查看更新日志"
	}
})

const effectiveNeedUpdate = computed<true | false | undefined>(() => {
	const latest = Number(updateInfo.value.latestVersionCode || 0)
	const current = Number(versionInfo.value.versionCode || 0)
	if (latest > 0 && current > 0) {
		return current < latest
	}
	if (typeof updateInfo.value.needUpdate === "boolean") {
		return updateInfo.value.needUpdate
	}
	return undefined
})

const formatTime = (timeStr: string) => {
	if (!timeStr) return ""
	try {
		const date = new Date(timeStr)
		return date.toLocaleString("zh-CN")
	} catch (e) {
		return timeStr
	}
}

const fetchCurrentVersion = async (silent = false) => {
	try {
		const res = await settingSystemVersionAPI()
		if (res.code === 0 && res.data) {
			let data = res.data as VersionInfo
			if (data.versionCode === 0 && data.buildTime) {
				const d = new Date(data.buildTime)
				const pad = (n: number) => n.toString().padStart(2, "0")
				const y = d.getFullYear()
				const M = pad(d.getMonth() + 1)
				const D = pad(d.getDate())
				const h = pad(d.getHours())
				const m = pad(d.getMinutes())
				data.versionCode = Number(`${y}${M}${D}${h}${m}`)
			}
			versionInfo.value = {
				versionName: data.versionName,
				versionCode: data.versionCode,
				buildTime: data.buildTime,
				installPath: data.installPath
			}
			return true
		}
		return silent
	} catch (_e) {
		return silent
	}
}

const fetchUpdateInfo = async (silent = false) => {
	checkingUpdate.value = true
	try {
		const res = await settingSystemCheckAPI({ lang: lang.value, appBrand: appBrand.value })
		if (res.code === 0 && res.data) {
			updateInfo.value = res.data as UpdateInfo
			return true
		}
		return silent
	} catch (_e) {
		return silent
	} finally {
		checkingUpdate.value = false
	}
}

const checkUpdate = async () => {
	await fetchUpdateInfo(false)
}

const appendLogLine = (line: string) => {
	if (!line) return
	if (streamLogs.value.length > 3000) {
		streamLogs.value.splice(0, streamLogs.value.length - 2000)
		streamLogs.value.unshift("... 之前的日志已折叠，请查看服务器日志文件获取完整内容 ...")
	}
	streamLogs.value.push(line)
	scrollToBottom()
}

const stopLogStream = () => {
	if (eventSource) {
		eventSource.close()
		eventSource = null
	}
	isReading.value = false
}

const refreshVersionAndUpdateState = async () => {
	await Promise.all([fetchCurrentVersion(true), fetchUpdateInfo(true)])
}

const verifyUpdateResultAfterRestart = async () => {
	logStatus.value = "restarting"
	for (let i = 0; i < 15; i++) {
		try {
			await refreshVersionAndUpdateState()
			const matched =
				!!pendingTargetVersion.value && versionInfo.value.versionName === pendingTargetVersion.value
			if (matched || effectiveNeedUpdate.value === false) {
				logStatus.value = "success"
				isReading.value = false
				appendLogLine("====== 更新完成，服务已重启 ======")
				return
			}
		} catch (_e) {
		}
		await new Promise(resolve => setTimeout(resolve, 2000))
	}
	logStatus.value = "failed"
	isReading.value = false
	appendLogLine("====== 未能确认更新结果，请稍后手动检查版本信息 ======")
}

const startLogStream = () => {
	stopLogStream()
	streamLogs.value = []
	logStatus.value = "connecting"
	isReading.value = true

	const apiUrl = (window as any).__VITE_API_URL__ || "/api"
	const authStore = useAuthStore()
			const safeToken = encodeURIComponent(authStore.auth || "")
	eventSource = new EventSource(
		`${apiUrl}/setting/system/upgrade/logs?log=${encodeURIComponent(currentLogName.value)}&token=${safeToken}`
	)

	eventSource.onmessage = event => {
		if (event.data === "ping" || event.data === ":") return
		logStatus.value = logStatus.value === "restarting" ? "restarting" : "streaming"
		appendLogLine(event.data)
		if (event.data.includes("successful update to version_code")) {
			logStatus.value = "restarting"
		}
		if (event.data.includes("restart panel")) {
			logStatus.value = "restarting"
		}
	}

	eventSource.addEventListener("status", event => {
		const data = (event as MessageEvent).data
		if (data === "success") {
			logStatus.value = "restarting"
		} else if (data === "failed") {
			logStatus.value = "failed"
		} else if (data === "running") {
			logStatus.value = "streaming"
		}
	})

	eventSource.addEventListener("eof", async () => {
		stopLogStream()
		if (logStatus.value !== "failed") {
			await verifyUpdateResultAfterRestart()
		}
	})

	eventSource.onerror = async () => {
		stopLogStream()
		appendLogLine("====== 日志连接已断开，正在校验更新结果... ======")
		await verifyUpdateResultAfterRestart()
	}
}

const startUpgrade = async (params: { containerName: string; currentVersion: string; targetVersion: string }) => {
	updating.value = true
	try {
		const res = await settingSystemUpgradeAPI(params)
		if (res.code !== 0) {
			return { ok: false as const, msg: res.msg || "更新失败" }
		}
		currentLogName.value = (res.data as any)?.log || "console_update"
		pendingTargetVersion.value = params.targetVersion
		logVisible.value = true
		startLogStream()
		return { ok: true as const, log: currentLogName.value }
	} catch (_e) {
		return { ok: false as const, msg: "更新异常" }
	} finally {
		updating.value = false
	}
}

onMounted(() => {
	fetchCurrentVersion(true)
	fetchUpdateInfo(true)
	if (pollIntervalMs > 0) {
		pollTimer = window.setInterval(() => {
			fetchUpdateInfo(true)
		}, pollIntervalMs)
	}
})

onUnmounted(() => {
	stopLogStream()
	if (pollTimer !== null) {
		window.clearInterval(pollTimer)
		pollTimer = null
	}
})
</script>
