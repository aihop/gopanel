import { ref, type Ref } from "vue"
import { NInput } from "naive-ui"
import {
  DaemonConfigAdd,
  DaemonConfigDelete,
  DaemonConfigUpdate,
  DaemonProcessReload,
  DaemonProcessStart,
  DaemonProcessStop,
  DaemonReload,
  DaemonStart,
  DaemonStop
} from "@/api/modules/daemon"
import { createDeleteConfirmContent, createStopConfirmContent } from "./daemonTableColumns"

export const useDaemonActions = (
  dialog: any,
  loading: Ref<boolean>,
  refreshAll: () => Promise<void>,
  getData: () => Promise<void> | void,
  daemonPostRef: { value: { close: () => void } | null }
) => {
  const stopConfirmInput = ref("")
  const deleteConfirmInput = ref("")

  const handleDaemonStart = async () => {
    loading.value = true
    await DaemonReload()
    await DaemonStart()
    getData()
  }

  const handleDaemonStop = async () => {
    stopConfirmInput.value = ""
    dialog.warning({
      title: "确认全部停止",
      content: () => createStopConfirmContent(stopConfirmInput),
      positiveText: "确定",
      negativeText: "取消",
      onPositiveClick: async () => {
        if (stopConfirmInput.value !== "全部停止") return false
        loading.value = true
        await DaemonStop()
        getData()
      }
    })
  }

  const handleProcessStart = async (name: string) => {
    loading.value = true
    await DaemonProcessStart(name)
    getData()
  }

  const handleProcessStop = async (name: string) => {
    loading.value = true
    await DaemonProcessStop(name)
    getData()
  }

  const handleProcessReload = async (name: string) => {
    loading.value = true
    await DaemonProcessReload(name)
    getData()
  }

  const handleProcessDelete = async (name: string) => {
    deleteConfirmInput.value = ""
    dialog.info({
      title: "确认删除",
      content: () => createDeleteConfirmContent(deleteConfirmInput),
      positiveText: "确定",
      negativeText: "取消",
      onPositiveClick: async () => {
        if (deleteConfirmInput.value !== "立即删除") return false
        loading.value = true
        await DaemonProcessStop(name)
        await DaemonConfigDelete({ names: [name] })
        await DaemonReload()
        getData()
      }
    })
  }

  const postConfirm = async (payload: { data: any; isUpdate: boolean }, submitLoading: Ref<boolean>) => {
    submitLoading.value = true
    if (payload.isUpdate) {
      await DaemonConfigUpdate(payload.data)
    } else {
      await DaemonConfigAdd(payload.data)
    }
    await DaemonReload()
    getData()
    daemonPostRef.value?.close()
  }

  return {
    handleDaemonStart,
    handleDaemonStop,
    handleProcessStart,
    handleProcessStop,
    handleProcessReload,
    handleProcessDelete,
    postConfirm
  }
}
