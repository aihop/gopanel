<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { getCodeDeliveryJob, mergeCodeSessionWorktree } from "@/api/modules/codeGit"
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

const props = defineProps<{ sessionId: number }>()
const emit = defineEmits<{ queued: [] }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const dialog = useDialog()
const message = useMessage()
const job = ref<CodeDeliveryJob | null>(null)
const available = ref(false)
const sessionStatus = ref("")
const loading = ref(false)
let pollTimer: ReturnType<typeof setTimeout> | null = null

// 交付状态判定与移动端共用，这里只负责把 phase 映射成桌面文案。
const { phase, busy: active, completed, canDeliver, progress, pendingLocalSync } = useCodeDelivery({
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
	retry: "code.retryDeliveryToMain",
	idle: "code.deliverToMain"
}
const label = computed(() => {
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
		available.value = Boolean(session.worktreeBranch || session.isolationMode === "multi_worktree")
		if (!available.value) {
			job.value = null
			return
		}
		job.value = (await getCodeDeliveryJob(props.sessionId)).data
	} catch (error) {
		void 0
	}
	if (active.value) pollTimer = setTimeout(() => void loadDelivery(true), 2000)
	else if (completed.value && sessionStatus.value === "active")
		pollTimer = setTimeout(() => void loadDelivery(true), 5000)
}

function deliver() {
	if (loading.value || active.value || !canDeliver.value) return
	dialog.warning({
		title: t("code.deliverToMain"),
		content: t("code.deliverToMainConfirm"),
		positiveText: t("code.confirmDeliveryToMain"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			loading.value = true
			try {
				job.value = (await mergeCodeSessionWorktree(props.sessionId)).data
				message.success(t("code.deliveryQueuedSuccess"))
				emit("queued")
				void loadDelivery(true)
			} catch (error) {
				void 0
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
		size="small"
		secondary
		:loading="loading"
		:disabled="active || !canDeliver"
		:type="codeDeliveryPhaseType(phase)"
		class="!rounded-xl"
		@click="deliver"
	>
		<template #icon><Icon :name="codeDeliveryPhaseIcon(phase)" /></template>
		{{ label }}
	</n-button>
</template>
