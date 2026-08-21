import { ref, type Ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
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
	const { t } = useI18n()

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
			title: t("website.confirmDeletePushRule"),
			content: t("website.deletePushRuleHint"),
			onPositiveClick: async () => {
				await DeleteSSLPushRule({ id: row.id })
				message.success(t("commons.msg.deleteSuccess"))
				await fetchPushRules()
			}
		})
	}

	async function handleSavePushRule() {
		if (!pushRuleForm.value.cloudAccountId || !options.currentPushSSL.value?.id) {
			message.warning(t("website.selectCloudAccount"))
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
				message.success(t("website.pushRuleUpdated"))
			} else {
				await CreateSSLPushRule({
					sslId: options.currentPushSSL.value.id,
					cloudAccountId: pushRuleForm.value.cloudAccountId,
					targetDomain: pushRuleForm.value.targetDomain
				})
				message.success(t("website.pushRuleCreated"))
			}
			resetPushRuleForm()
			await fetchPushRules()
		} catch (error: any) {
			void 0
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
