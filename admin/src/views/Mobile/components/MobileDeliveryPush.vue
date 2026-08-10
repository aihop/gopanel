<script setup lang="ts">
import { computed, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { getMobileDeliveryPushStatus, pushMobileSessionDelivery } from "@/api/modules/mobile"
import { useCodeDeliveryPush } from "@/composables/useCodeDeliveryPush"
import Icon from "@/components/common/Icon.vue"
import { mobileTaskDeliveryMessages } from "../mobileTaskDeliveryMessages"

const props = defineProps<{ sessionId: number | null; active: boolean; refreshKey?: string }>()
const { t } = useI18n({ messages: mobileTaskDeliveryMessages })
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
	localSyncPending,
	localSyncCommands,
	visible,
	loadStatus,
	runPush
} = useCodeDeliveryPush({
	sessionId: computed(() => props.sessionId),
	load: getMobileDeliveryPushStatus,
	push: pushMobileSessionDelivery
})

const pushDelivery = () => {
	if (!result.value?.available || pushing.value) return
	dialog.warning({
		title: t("mobile.gitPushTitle"),
		content: t("mobile.gitPushConfirm", { count: pendingCount.value, destinations: destinations.value }),
		positiveText: t("mobile.gitPush"),
		negativeText: t("mobile.cancel"),
		onPositiveClick: async () => {
			try {
				await runPush()
				message.success(t("mobile.gitPushSuccess"))
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("mobile.gitPushFailed"))
			}
		}
	})
}

const copyLocalSyncCommands = async () => {
	if (!localSyncCommands.value) return
	try {
		await navigator.clipboard.writeText(localSyncCommands.value)
		message.success(t("mobile.gitLocalSyncCopied"))
	} catch (error) {
		message.error(t("mobile.gitPushStatusFailed"))
	}
}

watch(
	[() => props.sessionId, () => props.active, () => props.refreshKey],
	([, active]) => {
		if (active) void loadStatus()
	},
	{ immediate: true }
)
</script>

<template>
	<div v-if="visible" class="mt-3">
		<n-spin :show="loading">
			<n-alert v-if="loadError" type="error" :show-icon="false">
				<div class="space-y-2">
					<p class="text-xs">{{ loadError }}</p>
					<n-button size="tiny" @click="loadStatus">{{ t("mobile.gitRetry") }}</n-button>
				</div>
			</n-alert>
			<template v-else>
				<div v-if="pushed" class="flex items-center gap-1.5 text-xs text-emerald-600">
					<Icon name="mdi:cloud-check-outline" :size="14" />
					{{ t("mobile.gitPushCompleted", { count: result?.repositories.length || 0 }) }}
				</div>
				<div v-else-if="result?.available" class="space-y-2">
					<p class="truncate text-[11px] text-slate-400" :title="destinations">{{ destinations }}</p>
					<n-button
						size="small"
						type="warning"
						secondary
						block
						class="!h-10 !rounded-xl"
						:loading="pushing"
						@click.stop="pushDelivery"
					>
						<template #icon><Icon name="mdi:cloud-upload-outline" :size="14" /></template>
						{{ t("mobile.gitPushPending", { count: pendingCount }) }}
					</n-button>
				</div>

				<!-- 本地主仓未同步不影响推送，手机上只提示，命令交给桌面执行。 -->
				<n-alert v-if="localSyncPending.length" type="info" :show-icon="false" class="mt-2">
					<p class="text-xs font-medium">
						{{ t("mobile.gitLocalSyncPending", { count: localSyncPending.length }) }}
					</p>
					<p class="mt-1 text-[11px] leading-5 text-slate-500">{{ t("mobile.gitLocalSyncHint") }}</p>
					<n-button v-if="localSyncCommands" size="tiny" secondary class="mt-2" @click="copyLocalSyncCommands">
						{{ t("mobile.gitLocalSyncCopy") }}
					</n-button>
				</n-alert>
			</template>
		</n-spin>
	</div>
</template>
