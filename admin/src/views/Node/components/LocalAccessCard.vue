<script setup lang="ts">
import { nodeLocalTokenIssueAPI, nodeLocalTokenRevokeAPI, nodeLocalTokenStatusAPI } from "@/api/modules/node"
import { t } from "@/i18n"
import { useDialog, useMessage } from "naive-ui"
import { onMounted, ref } from "vue"

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const enabled = ref(false)
/** 新签发的明文令牌，只在本次会话内存里存在，刷新页面就没了 */
const issuedToken = ref("")

async function fetchStatus() {
	loading.value = true
	try {
		const res = await nodeLocalTokenStatusAPI()
		enabled.value = !!res.data?.enabled
	} catch {
		// 失败提示由 axios 拦截器统一弹出
	} finally {
		loading.value = false
	}
}

async function issue() {
	loading.value = true
	try {
		const res = await nodeLocalTokenIssueAPI()
		if (res.data?.accessToken) {
			issuedToken.value = res.data.accessToken
			enabled.value = true
			message.success(t("node.local.issued"))
		}
	} catch {
		// 失败提示由 axios 拦截器统一弹出
	} finally {
		loading.value = false
	}
}

function confirmIssue() {
	if (!enabled.value) {
		issue()
		return
	}
	// 重新签发会让已经用旧令牌接入的主控立刻失效，必须让用户明确知道
	dialog.warning({
		title: t("commons.button.confirm"),
		content: t("node.local.reissueConfirm"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: issue
	})
}

function confirmRevoke() {
	dialog.warning({
		title: t("commons.button.confirm"),
		content: t("node.local.revokeConfirm"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			loading.value = true
			try {
				await nodeLocalTokenRevokeAPI()
				enabled.value = false
				issuedToken.value = ""
				message.success(t("node.local.revoked"))
			} catch {
				// 失败提示由 axios 拦截器统一弹出
			} finally {
				loading.value = false
			}
		}
	})
}

async function copyToken() {
	try {
		await navigator.clipboard.writeText(issuedToken.value)
		message.success(t("commons.msg.copySuccess"))
	} catch {
		message.error(t("node.copyFailed"))
	}
}

onMounted(fetchStatus)
</script>

<template>
	<n-card size="small" :title="t('node.local.title')">
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2 text-sm">
				<n-tag size="small" :type="enabled ? 'success' : 'default'" :bordered="false">
					{{ enabled ? t("node.local.enabled") : t("node.local.disabled") }}
				</n-tag>
				<span class="opacity-70">{{ t("node.local.desc") }}</span>
			</div>

			<div v-if="issuedToken" class="flex flex-col gap-1">
				<n-alert type="warning" :show-icon="false">
					{{ t("node.local.saveTokenNow") }}
				</n-alert>
				<div class="flex gap-2">
					<n-input :value="issuedToken" readonly />
					<n-button @click="copyToken">{{ t("commons.button.copy") }}</n-button>
				</div>
			</div>

			<div class="flex gap-2">
				<n-button size="small" :loading="loading" @click="confirmIssue">
					{{ enabled ? t("node.local.reissue") : t("node.local.issue") }}
				</n-button>
				<n-button v-if="enabled" size="small" :loading="loading" @click="confirmRevoke">
					{{ t("node.local.revoke") }}
				</n-button>
			</div>
		</div>
	</n-card>
</template>
