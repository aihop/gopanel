<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeSession } from "@/api/interface/code"
import type { CodeDeliveryJob, CodeGitStatus } from "@/api/interface/codeGit"
import { deliverMobileSession, getMobileGitStatus, saveMobileGitChanges } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import {
	codeDeliveryPhaseIcon,
	codeDeliveryPhaseType,
	useCodeDelivery,
	type CodeDeliveryPhase
} from "@/composables/useCodeDelivery"
import { mobileTaskDeliveryMessages } from "../mobileTaskDeliveryMessages"

const props = defineProps<{
	session: CodeSession
	delivery?: CodeDeliveryJob | null
	active: boolean
	revision?: string
	block?: boolean
}>()
const emit = defineEmits<{ updated: [] }>()
const { t } = useI18n({ messages: mobileTaskDeliveryMessages })
const dialog = useDialog()
const message = useMessage()
const loading = ref(false)
const queued = ref(false)
const gitStatus = ref<CodeGitStatus | null>(null)
const reviewStatus = ref<CodeGitStatus | null>(null)
const statusLoading = ref(false)
const statusError = ref(false)
const reviewReady = computed(() => Boolean(reviewStatus.value?.reviewReady && reviewStatus.value.reviewRevision))

// 交付状态判定与桌面端共用，这里只负责把 phase 映射成移动端文案。
const {
	available,
	phase,
	delivering,
	delivered,
	hasLocalChanges: hasChanges,
	canDeliverPending,
	pendingLocalSync
} = useCodeDelivery({
	job: computed(() => props.delivery ?? null),
	available: computed(() =>
		Boolean(props.session.worktreeBranch || props.session.isolationMode === "multi_worktree")
	),
	extraDelivering: computed(() => queued.value || props.session.status === "delivering"),
	extraDelivered: computed(() => props.session.status === "delivered"),
	hasLocalChanges: computed(() => (gitStatus.value?.files || 0) > 0)
})

const labelKeys: Record<CodeDeliveryPhase, string> = {
	queued: "mobile.deliveryQueuedShort",
	quality_check: "mobile.runningDeliveryQuality",
	running: "mobile.deliveringShort",
	needs_save: "mobile.saveChanges",
	deliverable: "mobile.deliverLatest",
	delivered: "mobile.deliveredShort",
	resume: "mobile.resumeDelivery",
	retry: "mobile.retryDelivery",
	idle: "mobile.deliverToMain"
}
const label = computed(() => {
	if (statusError.value) return t("mobile.retryGitStatus")
	if (statusLoading.value) return t("mobile.checkingChanges")
	if (!hasChanges.value && !reviewReady.value) return t("mobile.reviewTaskChangesBeforeDelivery")
	// 已交付态要区分主仓是否真的同步了，判定和桌面端共用 useCodeDelivery。
	if (phase.value === "delivered" && pendingLocalSync.value) return t("mobile.deliveredPendingSync")
	return t(labelKeys[phase.value])
})

async function loadGitStatus(silent = false) {
	if (!available.value || statusLoading.value) return
	statusLoading.value = true
	try {
		const [workspaceStatus, resultStatus] = await Promise.all([
			getMobileGitStatus(props.session.id, "workspace"),
			getMobileGitStatus(props.session.id, "result")
		])
		gitStatus.value = workspaceStatus
		reviewStatus.value = resultStatus
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
	if (!reviewReady.value) {
		void loadGitStatus()
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
				await deliverMobileSession(props.session.id, reviewStatus.value?.reviewRevision || "")
				queued.value = true
				message.success(t("mobile.deliveryQueuedSuccess"))
				emit("updated")
			} catch (error) {
				message.error(t("mobile.deliveryReviewChanged"))
				void loadGitStatus(true)
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
		reviewStatus.value = null
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
		:disabled="
			delivering ||
			(!statusError && !hasChanges && !reviewReady) ||
			(!statusError && delivered && !hasChanges && !canDeliverPending)
		"
		:block="block"
		:type="codeDeliveryPhaseType(phase)"
		class="!h-10 !rounded-xl"
		@click.stop="deliver"
	>
		<template #icon><Icon :name="codeDeliveryPhaseIcon(phase)" :size="14" /></template>
		{{ label }}
	</n-button>
</template>
