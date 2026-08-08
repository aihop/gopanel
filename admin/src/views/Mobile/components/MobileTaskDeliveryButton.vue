<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeSession } from "@/api/interface/code"
import type { CodeDeliveryJob, CodeGitStatus } from "@/api/interface/codeGit"
import { deliverMobileSession, getMobileGitStatus, saveMobileGitChanges } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileTaskDeliveryMessages } from "../mobileTaskDeliveryMessages"

const props = defineProps<{
	session: CodeSession
	delivery?: CodeDeliveryJob | null
	active: boolean
	revision?: string
}>()
const emit = defineEmits<{ updated: [] }>()
const { t } = useI18n({ messages: mobileTaskDeliveryMessages })
const dialog = useDialog()
const message = useMessage()
const loading = ref(false)
const queued = ref(false)
const gitStatus = ref<CodeGitStatus | null>(null)
const statusLoading = ref(false)
const statusError = ref(false)

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
const hasPendingCommits = computed(() => props.delivery?.hasPendingCommits === true)
const hasUncommittedChanges = computed(() => props.delivery?.hasUncommittedChanges === true)
const hasChanges = computed(() => (gitStatus.value?.files || 0) > 0 || hasUncommittedChanges.value)
const canDeliverPending = computed(() => hasPendingCommits.value && !hasUncommittedChanges.value)
const label = computed(() => {
	if (statusError.value) return t("mobile.retryGitStatus")
	if (statusLoading.value) return t("mobile.checkingChanges")
	if (props.delivery?.status === "queued") return t("mobile.deliveryQueuedShort")
	if (delivering.value && props.delivery?.stage === "quality_check") return t("mobile.runningDeliveryQuality")
	if (delivering.value) return t("mobile.deliveringShort")
	if (hasChanges.value) return t("mobile.saveChanges")
	if (canDeliverPending.value) return t("mobile.deliverLatest")
	if (delivered.value) return t("mobile.deliveredShort")
	if (["failed", "conflict", "partial"].includes(props.delivery?.status || "")) {
		return t("mobile.retryDelivery")
	}
	return t("mobile.deliverToMain")
})

async function loadGitStatus(silent = false) {
	if (!available.value || statusLoading.value) return
	statusLoading.value = true
	try {
		gitStatus.value = await getMobileGitStatus(props.session.id)
		statusError.value = false
	} catch (error) {
		statusError.value = true
	} finally {
		statusLoading.value = false
	}
}

async function saveChanges() {
	loading.value = true
	try {
		await saveMobileGitChanges(props.session.id)
		message.success(t("mobile.gitSaveSuccess"))
		await loadGitStatus(true)
		emit("updated")
	} catch (error) {
		void 0
	} finally {
		loading.value = false
	}
}

function deliver() {
	if (statusError.value) {
		void loadGitStatus()
		return
	}
	if (
		loading.value ||
		statusLoading.value ||
		delivering.value ||
		(delivered.value && !hasChanges.value && !canDeliverPending.value)
	)
		return
	if (hasChanges.value) {
		void saveChanges()
		return
	}
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
				emit("updated")
			} catch (error) {
				void 0
			} finally {
				loading.value = false
			}
		}
	})
}

watch(
	[() => props.session.id, () => props.active],
	([, active]) => {
		queued.value = false
		gitStatus.value = null
		statusError.value = false
		if (active) void loadGitStatus()
	},
	{ immediate: true }
)
watch(
	() => props.revision,
	() => {
		if (props.active) void loadGitStatus(true)
	}
)
watch(
	() => props.delivery?.status,
	status => {
		if (status) queued.value = false
	}
)
</script>

<template>
	<n-button
		v-if="available"
		size="tiny"
		secondary
		:loading="loading || statusLoading"
		:disabled="delivering || (!statusError && delivered && !hasChanges && !canDeliverPending)"
		:type="delivered && !hasChanges && !canDeliverPending ? 'success' : delivering ? 'info' : 'primary'"
		class="!h-10 !rounded-xl"
		@click.stop="deliver"
	>
		<template #icon>
			<Icon
				:name="
					hasChanges
						? 'mdi:content-save-outline'
						: delivered && !canDeliverPending && !hasChanges
							? 'mdi:cloud-check-outline'
							: delivering
								? 'mdi:cloud-sync-outline'
								: 'mdi:source-merge'
				"
				:size="14"
			/>
		</template>
		{{ label }}
	</n-button>
</template>
