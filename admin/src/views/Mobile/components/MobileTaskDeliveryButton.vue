<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeSession } from "@/api/interface/code"
import type { CodeDeliveryJob } from "@/api/interface/codeGit"
import { deliverMobileSession } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileTaskDeliveryMessages } from "../mobileTaskDeliveryMessages"

const props = defineProps<{ session: CodeSession; delivery?: CodeDeliveryJob | null }>()
const emit = defineEmits<{ queued: [] }>()
const { t } = useI18n({ messages: mobileTaskDeliveryMessages })
const dialog = useDialog()
const message = useMessage()
const loading = ref(false)
const queued = ref(false)

const available = computed(() =>
	Boolean(props.session.worktreeBranch || props.session.isolationMode === "multi_worktree")
)
const delivering = computed(
	() =>
		queued.value ||
		["queued", "running"].includes(props.delivery?.status || "") ||
		props.session.status === "delivering"
)
const delivered = computed(() => props.delivery?.status === "completed" || props.session.status === "delivered")
const label = computed(() =>
	delivered.value
		? t("mobile.deliveredShort")
		: delivering.value
			? t("mobile.deliveringShort")
			: t("mobile.deliverToMain")
)

function deliver() {
	if (loading.value || delivering.value || delivered.value) return
	dialog.warning({
		title: t("mobile.deliverToMain"),
		content: t("mobile.deliverToMainConfirm"),
		positiveText: t("mobile.confirmDeliveryToMain"),
		negativeText: t("mobile.cancel"),
		onPositiveClick: async () => {
			loading.value = true
			try {
				await deliverMobileSession(props.session.id)
				queued.value = true
				message.success(t("mobile.deliveryQueuedSuccess"))
				emit("queued")
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("mobile.deliveryQueueFailed"))
			} finally {
				loading.value = false
			}
		}
	})
}

watch(
	() => props.session.id,
	() => {
		queued.value = false
	}
)
</script>

<template>
	<n-button
		v-if="available"
		size="tiny"
		secondary
		:loading="loading"
		:disabled="delivering || delivered"
		:type="delivered ? 'success' : delivering ? 'info' : 'primary'"
		class="!h-10 !rounded-xl"
		@click.stop="deliver"
	>
		<template #icon>
			<Icon
				:name="
					delivered ? 'mdi:cloud-check-outline' : delivering ? 'mdi:cloud-sync-outline' : 'mdi:source-merge'
				"
				:size="14"
			/>
		</template>
		{{ label }}
	</n-button>
</template>
