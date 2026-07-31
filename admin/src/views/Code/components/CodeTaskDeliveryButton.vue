<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { getCodeDeliveryJob, mergeCodeSessionWorktree } from "@/api/modules/codeGit"
import { getCodeSession } from "@/api/modules/code"
import type { CodeDeliveryJob } from "@/api/interface/codeGit"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{ sessionId: number }>()
const emit = defineEmits<{ queued: [] }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const dialog = useDialog()
const message = useMessage()
const job = ref<CodeDeliveryJob | null>(null)
const available = ref(false)
const loading = ref(false)
let pollTimer: ReturnType<typeof setTimeout> | null = null

const active = computed(() => ["queued", "running"].includes(job.value?.status || ""))
const completed = computed(() => job.value?.status === "completed")
const label = computed(() => {
	if (job.value?.status === "queued") return t("code.deliverQueued")
	if (job.value?.status === "running") return t("code.deliveringToMain", { progress: job.value.progress })
	if (completed.value) return t("code.deliveredToMain")
	if (["failed", "conflict", "partial"].includes(job.value?.status || "")) return t("code.retryDeliveryToMain")
	return t("code.deliverToMain")
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
		available.value = Boolean(session.worktreeBranch || session.isolationMode === "multi_worktree")
		if (!available.value) {
			job.value = null
			return
		}
		job.value = (await getCodeDeliveryJob(props.sessionId)).data
	} catch (error) {
		if (!silent) message.error(error instanceof Error ? error.message : t("code.deliveryLoadFailed"))
	}
	if (active.value) pollTimer = setTimeout(() => void loadDelivery(true), 2000)
}

function deliver() {
	if (loading.value || active.value || completed.value) return
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
				message.error(error instanceof Error ? error.message : t("code.deliveryQueueFailed"))
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
		:disabled="active || completed"
		:type="completed ? 'success' : active ? 'info' : 'primary'"
		class="!rounded-xl"
		@click="deliver"
	>
		<template #icon>
			<Icon
				:name="completed ? 'mdi:cloud-check-outline' : active ? 'mdi:cloud-sync-outline' : 'mdi:source-merge'"
			/>
		</template>
		{{ label }}
	</n-button>
</template>
