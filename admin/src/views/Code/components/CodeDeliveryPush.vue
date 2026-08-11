<script setup lang="ts">
import { computed, onMounted, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { getCodeDeliveryPushStatus, pushCodeSessionDelivery } from "@/api/modules/codeGit"
import { useCodeDeliveryPush } from "@/composables/useCodeDeliveryPush"
import { codeGitReviewMessages } from "../codeGitReviewMessages"
import CodeLocalSyncPending from "./CodeLocalSyncPending.vue"

const props = defineProps<{ sessionId: number; refreshKey: number }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const dialog = useDialog()
const message = useMessage()

const {
	result,
	loading,
	pushing,
	loadError,
	pushed,
	pendingCount,
	destinations,
	repositories,
	visible,
	loadStatus,
	runPush
} = useCodeDeliveryPush({
	sessionId: computed(() => props.sessionId),
	load: sessionId => getCodeDeliveryPushStatus(sessionId).then(response => response.data),
	push: sessionId => pushCodeSessionDelivery(sessionId).then(response => response.data)
})

const pushDelivery = () => {
	if (!result.value?.available || pushing.value) return
	dialog.warning({
		title: t("code.gitPushTitle"),
		content: t("code.gitPushConfirm", { count: pendingCount.value, destinations: destinations.value }),
		positiveText: t("code.gitPush"),
		negativeText: t("code.gitCancel"),
		onPositiveClick: async () => {
			try {
				await runPush()
				message.success(t("code.gitPushSuccess"))
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("code.gitPushFailed"))
			}
		}
	})
}


watch(() => [props.sessionId, props.refreshKey], () => void loadStatus())
onMounted(() => void loadStatus())
</script>

<template>
	<div v-if="visible" class="border-b border-slate-200 p-3">
		<n-spin :show="loading">
			<n-alert v-if="loadError" type="error" :title="t('code.gitPushStatusFailed')">
				<div class="space-y-2">
					<p>{{ loadError }}</p>
					<n-button size="tiny" @click="loadStatus">{{ t("code.gitRetry") }}</n-button>
				</div>
			</n-alert>
			<template v-else>
				<div v-if="pushed" class="text-xs text-emerald-600">
					{{ t("code.gitPushCompleted", { count: result?.repositories.length || 0 }) }}
				</div>
				<div v-else-if="result" class="space-y-2">
					<p class="text-xs text-slate-500">
						{{ result.available ? t("code.gitPushReady", { count: pendingCount }) : t("code.gitPushUnavailable") }}
					</p>
					<p v-if="result.available" class="truncate text-[11px] text-slate-400" :title="destinations">
						{{ destinations }}
					</p>
					<n-button
						v-if="result.available"
						size="small"
						type="warning"
						secondary
						block
						:loading="pushing"
						@click="pushDelivery"
					>
						{{ t("code.gitPush") }}
					</n-button>
				</div>

				<CodeLocalSyncPending :repositories="repositories" />
			</template>
		</n-spin>
	</div>
</template>
