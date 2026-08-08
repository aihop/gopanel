import { computed, onMounted, ref } from "vue"
import {
	containerDaemonConfigAPI,
	containerInstanceOperateAPI,
	containerValidateAPI,
	loadDaemonFile,
	updateDaemonByfile,
	updateDaemonUpdate
} from "../../../api/modules/container"
import { isSucc } from "../../../utils/is"
import { buildRuntimeBadgeText, buildRuntimeDetailText } from "../../../utils/runtime"
import { containerRuntimeMessages } from "../../../i18n/locales/containerRuntime"
import { useI18n } from "vue-i18n"
import { useContainerRepair } from "./useContainerRepair"

export const useContainerSetting = (message: any) => {
	const { t } = useI18n({ messages: containerRuntimeMessages })
	const statusLoading = ref(false)
	const reloadLoading = ref(false)
	const daemon = ref<any>({})
	const daemonLoading = ref(false)
	const daemonRetryCount = ref(0)
	const daemonRetryTimer = ref<number | null>(null)
	const validate = ref<any>(null)
	const editingMirrorUrls = ref("")
	const showConfirmationModal = ref(false)
	const confirmationInput = ref("")
	const mirrorSaveLoading = ref(false)
	const dockerConf = ref("")
	const showRestartConfirm = ref(false)
	const saveLoading = ref(false)
	const cgroupInput = ref("")
	const logPruneLoading = ref(false)
	let iptablesTarget = false
	let liveRestoreTarget = false

	const currentRuntimeHost = computed(() => validate.value?.runtimeHost || "-")
	const runtimeBadgeText = computed(() =>
		validate.value?.summary?.runtimeInstalled === false
			? t("containerRuntime.notInstalled")
			: buildRuntimeBadgeText(validate.value, {
					kindFallback: t("containerRuntime.runtimeFallback"),
					rootlessLabel: "rootless",
					rootfulLabel: "rootful",
					defaultModeLabel: "default"
				})
	)
	const runtimeDetailText = computed(() => {
		const detail = buildRuntimeDetailText(validate.value, {
			kindFallback: t("containerRuntime.runtimeFallback"),
			userFallback: t("containerRuntime.panelDefault"),
			runtimePrefix: t("containerRuntime.runtimePrefix"),
			runUserPrefix: t("containerRuntime.runUserPrefix")
		})
		return `${detail} · ${t("containerRuntime.currentPrefix")}${currentRuntimeHost.value}`
	})
	const dockerOnly = computed(() => !!validate.value?.runtimeKind && validate.value.runtimeKind !== "docker")
	const canAutoRepair = computed(() => {
		if (!validate.value) return false
		if (validate.value?.os !== "linux") return false
		if (validate.value?.runtimeKind === "podman") return true
		return validate.value?.runtimeKind === "docker" && !validate.value?.cli?.docker && validate.value?.cli?.podman
	})
	const logSwitchValue = computed(
		() =>
			!!(
				daemon.value.logMaxSize &&
				daemon.value.logMaxFile &&
				daemon.value.logMaxSize !== "" &&
				daemon.value.logMaxFile !== ""
			)
	)

	function clearDaemonRetry() {
		if (daemonRetryTimer.value) {
			window.clearTimeout(daemonRetryTimer.value)
			daemonRetryTimer.value = null
		}
	}

	async function updateDockerStatus(operation: string) {
		if (operation === "restart") reloadLoading.value = true
		else statusLoading.value = true
		try {
			const res: any = await containerInstanceOperateAPI({ operation })
			if (isSucc(res.code)) {
				getDaemon()
			}
		} finally {
			if (operation === "restart") reloadLoading.value = false
			else statusLoading.value = false
		}
	}

	async function loadValidate() {
		try {
			const res: any = await containerValidateAPI()
			if (isSucc(res.code)) validate.value = res.data
			else message.error(res.msg || t("containerRuntime.loadFailed"))
		} catch (error: any) {
			// 错误提示由请求拦截器统一处理
		}
	}

	function getDaemon(resetRetry = false) {
		if (resetRetry) daemonRetryCount.value = 0
		clearDaemonRetry()
		daemonLoading.value = true
		containerDaemonConfigAPI()
			.then((res: any) => {
				if (isSucc(res.code)) {
					daemon.value = res.data || {}
					daemonRetryCount.value = 0
				}
			})
			.catch((e: any) => {
				if (daemonRetryCount.value === 0) {
					message.warning("容器运行时配置暂时不可用，正在自动重试…")
				}
				if (daemonRetryCount.value < 8) {
					daemonRetryCount.value += 1
					daemonRetryTimer.value = window.setTimeout(() => getDaemon(false), 1500)
					return
				}
			})
			.finally(() => {
				daemonLoading.value = false
			})
	}

	function updateMirrorUrls(value: string, key: string) {
		if (key === "registryMirrors") {
			editingMirrorUrls.value = value
		}
		daemon.value[key] = value
			.split("\n")
			.map((url: string) => url.trim())
			.filter((url: string) => url)
	}

	function saveMirrorUrls() {
		editingMirrorUrls.value = Array.isArray(daemon.value?.registryMirrors)
			? daemon.value.registryMirrors.join("\n")
			: ""
		confirmationInput.value = ""
		showConfirmationModal.value = true
	}

	function openDrawer(drawerRef: any, key: string) {
		const needRestart = !!daemon.value?.capabilities?.daemonJson
		drawerRef.value?.open(daemon.value[key], key, { needRestart })
	}

	async function handleConfirmSaveChanges() {
		if (confirmationInput.value !== "立即重启") {
			message.error('输入错误，请输入 "立即重启"')
			return
		}
		mirrorSaveLoading.value = true
		try {
			const value = editingMirrorUrls.value
				.split("\n")
				.map(url => url.trim())
				.filter(url => url)
			const res = await updateDaemonUpdate("Mirrors", value.join(","))
			if (res && res.code === 0) {
				daemon.value.registryMirrors = value
				showConfirmationModal.value = false
				message.success("镜像加速配置已保存")
				getDaemon(true)
			} else {
				message.error(res?.msg || "保存失败")
			}
		} catch (error: any) {
			// 错误提示由请求拦截器统一处理
		} finally {
			mirrorSaveLoading.value = false
		}
	}

	async function fetchDaemonJsonFile() {
		if (!daemon.value?.capabilities?.daemonJson) return
		const res = await loadDaemonFile()
		if (res && res.code === 0 && typeof res.data === "string") {
			dockerConf.value = res.data
		}
	}

	function handleTabChange(tabName: string) {
		if (tabName === "advanced") {
			fetchDaemonJsonFile()
		}
	}

	function onSaveFile() {
		showRestartConfirm.value = true
	}

	async function handleConfirmRestart() {
		saveLoading.value = true
		try {
			const res = await updateDaemonByfile({ file: dockerConf.value })
			if (res && res.code === 0) {
				message.success("保存成功，Docker正在重启...")
				showRestartConfirm.value = false
				fetchDaemonJsonFile()
			} else {
				message.error(res.msg || "保存失败")
			}
		} catch {
			// 错误提示由请求拦截器统一处理
		} finally {
			saveLoading.value = false
		}
	}

	function onCgroupDriverChange(val: string, rebootCgroupRef: any) {
		cgroupInput.value = val
		rebootCgroupRef.value?.open({
			title: "cgroup driver 变更",
			input: "立即重启",
			msg: "切换 cgroup driver 后将会重启 Docker 服务。"
		})
	}

	async function handleCgroupConfirm(rebootCgroupRef: any) {
		rebootCgroupRef.value?.close()
		daemonLoading.value = true
		try {
			const res = await updateDaemonUpdate("Driver", cgroupInput.value)
			if (res && res.code === 0) {
				message.success("cgroup driver配置已保存，Docker正在重启...")
				getDaemon()
			} else {
				message.error(res.msg || "保存失败")
			}
		} catch {
			// 错误提示由请求拦截器统一处理
		} finally {
			daemonLoading.value = false
		}
	}

	async function onLogSwitchChange(val: boolean, logDrawerRef: any) {
		if (val) {
			logDrawerRef.value?.acceptParams({
				logMaxSize: daemon.value.logMaxSize,
				logMaxFile: daemon.value.logMaxFile
			})
			return
		}
		logPruneLoading.value = true
		try {
			const res = await updateDaemonUpdate("LogOption", "disable")
			if (res && res.code === 0) {
				message.success("日志切割已关闭")
				getDaemon()
			} else {
				message.error(res.msg || "操作失败")
			}
		} catch {
			// 错误提示由请求拦截器统一处理
		} finally {
			logPruneLoading.value = false
		}
	}

	function onIptablesChange(val: boolean, rebootIptablesRef: any) {
		iptablesTarget = val
		rebootIptablesRef.value?.open({
			title: "iptables 变更",
			input: "立即重启",
			msg: "变更 iptables 配置后需要重启 Docker 服务。"
		})
	}

	async function handleIptablesConfirm(rebootIptablesRef: any) {
		const value = iptablesTarget ? "enable" : "disable"
		rebootIptablesRef.value?.close()
		daemonLoading.value = true
		try {
			const res = await updateDaemonUpdate("IPtables", value)
			if (res && res.code === 0) {
				message.success("iptables 配置已保存，Docker正在重启...")
				getDaemon()
			} else {
				message.error(res.msg || "保存失败")
			}
		} catch {
			// 错误提示由请求拦截器统一处理
		} finally {
			daemonLoading.value = false
		}
	}

	function onLiveRestoreChange(val: boolean, rebootLiveRestoreRef: any) {
		liveRestoreTarget = val
		rebootLiveRestoreRef.value?.open({
			title: "Live restore 变更",
			input: "立即重启",
			msg: "变更 Live restore 配置后需要重启 Docker 服务。"
		})
	}

	async function handleLiveRestoreConfirm(rebootLiveRestoreRef: any) {
		const value = liveRestoreTarget ? "enable" : "disable"
		rebootLiveRestoreRef.value?.close()
		daemonLoading.value = true
		try {
			const res = await updateDaemonUpdate("LiveRestore", value)
			if (res && res.code === 0) {
				message.success("Live restore 配置已保存，Docker正在重启...")
				getDaemon()
			} else {
				message.error(res.msg || "保存失败")
			}
		} catch {
			// 错误提示由请求拦截器统一处理
		} finally {
			daemonLoading.value = false
		}
	}

	onMounted(async () => {
		await loadValidate()
		if (validate.value?.summary?.runtimeInstalled) getDaemon(true)
	})

	const {
		showRepairModal,
		repairSocketLoading,
		repairLingerLoading,
		autoRepairLoading,
		installLoading,
		installTask,
		repairHintType,
		openRepairModal,
		autoRepair,
		repairPodmanSocket,
		repairLinger,
		installRuntime
	} = useContainerRepair(message, validate, canAutoRepair, loadValidate, getDaemon)

	return {
		statusLoading,
		reloadLoading,
		daemon,
		daemonLoading,
		validate,
		currentRuntimeHost,
		runtimeBadgeText,
		runtimeDetailText,
		dockerOnly,
		canAutoRepair,
		repairHintType,
		showRepairModal,
		repairSocketLoading,
		repairLingerLoading,
		autoRepairLoading,
		installLoading,
		installTask,
		updateDockerStatus,
		openRepairModal,
		autoRepair,
		repairPodmanSocket,
		repairLinger,
		installRuntime,
		getDaemon,
		updateMirrorUrls,
		saveMirrorUrls,
		openDrawer,
		editingMirrorUrls,
		showConfirmationModal,
		confirmationInput,
		mirrorSaveLoading,
		handleConfirmSaveChanges,
		dockerConf,
		fetchDaemonJsonFile,
		handleTabChange,
		showRestartConfirm,
		saveLoading,
		onSaveFile,
		handleConfirmRestart,
		onCgroupDriverChange,
		handleCgroupConfirm,
		logPruneLoading,
		onLogSwitchChange,
		logSwitchValue,
		onIptablesChange,
		handleIptablesConfirm,
		onLiveRestoreChange,
		handleLiveRestoreConfirm,
		cgroupInput
	}
}
