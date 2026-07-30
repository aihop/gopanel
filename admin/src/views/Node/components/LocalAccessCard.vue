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
import { computed, onMounted, ref } from "vue"

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

/**
 * 两块接入能力结构完全一样（标题/说明/状态/令牌/操作），用一个数组驱动同一套模板，
 * 避免两边样式各写一遍以后慢慢长歪。
 * tone 区分语义：只读是安全的，控制等价于本机管理员，开启后要更醒目。
 */
const tiles = computed(() => [
	{
		key: "readonly",
		tone: "safe" as const,
		title: t("node.local.title"),
		desc: t("node.local.desc"),
		enabled: enabled.value,
		loading: loading.value,
		token: issuedToken.value,
		issueLabel: enabled.value ? t("node.local.reissue") : t("node.local.issue"),
		issue: confirmIssue,
		revoke: confirmRevoke,
		copy: copyToken
	},
	{
		key: "control",
		tone: "warn" as const,
		title: t("node.local.controlTitle"),
		desc: t("node.local.controlDesc"),
		enabled: controlEnabled.value,
		loading: controlLoading.value,
		token: issuedControlToken.value,
		issueLabel: controlEnabled.value ? t("node.local.reissue") : t("node.local.controlIssue"),
		issue: confirmIssueControl,
		revoke: confirmRevokeControl,
		copy: copyControlToken
	}
])

onMounted(() => {
	fetchStatus()
	fetchControlStatus()
})
</script>

<template>
	<!--
		只读接入 / 控制接入是并列的两种能力，做成一行两块。
		每块内部压成两行：标题+状态一行，说明+按钮一行——说明和按钮各占一行时
		单块要 130px，并排后 80px 出头，两块加起来省下近一半竖向空间。
		窄屏（< md）自动堆成一列；说明文字用 min-w-0 + 省略号收缩，
		按钮永远不会被挤出去，也不会撑出横向滚动条。
	-->
	<div class="local-access grid gap-4 md:grid-cols-2">
		<section v-for="tile of tiles" :key="tile.key" class="access-tile bg-base-accent border-base-accent">
			<header class="tile-head">
				<h3 class="tile-title">{{ tile.title }}</h3>
				<span class="tile-state" :class="tile.enabled ? `is-on is-${tile.tone}` : 'is-off'">
					<i class="state-dot" />
					{{ tile.enabled ? t("node.local.enabled") : t("node.local.disabled") }}
				</span>
			</header>

			<!-- 明文令牌只在签发后的这次会话里出现，位置固定在操作区上方 -->
			<div v-if="tile.token" class="token-box">
				<div class="token-hint">{{ t("node.local.saveTokenNow") }}</div>
				<div class="flex flex-wrap items-center gap-2">
					<n-input :value="tile.token" readonly size="small" class="token-input min-w-0 grow" />
					<n-button size="small" secondary @click="tile.copy()">
						{{ t("commons.button.copy") }}
					</n-button>
				</div>
			</div>

			<!-- 说明和按钮并排：这两样各占一行是最浪费竖向空间的地方 -->
			<footer class="tile-foot">
				<p class="tile-desc" :title="tile.desc">{{ tile.desc }}</p>
				<div class="tile-actions">
					<n-button size="small" secondary :loading="tile.loading" @click="tile.issue()">
						{{ tile.issueLabel }}
					</n-button>
					<n-button v-if="tile.enabled" size="small" quaternary :loading="tile.loading" @click="tile.revoke()">
						{{ t("node.local.revoke") }}
					</n-button>
				</div>
			</footer>
		</section>
	</div>
</template>

<style lang="scss" scoped>
.access-tile {
	display: flex;
	flex-direction: column;
	gap: 10px;
	min-width: 0;
	padding: 14px 18px;
	border-radius: 16px;
	transition: border-color 0.2s var(--bezier-ease);
}

.tile-head {
	display: flex;
	align-items: center;
	gap: 10px;
}

.tile-foot {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: 8px;
}

.tile-title {
	margin: 0;
	color: var(--fg-default-color);
	font-size: 15px;
	font-weight: 500;
	line-height: 22px;
}

// 说明是辅助信息，让它收缩让位给按钮；完整文案挂在 title 属性上
.tile-desc {
	flex: 1 1 220px;
	min-width: 0;
	overflow: hidden;
	margin: 0;
	color: var(--fg-secondary-color);
	font-size: 12px;
	line-height: 18px;
	text-overflow: ellipsis;
	white-space: nowrap;
}

// 扁平状态：不用填充色标签，一个小圆点 + 文字，安静但看得见
.tile-state {
	display: inline-flex;
	flex-shrink: 0;
	align-items: center;
	gap: 6px;
	font-size: 12px;
	line-height: 18px;
	white-space: nowrap;

	.state-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background-color: currentcolor;
	}

	&.is-off {
		color: var(--fg-secondary-color);
		opacity: 0.65;
	}

	&.is-on.is-safe {
		color: var(--success-color);
	}

	// 控制接入等价于本机管理员，开启后用警告色提示，别做成和只读一样安静
	&.is-on.is-warn {
		color: var(--warning-color);
	}
}

.token-box {
	display: flex;
	flex-direction: column;
	gap: 8px;
	padding: 12px 14px;
	border-radius: 14px;
	background-color: rgba(var(--warning-color-rgb) / 0.1);
}

.token-hint {
	color: var(--warning-color);
	font-size: 12px;
	line-height: 18px;
}

.tile-actions {
	display: flex;
	flex-shrink: 0;
	flex-wrap: wrap;
	gap: 8px;
}
</style>
