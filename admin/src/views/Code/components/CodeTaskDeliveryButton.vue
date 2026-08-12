<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import {
	getCodeDeliveryJob,
	getCodeGitStatus,
	mergeCodeSessionWorktree,
	syncCodeSessionDeliveryLocal
} from "@/api/modules/codeGit"
import { getCodeSession } from "@/api/modules/code"
import type { CodeDeliveryJob } from "@/api/interface/codeGit"
import Icon from "@/components/common/Icon.vue"
import {
	codeDeliveryPhaseIcon,
	codeDeliveryPhaseType,
	useCodeDelivery,
	type CodeDeliveryPhase
} from "@/composables/useCodeDelivery"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = withDefaults(defineProps<{ sessionId: number; compact?: boolean }>(), { compact: false })
const emit = defineEmits<{ queued: []; settled: []; conflict: [] }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const dialog = useDialog()
const message = useMessage()
const job = ref<CodeDeliveryJob | null>(null)
const available = ref(false)
const sessionStatus = ref("")
const reviewReady = ref(false)
const reviewRevision = ref("")
const loading = ref(false)
const waitingForSettlement = ref(false)
let pollTimer: ReturnType<typeof setTimeout> | null = null

// 交付状态判定与移动端共用，这里只负责把 phase 映射成桌面文案。
const {
	phase,
	busy: active,
	completed,
	canDeliver,
	progress,
	pendingLocalSync
} = useCodeDelivery({
	job,
	available,
	pendingRequiresCompleted: true
})

const labelKeys: Record<CodeDeliveryPhase, string> = {
	queued: "code.deliverQueued",
	quality_check: "code.runningDeliveryQuality",
	running: "code.deliveringToMain",
	needs_save: "code.saveLatestBeforeDelivery",
	deliverable: "code.deliverLatestToMain",
	delivered: "code.deliveredToMain",
	resume: "code.resumeDeliveryToMain",
	retry: "code.retryDeliveryToMain",
	idle: "code.deliverToMain"
}
const mergeLocalOnly = computed(() => pendingLocalSync.value || (props.compact && completed.value && !canDeliver.value))
const label = computed(() => {
	if (props.compact && !active.value) return t("code.mergeTaskResult")
	if (!reviewReady.value && canDeliver.value) return t("code.reviewTaskChangesBeforeDelivery")
	if (phase.value === "running") return t("code.deliveringToMain", { progress: progress.value })
	// 已交付态要区分主仓是否真的同步了，判定在 useCodeDelivery 里，两端一致。
	if (phase.value === "delivered") {
		return t(pendingLocalSync.value ? "code.deliveredPendingLocalSync" : "code.deliveredToMain")
	}
	return t(labelKeys[phase.value])
})

function clearPoll() {
	if (pollTimer) clearTimeout(pollTimer)
	pollTimer = null
}

async function loadDelivery(silent = false) {
	clearPoll()
	try {
		const sessionResponse = await getCodeSession(props.sessionId)
		const session = sessionResponse.data.session
		sessionStatus.value = session.status
		const [deliveryResponse, reviewResponse] = await Promise.all([
			getCodeDeliveryJob(props.sessionId),
			getCodeGitStatus(props.sessionId, "result")
		])
		job.value = deliveryResponse.data
		const localFact = deliveryResponse.data?.facts?.find(fact => fact.key === "local")
		const hasPendingLocalSync = Boolean(
			deliveryResponse.data?.status === "completed" && localFact && localFact.status !== "completed"
		)
		available.value = Boolean(
			session.worktreeBranch ||
				session.isolationMode === "multi_worktree" ||
				hasPendingLocalSync ||
				(props.compact && deliveryResponse.data?.status === "completed" && reviewResponse.data.available)
		)
		reviewReady.value = reviewResponse.data.reviewReady
		reviewRevision.value = reviewResponse.data.reviewRevision || ""
		if (waitingForSettlement.value && !active.value) {
			waitingForSettlement.value = false
			emit("settled")
		}
	} catch (error) {
		if (!silent) message.error(t("code.deliveryLoadFailed"))
	}
	if (active.value) pollTimer = setTimeout(() => void loadDelivery(true), 2000)
	else if (completed.value && sessionStatus.value === "active")
		pollTimer = setTimeout(() => void loadDelivery(true), 5000)
}

function deliver() {
	if (loading.value || active.value) return
	if (mergeLocalOnly.value) {
		dialog.warning({
			title: t("code.mergeIntoLocalMain"),
			content: t("code.mergeIntoLocalMainConfirm"),
			positiveText: t("code.mergeTaskResult"),
			negativeText: t("code.cancel"),
			onPositiveClick: async () => {
				loading.value = true
				try {
					const response = await syncCodeSessionDeliveryLocal(props.sessionId)
					if (response.data.status === "conflict") {
						message.warning(t("code.localMainMergeConflict"))
						emit("conflict")
					} else if (response.data.status === "completed") {
						message.success(t("code.localMainMergeSuccess"))
						emit("settled")
					} else {
						message.warning(t("code.localMainMergeBlocked"))
					}
					void loadDelivery(true)
				} catch (error) {
					message.error(t("code.localMainMergeFailed"))
				} finally {
					loading.value = false
				}
			}
		})
		return
	}
	if (!canDeliver.value || !reviewReady.value || !reviewRevision.value) return
	dialog.warning({
		title: t("code.deliverToMain"),
		content: t("code.deliverToMainConfirm"),
		positiveText: t("code.confirmDeliveryToMain"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			loading.value = true
			try {
				job.value = (await mergeCodeSessionWorktree(props.sessionId, reviewRevision.value)).data
				waitingForSettlement.value = true
				message.success(t("code.deliveryQueuedSuccess"))
				emit("queued")
				void loadDelivery(true)
			} catch (error) {
				message.error(t("code.deliveryReviewChanged"))
				void loadDelivery(true)
			} finally {
				loading.value = false
			}
		}
	})
}

watch(
	() => props.sessionId,
	() => void loadDelivery(),
	{ immediate: true }
)
onBeforeUnmount(clearPoll)
</script>

<template>
	<n-button
		v-if="available"
		:size="compact ? 'tiny' : 'small'"
		secondary
		:loading="loading"
		:disabled="active || (!mergeLocalOnly && (!canDeliver || !reviewReady || !reviewRevision))"
		:type="codeDeliveryPhaseType(phase)"
		class="!rounded-xl"
		@click="deliver"
	>
		<template #icon><Icon :name="codeDeliveryPhaseIcon(phase)" /></template>
		{{ label }}
	</n-button>
</template>
