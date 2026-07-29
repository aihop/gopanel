<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import {
	approveCodeInstruction,
	getCodeApprovals,
	getCodexRuntimeState,
	rejectCodeInstruction
} from "@/api/modules/code"
import type { CodeApproval, CodexRuntimeState } from "@/api/interface/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const props = defineProps<{ sessionId: number | null }>()
const emit = defineEmits<{ (event: "take-terminal"): void }>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()
const show = ref(false)
const loading = ref(false)
const loadError = ref(false)
const approvals = ref<CodeApproval[]>([])
const runtime = ref<CodexRuntimeState | null>(null)
const decidingId = ref<number | null>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

const nativeNeedsInput = computed(() => Boolean(runtime.value?.awaitingApproval))
const pendingCount = computed(() => approvals.value.length + (nativeNeedsInput.value ? 1 : 0))

const loadPending = async (notify = false) => {
	if (loading.value) return
	loading.value = true
	try {
		const [approvalResponse, runtimeResponse] = await Promise.all([
			getCodeApprovals(),
			props.sessionId ? getCodexRuntimeState(props.sessionId) : Promise.resolve(null)
		])
		approvals.value = approvalResponse.data.items || []
		runtime.value = runtimeResponse?.data || null
		loadError.value = false
	} catch (error) {
		loadError.value = true
		if (notify) message.error(error instanceof Error ? error.message : t("code.approvalLoadFailed"))
	} finally {
		loading.value = false
	}
}

const decide = async (approval: CodeApproval, approved: boolean) => {
	decidingId.value = approval.id
	try {
		if (approved) await approveCodeInstruction(approval.id)
		else await rejectCodeInstruction(approval.id)
		message.success(t(approved ? "code.approvalApproved" : "code.approvalRejected"))
		await loadPending()
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.approvalDecisionFailed"))
	} finally {
		decidingId.value = null
	}
}

const takeTerminal = () => {
	show.value = false
	emit("take-terminal")
}

watch(() => props.sessionId, () => void loadPending())
onMounted(() => {
	void loadPending()
	pollTimer = setInterval(() => void loadPending(), 5000)
})
onBeforeUnmount(() => {
	if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
	<n-badge :value="pendingCount" :max="99" :show="pendingCount > 0">
		<n-button size="small" :type="pendingCount > 0 ? 'warning' : 'default'" @click="show = true">
			{{ t("code.approvalCenter") }}
		</n-button>
	</n-badge>
	<n-drawer v-model:show="show" placement="right" style="width: min(440px, 100vw)">
		<n-drawer-content :title="t('code.approvalCenter')" closable>
			<n-spin :show="loading">
				<n-alert v-if="loadError" type="error" :title="t('code.approvalLoadFailed')" class="mb-4">
					<n-button size="small" @click="loadPending(true)">{{ t("code.retry") }}</n-button>
				</n-alert>
				<div class="space-y-4">
					<div v-if="nativeNeedsInput" class="rounded-2xl border border-amber-200 bg-amber-50 p-4">
						<n-tag type="warning" size="small" :bordered="false">{{ t("code.nativeApproval") }}</n-tag>
						<div class="mt-3 text-sm font-semibold text-slate-800">{{ t("code.codexApprovalHint") }}</div>
						<p v-if="runtime?.lastAssistantPreview" class="mt-2 text-xs leading-5 text-slate-600">
							{{ runtime.lastAssistantPreview }}
						</p>
						<n-button type="warning" block class="mt-4" @click="takeTerminal">
							{{ t("code.takeTerminalControl") }}
						</n-button>
					</div>
					<div v-for="approval in approvals" :key="approval.id" class="rounded-2xl border border-slate-200 bg-white p-4">
						<div class="flex items-center justify-between gap-3">
							<div class="text-sm font-semibold text-slate-800">{{ approval.title }}</div>
							<n-tag type="error" size="small" :bordered="false">{{ approval.riskLevel }}</n-tag>
						</div>
						<p class="mt-3 whitespace-pre-wrap break-words text-xs leading-5 text-slate-600">{{ approval.content }}</p>
						<div class="mt-4 grid grid-cols-2 gap-3">
							<n-button :disabled="decidingId === approval.id" @click="decide(approval, false)">
								{{ t("code.rejectApproval") }}
							</n-button>
							<n-button type="primary" :loading="decidingId === approval.id" @click="decide(approval, true)">
								{{ t("code.approveApproval") }}
							</n-button>
						</div>
					</div>
					<n-empty v-if="!loading && !loadError && pendingCount === 0" :description="t('code.noPendingApprovals')" />
				</div>
			</n-spin>
		</n-drawer-content>
	</n-drawer>
</template>
