import { ref, type Ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import type { Website } from "@/api/interface/website"
import { CreateSSLPushRule, DeleteSSLPushRule, SearchSSLPushRule, UpdateSSLPushRule } from "@/api/modules/ssl"
import { createDefaultPushRuleForm, type PushRuleFormState, type SSLRow } from "./sslHelpers"

interface UseSSLPushRulesOptions {
  currentPushSSL: Ref<SSLRow | null>
  submitting: Ref<boolean>
}

export const useSSLPushRules = (options: UseSSLPushRulesOptions) => {
  const message = useMessage()
  const dialog = useDialog()

  const pushRuleLoading = ref(false)
  const pushRuleData = ref<Website.SSLPushRule[]>([])
  const pushRuleForm = ref<PushRuleFormState>(createDefaultPushRuleForm())

  async function fetchPushRules() {
    if (!options.currentPushSSL.value?.id) return
    pushRuleLoading.value = true
    try {
      const res = await SearchSSLPushRule({
        page: 1,
        limit: 100,
        wheres: [{ column: "ssl_id", value: options.currentPushSSL.value.id, operator: "=" }]
      } as any)
      pushRuleData.value = Array.isArray(res.data?.items) ? res.data.items : []
    } finally {
      pushRuleLoading.value = false
    }
  }

  function resetPushRuleForm() {
    pushRuleForm.value = createDefaultPushRuleForm()
  }

  function editPushRule(row: Website.SSLPushRule) {
    pushRuleForm.value = {
      id: row.id,
      cloudAccountId: row.cloudAccountId,
      targetDomain: row.targetDomain
    }
  }

  function deletePushRule(row: Website.SSLPushRule) {
    dialog.warning({
      title: "确认删除部署规则？",
      content: "删除后，证书续签将不再自动推送到此目标资源。",
      onPositiveClick: async () => {
        await DeleteSSLPushRule({ id: row.id })
        message.success("删除成功")
        await fetchPushRules()
      }
    })
  }

  async function handleSavePushRule() {
    if (!pushRuleForm.value.cloudAccountId || !options.currentPushSSL.value?.id) {
      message.warning("请选择云账号")
      return
    }
    options.submitting.value = true
    try {
      if (pushRuleForm.value.id) {
        await UpdateSSLPushRule({
          id: pushRuleForm.value.id,
          cloudAccountId: pushRuleForm.value.cloudAccountId,
          targetDomain: pushRuleForm.value.targetDomain
        })
        message.success("部署规则已更新")
      } else {
        await CreateSSLPushRule({
          sslId: options.currentPushSSL.value.id,
          cloudAccountId: pushRuleForm.value.cloudAccountId,
          targetDomain: pushRuleForm.value.targetDomain
        })
        message.success("部署规则已新增")
      }
      resetPushRuleForm()
      await fetchPushRules()
    } catch (error: any) {
    } finally {
      options.submitting.value = false
    }
  }

  return {
    pushRuleLoading,
    pushRuleData,
    pushRuleForm,
    fetchPushRules,
    resetPushRuleForm,
    editPushRule,
    deletePushRule,
    handleSavePushRule
  }
}
