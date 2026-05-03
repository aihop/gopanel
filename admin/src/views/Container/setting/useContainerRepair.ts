import { computed, ref, type Ref } from "vue"
import { repairPodmanSocketAPI, repairSystemdLingerAPI } from "../../../api/modules/container"
import { isSucc } from "../../../utils/is"

export const useContainerRepair = (
  message: any,
  validate: Ref<any>,
  canAutoRepair: Ref<boolean>,
  loadValidate: () => Promise<void>,
  refreshDaemon: () => void
) => {
  const showRepairModal = ref(false)
  const repairSocketLoading = ref(false)
  const repairLingerLoading = ref(false)
  const autoRepairLoading = ref(false)

  const repairHintType = computed(() => {
    if (!validate.value) return "default"
    if (validate.value?.runtime?.serviceActive && !validate.value?.runtime?.apiReady) return "warning"
    return "default"
  })

  const repairPodmanSocket = async () => {
    repairSocketLoading.value = true
    try {
      const res: any = await repairPodmanSocketAPI()
      if (isSucc(res.code)) {
        message.success("已触发修复，正在刷新状态…")
        await loadValidate()
        refreshDaemon()
      } else {
        message.error(res.msg || "修复失败")
      }
    } catch (e: any) {
      message.error(e?.message || "修复失败")
    } finally {
      repairSocketLoading.value = false
    }
  }

  const repairLinger = async () => {
    repairLingerLoading.value = true
    try {
      const res: any = await repairSystemdLingerAPI()
      if (isSucc(res.code)) {
        message.success("已启用 linger")
        await loadValidate()
      } else {
        message.error(res.msg || "操作失败")
      }
    } catch (e: any) {
      message.error(e?.message || "操作失败")
    } finally {
      repairLingerLoading.value = false
    }
  }

  const autoRepair = async () => {
    if (autoRepairLoading.value) return
    await loadValidate()
    if (!validate.value || !canAutoRepair.value || !validate.value?.gpc?.reachable || validate.value?.runtime?.apiReady) return

    autoRepairLoading.value = true
    try {
      message.info("正在尝试自动修复…")
      const runtimeInfo: any = validate.value?.runtime || {}
      const isRootless = !!runtimeInfo.rootless || !!validate.value?.rootlessHost
      const notes = Array.isArray(validate.value?.notes) ? validate.value.notes.join(" ").toLowerCase() : ""
      const maybeRootless = typeof validate.value?.runtimeHost === "string" && validate.value.runtimeHost.includes("/run/user/")
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
          message.success("自动修复成功")
          return
        }
      }
      await repairPodmanSocket()
      await loadValidate()
      if (validate.value?.runtime?.apiReady) message.success("自动修复成功")
      else message.warning("自动修复未完全成功，请查看提示信息")
    } finally {
      autoRepairLoading.value = false
    }
  }

  const openRepairModal = async () => {
    await loadValidate()
    showRepairModal.value = true
    await autoRepair()
  }

  return {
    showRepairModal,
    repairSocketLoading,
    repairLingerLoading,
    autoRepairLoading,
    repairHintType,
    openRepairModal,
    autoRepair,
    repairPodmanSocket,
    repairLinger
  }
}
