<script setup lang="ts">
import {
	nodeLocalControlIssueAPI,
	nodeLocalControlRevokeAPI,
	nodeLocalControlStatusAPI,
	nodeLocalTokenIssueAPI,
	nodeLocalTokenRevokeAPI,
	nodeLocalTokenStatusAPI
} from "@/api/modules/node"
import { t } from "@/i18n"
import { useDialog, useMessage } from "naive-ui"
import { onMounted, ref } from "vue"

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const enabled = ref(false)
const controlLoading = ref(false)
const controlEnabled = ref(false)
const issuedControlToken = ref("")
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

async function fetchControlStatus() {
	controlLoading.value = true
	try {
		const res = await nodeLocalControlStatusAPI()
		controlEnabled.value = !!res.data?.enabled
	} catch {
		// 失败提示由 axios 拦截器统一弹出
	} finally {
		controlLoading.value = false
	}
}

/** 控制令牌等价于本机管理员，签发和重签都必须让用户明确知道后果 */
function confirmIssueControl() {
	dialog.warning({
		title: t("commons.button.confirm"),
		content: controlEnabled.value ? t("node.local.controlReissueConfirm") : t("node.local.controlIssueConfirm"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			controlLoading.value = true
			try {
				const res = await nodeLocalControlIssueAPI()
				if (res.data?.accessToken) {
					issuedControlToken.value = res.data.accessToken
					controlEnabled.value = true
					message.success(t("node.local.issued"))
				}
			} catch {
				// 拦截器已提示
			} finally {
				controlLoading.value = false
			}
		}
	})
}

function confirmRevokeControl() {
	dialog.warning({
		title: t("commons.button.confirm"),
		content: t("node.local.controlRevokeConfirm"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			controlLoading.value = true
			try {
				await nodeLocalControlRevokeAPI()
				controlEnabled.value = false
				issuedControlToken.value = ""
				message.success(t("node.local.controlRevoked"))
			} catch {
				// 拦截器已提示
			} finally {
				controlLoading.value = false
			}
		}
	})
}

async function copyControlToken() {
	try {
		await navigator.clipboard.writeText(issuedControlToken.value)
		message.success(t("commons.msg.copySuccess"))
	} catch {
		message.error(t("node.copyFailed"))
	}
}

onMounted(() => {
	fetchStatus()
	fetchControlStatus()
})
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

			<n-divider class="!my-1" />

			<!-- 控制接入与只读接入分开：只读只能取摘要，控制等价于本机管理员 -->
			<div class="flex items-center gap-2 text-sm">
				<n-tag size="small" :type="controlEnabled ? 'warning' : 'default'" :bordered="false">
					{{ controlEnabled ? t("node.local.enabled") : t("node.local.disabled") }}
				</n-tag>
				<span class="font-medium">{{ t("node.local.controlTitle") }}</span>
				<span class="opacity-70">{{ t("node.local.controlDesc") }}</span>
			</div>

			<div v-if="issuedControlToken" class="flex flex-col gap-1">
				<n-alert type="warning" :show-icon="false">{{ t("node.local.saveTokenNow") }}</n-alert>
				<div class="flex gap-2">
					<n-input :value="issuedControlToken" readonly />
					<n-button @click="copyControlToken">{{ t("commons.button.copy") }}</n-button>
				</div>
			</div>

			<div class="flex gap-2">
				<n-button size="small" :loading="controlLoading" @click="confirmIssueControl">
					{{ controlEnabled ? t("node.local.reissue") : t("node.local.controlIssue") }}
				</n-button>
				<n-button v-if="controlEnabled" size="small" :loading="controlLoading" @click="confirmRevokeControl">
					{{ t("node.local.revoke") }}
				</n-button>
			</div>
		</div>
	</n-card>
</template>
