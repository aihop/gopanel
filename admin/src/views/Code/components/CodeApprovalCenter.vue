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
		if (notify) message.error(t("code.approvalLoadFailed"))
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
		message.error(t("code.approvalDecisionFailed"))
	} finally {
		decidingId.value = null
	}
}

const takeTerminal = () => {
	show.value = false
	emit("take-terminal")
}

const updateShow = (value: boolean) => {
	show.value = value
	if (value) void loadPending()
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
	<n-popover
		:show="show"
		trigger="click"
		placement="bottom-end"
		:show-arrow="false"
		style="width: min(380px, calc(100vw - 24px))"
		@update:show="updateShow"
	>
		<template #trigger>
			<n-badge :value="pendingCount" :max="99" :show="pendingCount > 0">
				<n-button size="small" :type="pendingCount > 0 ? 'warning' : 'default'">
					{{ t("code.approvalCenter") }}
				</n-button>
			</n-badge>
		</template>
		<div class="p-1">
			<div class="mb-3 flex items-center justify-between gap-3">
				<div class="text-sm font-semibold text-[var(--n-text-color)]">{{ t("code.approvalCenter") }}</div>
				<n-tag v-if="pendingCount > 0" type="warning" size="small" :bordered="false">
					{{ pendingCount }}
				</n-tag>
			</div>
			<n-spin :show="loading">
				<n-alert v-if="loadError" type="error" :title="t('code.approvalLoadFailed')" class="mb-3">
					<n-button size="tiny" @click="loadPending(true)">{{ t("code.retry") }}</n-button>
				</n-alert>
				<n-scrollbar style="max-height: min(520px, calc(100vh - 160px))">
					<div class="space-y-3 pr-1">
						<div v-if="nativeNeedsInput" class="rounded-xl border border-amber-200 bg-amber-50 p-3 dark:border-amber-500/30 dark:bg-amber-500/10">
							<n-tag type="warning" size="small" :bordered="false">{{ t("code.nativeApproval") }}</n-tag>
							<div class="mt-2 text-sm font-semibold text-[var(--n-text-color)]">{{ t("code.codexApprovalHint") }}</div>
							<p v-if="runtime?.lastAssistantPreview" class="mt-2 text-xs leading-5 text-[var(--n-text-color-3)]">
								{{ runtime.lastAssistantPreview }}
							</p>
							<n-button type="warning" block size="small" class="mt-3" @click="takeTerminal">
								{{ t("code.takeTerminalControl") }}
							</n-button>
						</div>
						<div v-for="approval in approvals" :key="approval.id" class="rounded-xl border border-slate-200 bg-white p-3 dark:border-[var(--border-color)] dark:bg-white/5">
							<div class="flex items-center justify-between gap-3">
								<div class="text-sm font-semibold text-[var(--n-text-color)]">{{ approval.title }}</div>
								<n-tag type="error" size="small" :bordered="false">{{ approval.riskLevel }}</n-tag>
							</div>
							<p class="mt-2 whitespace-pre-wrap break-words text-xs leading-5 text-[var(--n-text-color-3)]">{{ approval.content }}</p>
							<div class="mt-3 flex justify-end gap-2">
								<n-button size="small" :disabled="decidingId === approval.id" @click="decide(approval, false)">
									{{ t("code.rejectApproval") }}
								</n-button>
								<n-button size="small" type="primary" :loading="decidingId === approval.id" @click="decide(approval, true)">
									{{ t("code.approveApproval") }}
								</n-button>
							</div>
						</div>
						<n-empty v-if="!loading && !loadError && pendingCount === 0" :description="t('code.noPendingApprovals')" />
					</div>
				</n-scrollbar>
			</n-spin>
		</div>
	</n-popover>
</template>
