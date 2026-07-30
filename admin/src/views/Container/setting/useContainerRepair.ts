import { computed, onBeforeUnmount, ref, type Ref } from "vue"
import {
	containerRuntimeInstallStatusAPI,
	installContainerRuntimeAPI,
	repairPodmanSocketAPI,
	repairSystemdLingerAPI
} from "../../../api/modules/container"
import { containerRuntimeMessages } from "../../../i18n/locales/containerRuntime"
import { isSucc } from "../../../utils/is"
import { useI18n } from "vue-i18n"

export const useContainerRepair = (
	message: any,
	validate: Ref<any>,
	canAutoRepair: Ref<boolean>,
	loadValidate: () => Promise<void>,
	refreshDaemon: () => void
) => {
	const { t } = useI18n({ messages: containerRuntimeMessages })
	const showRepairModal = ref(false)
	const repairSocketLoading = ref(false)
	const repairLingerLoading = ref(false)
	const autoRepairLoading = ref(false)
	const installLoading = ref(false)
	const installTask = ref<any>(null)
	let installPollTimer: number | null = null

	const repairHintType = computed(() => {
		if (!validate.value) return "default"
		if (validate.value?.summary?.state === "notInstalled") return "primary"
		if (validate.value?.runtime?.serviceActive && !validate.value?.runtime?.apiReady) return "warning"
		return "default"
	})

	const repairPodmanSocket = async () => {
		repairSocketLoading.value = true
		try {
			const res: any = await repairPodmanSocketAPI()
			if (isSucc(res.code)) {
				message.success(t("containerRuntime.repairTriggered"))
				await loadValidate()
				refreshDaemon()
			} else {
				message.error(res.msg || t("containerRuntime.repairFailed"))
			}
		} catch (e: any) {
			message.error(e?.message || t("containerRuntime.repairFailed"))
		} finally {
			repairSocketLoading.value = false
		}
	}

	const repairLinger = async () => {
		repairLingerLoading.value = true
		try {
			const res: any = await repairSystemdLingerAPI()
			if (isSucc(res.code)) {
				message.success(t("containerRuntime.lingerEnabled"))
				await loadValidate()
			} else {
				message.error(res.msg || t("containerRuntime.operationFailed"))
			}
		} catch (e: any) {
			message.error(e?.message || t("containerRuntime.operationFailed"))
		} finally {
			repairLingerLoading.value = false
		}
	}

	const autoRepair = async () => {
		if (autoRepairLoading.value) return
		await loadValidate()
		if (
			!validate.value ||
			!canAutoRepair.value ||
			!validate.value?.gpc?.reachable ||
			validate.value?.runtime?.apiReady
		)
			return

		autoRepairLoading.value = true
		try {
			message.info(t("containerRuntime.autoRepairing"))
			const runtimeInfo: any = validate.value?.runtime || {}
			const isRootless = !!runtimeInfo.rootless || !!validate.value?.rootlessHost
			const notes = Array.isArray(validate.value?.notes) ? validate.value.notes.join(" ").toLowerCase() : ""
			const maybeRootless =
				typeof validate.value?.runtimeHost === "string" && validate.value.runtimeHost.includes("/run/user/")
			const needLinger =
				isRootless ||
				maybeRootless ||
				notes.includes("linger") ||
				notes.includes("user session") ||
				notes.includes("no medium found") ||
				notes.includes("cgroupv2")
			if (needLinger) {
				await repairLinger()
				await loadValidate()
				if (validate.value?.runtime?.apiReady) {
					message.success(t("containerRuntime.autoRepairSuccess"))
					return
				}
			}
			await repairPodmanSocket()
			await loadValidate()
			if (validate.value?.runtime?.apiReady) message.success(t("containerRuntime.autoRepairSuccess"))
			else message.warning(t("containerRuntime.autoRepairIncomplete"))
		} finally {
			autoRepairLoading.value = false
		}
	}

	const openRepairModal = async () => {
		await loadValidate()
		showRepairModal.value = true
	}

	const clearInstallPoll = () => {
		if (installPollTimer !== null) {
			window.clearTimeout(installPollTimer)
			installPollTimer = null
		}
	}

	const installFailureMessage = (task: any) =>
		task?.needsAction === "updateGpc"
			? t("containerRuntime.updateGpcAction")
			: task?.message || t("containerRuntime.installFailed")

	const pollInstallTask = async (taskId: string) => {
		clearInstallPoll()
		try {
			const res: any = await containerRuntimeInstallStatusAPI(taskId)
			if (!isSucc(res.code)) {
				installLoading.value = false
				message.error(res.msg || t("containerRuntime.installFailed"))
				return
			}
			installTask.value = res.data
			if (res.data?.status === "running") {
				installPollTimer = window.setTimeout(() => pollInstallTask(taskId), 1500)
				return
			}
			installLoading.value = false
			if (res.data?.status === "success") {
				const runtimeName = res.data.runtime === "docker" ? "Docker" : "Podman"
				message.success(t("containerRuntime.installSuccess", { runtime: runtimeName }))
				await loadValidate()
				refreshDaemon()
			} else {
				message.error(installFailureMessage(res.data))
			}
		} catch (error: any) {
			installLoading.value = false
			message.error(error?.message || t("containerRuntime.installFailed"))
		}
	}

	const installRuntime = async (runtime: "docker" | "podman") => {
		if (installLoading.value) return
		installLoading.value = true
		installTask.value = null
		try {
			const res: any = await installContainerRuntimeAPI(runtime)
			if (!isSucc(res.code)) {
				installLoading.value = false
				message.error(res.msg || t("containerRuntime.installFailed"))
				return
			}
			installTask.value = res.data
			await pollInstallTask(res.data.id)
		} catch (error: any) {
			installLoading.value = false
			message.error(error?.message || t("containerRuntime.installFailed"))
		}
	}

	onBeforeUnmount(clearInstallPoll)

	return {
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
	}
}
