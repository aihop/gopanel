import { nextTick, reactive, ref } from "vue"
import { ReadByLine } from "@/api/modules/file"
import { testCompose, upCompose } from "@/api/modules/container"
import { MsgError, MsgSuccess } from "@/utils/message"
import { initialComposeFormState } from "./composeTypes"

export const useComposeCreateFlow = () => {
  const showCreateModal = ref(false)
  const composeForm = reactive({ ...initialComposeFormState })
  const envPlaceholder = "一行一个, 例:\nkey1=value1\nkey2=value2"
  const activeTab = ref("compose-definition")
  const logContent = ref("")
  const logLoading = ref(false)
  let logTimer: any = null

  const openCreateModal = () => {
    Object.assign(composeForm, initialComposeFormState)
    composeForm.composeContent = ""
    composeForm.selectedTemplateId = null
    activeTab.value = "compose-definition"
    logContent.value = ""
    showCreateModal.value = true
  }

  const stopLogPolling = () => {
    if (logTimer) {
      clearInterval(logTimer)
      logTimer = null
    }
  }

  const handleConfirmCreate = async () => {
    if (!composeForm.projectName || composeForm.projectName.trim() === "") {
      MsgError("请填写文件夹名称")
      return false
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(composeForm.projectName)) {
      MsgError("文件夹名称错误")
      return false
    }
    const envArr = (composeForm.envContent || "")
      .split("\n")
      .map(line => line.trim())
      .filter(Boolean)
    const envStr = envArr.join("\n")
    const params: any = {
      name: composeForm.projectName,
      from: composeForm.source === "editor" ? "edit" : composeForm.source,
      path: composeForm.source === "path" ? composeForm.pathValue : "",
      file: composeForm.composeContent,
      env: envArr,
      envStr,
      envFileContent: composeForm.envContent
    }
    if (composeForm.selectedTemplateId) {
      params.template = composeForm.selectedTemplateId
    }
    try {
      const testRes = await testCompose(params)
      if (testRes.code !== 200 && testRes.code !== 0) {
        MsgError(testRes.message || "配置测试失败")
        return false
      }
      const createRes = await upCompose(params)
      if (createRes.code !== 200 && createRes.code !== 0) {
        MsgError(createRes.message || "创建失败")
        return false
      }
      nextTick(() => {
        activeTab.value = "compose-logs"
      })
      const logFile = createRes.data
      logContent.value = ""
      logLoading.value = true
      stopLogPolling()
      const fetchLog = async () => {
        try {
          const res = await ReadByLine({ type: "compose-create", name: logFile, page: 1, limit: 1000 })
          if (res.data && res.data.lines) {
            logContent.value = res.data.lines.join("\n")
            const lastLine = res.data.lines[res.data.lines.length - 1] || ""
            if (lastLine.includes("successful")) {
              stopLogPolling()
            }
          }
        } catch {}
      }
      await fetchLog()
      logTimer = setInterval(fetchLog, 2000)
      MsgSuccess("创建任务已提交，正在获取日志...")
      return true
    } catch (e: any) {
      return false
    }
  }

  return {
    showCreateModal,
    composeForm,
    envPlaceholder,
    activeTab,
    logContent,
    logLoading,
    openCreateModal,
    handleConfirmCreate,
    stopLogPolling
  }
}
