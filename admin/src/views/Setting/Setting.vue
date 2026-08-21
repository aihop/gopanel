<template>
	<n-tabs type="line" animated class="mt-2">
		<template #suffix>
			<n-space v-if="!isSubAdmin" size="small" align="center">
				<n-button ghost class="!rounded-[16px]" :loading="restartingPanel" @click="handleRestart('panel')">
					{{ t("setting.restartPanel") }}
				</n-button>
				<n-button ghost class="!rounded-[16px]" :loading="restartingServer" @click="handleRestart('server')">
					{{ t("setting.restartServer") }}
				</n-button>
			</n-space>
		</template>
		<n-tab-pane name="system" :tab="t('setting.panelSettings')">
			<conf />
		</n-tab-pane>
		<n-tab-pane v-if="!isSubAdmin" name="subadmin" :tab="t('setting.adminSettings')">
			<sub-admin />
		</n-tab-pane>
		<n-tab-pane name="cloud" :tab="t('setting.cloudAccountAuth')">
			<cloud-account />
		</n-tab-pane>
		<n-tab-pane name="aiAccount" :tab="t('setting.aiAccount')">
			<AIAccount />
		</n-tab-pane>
		<n-tab-pane v-if="!isSubAdmin" name="notify" :tab="t('setting.mailNotify')">
			<notify />
		</n-tab-pane>
		<n-tab-pane name="update" :tab="t('setting.versionUpdate')">
			<update />
		</n-tab-pane>
	</n-tabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane, NButton, NSpace, useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import Update from "./components/Update.vue"
import Conf from "./components/Conf.vue"
import CloudAccount from "./components/CloudAccount.vue"
import AIAccount from "./components/AIAccount.vue"
import SubAdmin from "./components/SubAdmin.vue"
import Notify from "./components/Notify.vue"
import { useAuthStore } from "@/store/auth"
import { computed, ref } from "vue"
import { settingSystemRestart } from "@/api/modules/setting"

const authStore = useAuthStore()
const dialog = useDialog()
const message = useMessage()
const { t } = useI18n()
const isSubAdmin = computed(() => authStore.user?.role === "SUB_ADMIN")
const restartingPanel = ref(false)
const restartingServer = ref(false)

const highlights = [
	{
		label: "Update",
		value: t("setting.versionUpdate"),
		desc: t("setting.versionUpdateDesc")
	},
	{
		label: "Access",
		value: t("setting.entrance"),
		desc: t("setting.entranceDesc")
	},
	{
		label: "Storage",
		value: t("setting.dirManagement"),
		desc: t("setting.dirManagementDesc")
	}
]

const handleRestart = (operation: "panel" | "server") => {
	const isPanel = operation === "panel"
	dialog.warning({
		title: isPanel ? t("setting.restartPanel") : t("setting.restartServer"),
		content: isPanel ? t("setting.restartPanelConfirm") : t("setting.restartServerConfirm"),
		positiveText: isPanel ? t("setting.restartPanelNow") : t("setting.restartServerNow"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			if (isPanel) {
				restartingPanel.value = true
			} else {
				restartingServer.value = true
			}
			try {
				const res = await settingSystemRestart(operation)
				if (res.code === 0) {
					message.success(isPanel ? t("setting.panelRestartSent") : t("setting.serverRestartSent"))
				} else {
					message.error(
						res.msg || (isPanel ? t("setting.restartPanelFailed") : t("setting.restartServerFailed"))
					)
				}
			} catch (_e) {
				void 0
			} finally {
				if (isPanel) {
					restartingPanel.value = false
				} else {
					restartingServer.value = false
				}
			}
		}
	})
}
</script>
