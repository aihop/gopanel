<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import CodeConflictManualMerge from "./CodeConflictManualMerge.vue"
import CodeConflictResolverDrawer from "./CodeConflictResolverDrawer.vue"
import CodeDeliveryPush from "./CodeDeliveryPush.vue"
import CodeDeliveryFacts from "./CodeDeliveryFacts.vue"
import CodeGitDiffViewer from "./CodeGitDiffViewer.vue"
import CodeGitHistory from "./CodeGitHistory.vue"
import CodeGitReviewHeader from "./CodeGitReviewHeader.vue"
import CodeGitRepositoryChanges from "./CodeGitRepositoryChanges.vue"
import CodeLocalSyncPending from "./CodeLocalSyncPending.vue"
import CodeSessionRepositorySync from "./CodeSessionRepositorySync.vue"
import {
	getCodeDeliveryJob,
	getCodeGitDiff,
	getCodeGitStatus,
	saveCodeGitChanges,
	updateCodeGitStage
} from "@/api/modules/codeGit"
import { getCodeSession } from "@/api/modules/code"
import type { CodeDeliveryJob, CodeGitHistorySelection, CodeGitStatus } from "@/api/interface/codeGit"
import { codeGitReviewEntries, codeGitReviewTotals, type CodeGitReviewEntry } from "../codeGitReviewEntries"
import { codeGitReviewMessages } from "../codeGitReviewMessages"
import type { CodeGitReviewView } from "../codeGitReviewView"

const props = defineProps<{ sessionId: number | null; active: boolean }>()
const emit = defineEmits<{ (event: "open-file", file: { path: string; extension: string }): void }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const message = useMessage()
const view = ref<CodeGitReviewView>("changes")
const status = ref<CodeGitStatus | null>(null)
const historySelection = ref<CodeGitHistorySelection | null>(null)
const historyRefreshKey = ref(0)
const loading = ref(false)
const refreshing = ref(false)
const loadError = ref("")
const selectedKey = ref("")
const diffContent = ref("")
const diffTruncated = ref(false)
const diffLoading = ref(false)
const stagingKey = ref("")
const worktreeBranch = ref("")
const isolationMode = ref("")
const commitMessage = ref("")
const showAdvancedOperations = ref(false)
const deliveryLoading = ref(false)
const deliveryPushKey = ref(0)
const deliveryJob = ref<CodeDeliveryJob | null>(null)
const conflictResolverVisible = ref(false)
let statusPending = false
let diffSequence = 0
const entries = computed(() => codeGitReviewEntries(status.value))
const selectedEntry = computed(() => entries.value.find(entry => entry.key === selectedKey.value) || null)
const hasChanges = computed(() => entries.value.length > 0)
const totals = computed(() => codeGitReviewTotals(status.value))
const isolatedRepositories = computed(() =>
	(status.value?.repositories || []).filter(repository => repository.isolated)
)
const hasIsolation = computed(() => Boolean(worktreeBranch.value || isolationMode.value === "multi_worktree"))
const deliveryActive = computed(() => ["queued", "running"].includes(deliveryJob.value?.status || ""))
const conflictRepositories = computed(() => {
	if (deliveryJob.value?.status !== "conflict") return []
	return (deliveryJob.value.repositories || []).filter(repository => repository.status === "conflict")
})
const canSave = computed(() =>
	Boolean(
		view.value === "commit" &&
			hasIsolation.value &&
			!deliveryActive.value &&
			deliveryJob.value?.status !== "conflict" &&
			hasChanges.value
	)
)
const deliveryStatusLabel = computed(() => {
	if (!deliveryJob.value) return ""
	if (deliveryJob.value.status === "completed" && deliveryJob.value.hasUncommittedChanges) {
		return t("code.gitDeliveryStatus_unsaved")
	}
	if (deliveryJob.value.status === "completed" && deliveryJob.value.hasPendingCommits) {
		return t("code.gitDeliveryStatus_pending")
	}
	return t(`code.gitDeliveryStatus_${deliveryJob.value.status}`, {
		position: deliveryJob.value.queuePosition,
		progress: deliveryJob.value.progress
	})
})
const deliveryLabel = computed(() => {
	if (isolationMode.value === "multi_worktree") {
		return t("code.gitMultiWorktree", { count: isolatedRepositories.value.length })
	}
	return t("code.gitWorktreeBranch", { branch: worktreeBranch.value })
})
const loadDiff = async (entry: CodeGitReviewEntry, preserveContent = false) => {
	const sequence = ++diffSequence
	selectedKey.value = entry.key
	if (!preserveContent) {
		diffLoading.value = true
		diffContent.value = ""
	}
	diffTruncated.value = false
	try {
		const response = await getCodeGitDiff(
			props.sessionId as number,
			entry.repository.id,
			entry.file.path,
			entry.kind,
			entry.kind === "result" ? "result" : "workspace"
		)
		if (sequence !== diffSequence || selectedKey.value !== entry.key) return
		diffContent.value = response.data.content || ""
		diffTruncated.value = response.data.truncated
	} catch (error) {
		if (sequence !== diffSequence) return
	} finally {
		if (sequence === diffSequence && !preserveContent) diffLoading.value = false
	}
}

const reconcileSelection = async () => {
	if (!entries.value.length) {
		selectedKey.value = ""
		diffContent.value = ""
		return
	}
	const entry = selectedEntry.value || entries.value[0]
	await loadDiff(entry, entry.key === selectedKey.value && Boolean(diffContent.value))
}

const loadStatus = async (silent = false) => {
	if (!props.sessionId || statusPending || view.value === "history") return
	const requestedView = view.value
	statusPending = true
	if (!silent) loading.value = true
	else refreshing.value = true
	try {
		const [response, sessionResponse, deliveryResponse] = await Promise.all([
			getCodeGitStatus(props.sessionId, "workspace"),
			getCodeSession(props.sessionId),
			getCodeDeliveryJob(props.sessionId)
		])
		if (requestedView !== view.value) return
		status.value = response.data
		deliveryJob.value = deliveryResponse.data
		worktreeBranch.value = sessionResponse.data.session.worktreeBranch || ""
		isolationMode.value = sessionResponse.data.session.isolationMode || ""
		loadError.value = ""
		await reconcileSelection()
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.gitLoadFailed")
	} finally {
		statusPending = false
		loading.value = false
		refreshing.value = false
		if (requestedView !== view.value && props.active) void loadStatus()
	}
}

const saveChanges = async () => {
	if (!props.sessionId || !canSave.value) return
	deliveryLoading.value = true
	try {
		await saveCodeGitChanges(props.sessionId, commitMessage.value.trim())
		commitMessage.value = ""
		message.success(t("code.gitSaveSuccess"))
		await loadStatus(true)
	} catch (error) {
		void 0
	} finally {
		deliveryLoading.value = false
	}
}

const updateStage = async (entry: CodeGitReviewEntry, staged: boolean) => {
	if (!props.sessionId) return
	stagingKey.value = entry.key
	try {
		const response = await updateCodeGitStage(props.sessionId, entry.repository.id, [entry.file.path], staged)
		status.value = response.data
		selectedKey.value = ""
		diffContent.value = ""
		await reconcileSelection()
		message.success(t("code.gitStageSuccess"))
	} catch (error) {
		void 0
	} finally {
		stagingKey.value = ""
	}
}

const openSelectedFile = () => {
	const entry = selectedEntry.value
	if (!entry) return
	const filename = entry.file.workspacePath.split("/").pop() || ""
	const extension = filename.includes(".") ? filename.split(".").pop() || "" : ""
	emit("open-file", { path: entry.file.workspacePath, extension })
}

watch(view, () => {
	status.value = null
	historySelection.value = null
	selectedKey.value = ""
	diffContent.value = ""
	loadError.value = ""
	if (props.active) void loadStatus()
})
const refreshReview = () => {
	if (view.value === "history") {
		historyRefreshKey.value++
		return
	}
	void loadStatus()
}
const selectHistoryCommit = (selection: CodeGitHistorySelection | null) => {
	historySelection.value = selection
	diffContent.value = selection?.content || ""
	diffTruncated.value = selection?.truncated || false
}

watch(
	() => props.sessionId,
	() => {
		view.value = "changes"
		status.value = null
		deliveryJob.value = null
		selectedKey.value = ""
		diffContent.value = ""
		loadError.value = ""
		worktreeBranch.value = ""
		isolationMode.value = ""
		commitMessage.value = ""
		showAdvancedOperations.value = false
		if (props.active) void loadStatus()
	}
)
watch(
	() => props.active,
	active => {
		if (active) void loadStatus(Boolean(status.value))
	}
)
useIntervalFn(() => {
	if (props.active) void loadStatus(true)
}, 5000)
</script>

<template>
	<div class="flex h-full min-h-0 bg-white">
		<CodeGitDiffViewer
			:title="historySelection?.title || selectedEntry?.file.path || ''"
			:subtitle="
				historySelection?.subtitle ||
				t(
					selectedEntry?.kind === 'result'
						? 'code.gitResultDiff'
						: selectedEntry?.kind === 'staged'
							? 'code.gitStagedDiff'
							: 'code.gitWorkingDiff'
				)
			"
			:content="diffContent"
			:truncated="diffTruncated"
			:loading="diffLoading"
			:empty-description="t(view === 'history' ? 'code.gitHistorySelect' : 'code.gitSelectFile')"
			:diff-empty-description="t('code.gitDiffEmpty')"
			:truncated-description="t('code.gitDiffTruncated')"
			:open-file-label="t('code.gitOpenFile')"
			:can-open-file="
				Boolean(
					selectedEntry &&
						selectedEntry.repository.reviewState !== 'delivered' &&
						selectedEntry.file.resultStatus !== 'D' &&
						selectedEntry.file.indexStatus !== 'D' &&
						selectedEntry.file.worktreeStatus !== 'D'
				)
			"
			@open-file="openSelectedFile"
		/>

		<aside class="flex min-h-0 w-80 shrink-0 flex-col overflow-hidden border-l border-slate-200 bg-slate-50/70">
			<CodeGitReviewHeader
				v-model="view"
				:status="status"
				:additions="totals.additions"
				:deletions="totals.deletions"
				:refreshing="refreshing"
				@refresh="refreshReview"
			/>
			<CodeSessionRepositorySync
				v-if="view === 'commit' && hasIsolation && sessionId"
				:session-id="sessionId"
				:disabled="deliveryActive"
				@synced="loadStatus(true)"
			/>
			<div
				v-if="view === 'commit' && hasIsolation"
				class="max-h-[46%] shrink-0 space-y-2 overflow-y-auto border-b border-slate-200 p-3"
			>
				<div class="truncate text-xs text-slate-500" :title="deliveryLabel">{{ deliveryLabel }}</div>
				<n-alert
					v-if="deliveryJob"
					:type="
						deliveryJob.status === 'failed' ||
						deliveryJob.status === 'conflict' ||
						deliveryJob.status === 'partial'
							? 'error'
							: deliveryJob.status === 'completed'
								? 'success'
								: 'info'
					"
					:show-icon="false"
				>
					<div class="flex items-center justify-between gap-2 text-xs">
						<span>{{ deliveryStatusLabel }}</span>
						<span v-if="deliveryActive">{{ t(`code.gitDeliveryStage_${deliveryJob.stage}`) }}</span>
					</div>
					<n-progress
						v-if="deliveryActive"
						class="mt-2"
						type="line"
						:percentage="deliveryJob.progress"
						:show-indicator="false"
					/>
					<div
						v-if="deliveryJob.errorMessage && !conflictRepositories.length"
						class="mt-2 break-words text-xs"
					>
						{{ deliveryJob.errorMessage }}
					</div>
					<CodeConflictManualMerge
						:repositories="conflictRepositories"
						:session-id="sessionId"
						@resolve="conflictResolverVisible = true"
						@completed="loadStatus(true)"
					/>
					<CodeDeliveryFacts :facts="deliveryJob.facts" :job-status="deliveryJob.status" />
				</n-alert>
				<CodeLocalSyncPending
					v-if="deliveryJob?.repositories?.length"
					:repositories="deliveryJob.repositories"
				/>
				<n-input
					v-model:value="commitMessage"
					size="small"
					:placeholder="t('code.gitSavePlaceholder')"
					:disabled="deliveryLoading"
					@keyup.enter="saveChanges"
				/>
				<n-button
					block
					size="small"
					type="primary"
					:disabled="!canSave"
					:loading="deliveryLoading"
					@click="saveChanges"
				>
					{{ t("code.gitSave") }}
				</n-button>
				<p class="text-[11px] leading-4 text-slate-400">
					{{ t(hasChanges ? "code.gitSaveHint" : "code.gitMergeReady") }}
				</p>
				<n-button text size="tiny" @click="showAdvancedOperations = !showAdvancedOperations">
					{{ t(showAdvancedOperations ? "code.gitAdvancedHide" : "code.gitAdvancedShow") }}
				</n-button>
			</div>
			<CodeDeliveryPush
				v-if="view === 'commit' && !hasIsolation && sessionId"
				:session-id="sessionId"
				:refresh-key="deliveryPushKey"
			/>
			<CodeGitHistory
				v-if="view === 'history'"
				:session-id="sessionId"
				:active="active"
				:refresh-key="historyRefreshKey"
				@selected="selectHistoryCommit"
			/>
			<n-spin
				v-else
				:show="loading"
				class="min-h-0 flex-1 overflow-hidden"
				content-class="flex h-full min-h-0 flex-col"
			>
				<div v-if="loadError" class="p-4">
					<n-alert type="error" :title="t('code.gitLoadFailed')">{{ loadError }}</n-alert>
				</div>
				<n-empty
					v-else-if="status && !status.available"
					:description="t('code.gitNoRepository')"
					class="mt-16"
				/>
				<n-empty
					v-else-if="status && !hasChanges"
					:description="t('code.gitNoChanges')"
					class="mt-16"
				/>
				<CodeGitRepositoryChanges
					v-else
					class="min-h-0 flex-1"
					:repositories="status?.repositories || []"
					:entries="entries"
					:scope="'workspace'"
					:selected-key="selectedKey"
					:staging-key="stagingKey"
					:show-advanced-operations="view === 'commit' && showAdvancedOperations"
					@select="loadDiff"
					@update-stage="({ entry, staged }) => updateStage(entry, staged)"
				/>
			</n-spin>
		</aside>
	</div>
	<CodeConflictResolverDrawer
		v-if="sessionId"
		v-model:show="conflictResolverVisible"
		:session-id="sessionId"
		@completed="loadStatus(true)"
	/>
</template>
