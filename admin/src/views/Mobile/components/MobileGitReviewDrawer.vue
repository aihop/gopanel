<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { CodeSession } from "@/api/interface/code"
import type { CodeDeliveryJob, CodeGitHistorySelection, CodeGitStatus } from "@/api/interface/codeGit"
import {
	completeMobileDeliveryConflicts,
	getMobileDeliveryConflictFile,
	getMobileDeliveryConflicts,
	getMobileGitDiff,
	getMobileGitHistory,
	getMobileGitHistoryDiff,
	getMobileGitStatus,
	saveMobileDeliveryConflictFile,
	saveMobileGitChanges,
	updateMobileGitStage
} from "@/api/modules/mobile"
import CodeConflictResolverDrawer from "@/views/Code/components/CodeConflictResolverDrawer.vue"
import CodeGitHistory from "@/views/Code/components/CodeGitHistory.vue"
import CodeGitRepositoryChanges from "@/views/Code/components/CodeGitRepositoryChanges.vue"
import CodeGitReviewHeader from "@/views/Code/components/CodeGitReviewHeader.vue"
import { codeGitReviewEntries, codeGitReviewTotals, type CodeGitReviewEntry } from "@/views/Code/codeGitReviewEntries"
import { codeGitReviewMessages } from "@/views/Code/codeGitReviewMessages"
import type { CodeGitReviewView } from "@/views/Code/codeGitReviewView"
import MobileGitCommitActions from "./MobileGitCommitActions.vue"
import MobileGitDiffDrawer from "./MobileGitDiffDrawer.vue"

const props = defineProps<{
	show: boolean
	session: CodeSession
	delivery?: CodeDeliveryJob | null
}>()
const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "updated"): void
}>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const message = useMessage()
const view = ref<CodeGitReviewView>("changes")
const status = ref<CodeGitStatus | null>(null)
const loading = ref(false)
const refreshing = ref(false)
const loadError = ref("")
const selectedKey = ref("")
const diffTitle = ref("")
const diffSubtitle = ref("")
const diffContent = ref("")
const diffTruncated = ref(false)
const diffLoading = ref(false)
const diffVisible = ref(false)
const stagingKey = ref("")
const saving = ref(false)
const commitMessage = ref("")
const showAdvancedOperations = ref(false)
const conflictResolverVisible = ref(false)
const revision = ref(0)
const historyRefreshKey = ref(0)
let statusPending = false
let diffSequence = 0

const entries = computed(() => codeGitReviewEntries(status.value))
const totals = computed(() => codeGitReviewTotals(status.value))
const reviewScope = computed(() => (view.value === "changes" ? "result" : "workspace"))

const loadStatus = async (silent = false) => {
	if (!props.session.id || statusPending || view.value === "history") return
	const requestedView = view.value
	const requestedSessionId = props.session.id
	statusPending = true
	if (silent) refreshing.value = true
	else loading.value = true
	try {
		const response = await getMobileGitStatus(requestedSessionId, reviewScope.value)
		if (requestedView !== view.value || requestedSessionId !== props.session.id) return
		status.value = response
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.gitLoadFailed")
	} finally {
		statusPending = false
		loading.value = false
		refreshing.value = false
		if ((requestedView !== view.value || requestedSessionId !== props.session.id) && props.show) void loadStatus()
	}
}

const loadDiff = async (entry: CodeGitReviewEntry) => {
	const sequence = ++diffSequence
	selectedKey.value = entry.key
	diffTitle.value = entry.file.path
	diffSubtitle.value = t(
		entry.kind === "result"
			? "code.gitResultDiff"
			: entry.kind === "staged"
				? "code.gitStagedDiff"
				: "code.gitWorkingDiff"
	)
	diffContent.value = ""
	diffTruncated.value = false
	diffLoading.value = true
	diffVisible.value = true
	try {
		const response = await getMobileGitDiff(
			props.session.id,
			entry.repository.id,
			entry.file.path,
			entry.kind,
			entry.kind === "result" ? "result" : "workspace"
		)
		if (sequence !== diffSequence) return
		diffContent.value = response.content || ""
		diffTruncated.value = response.truncated
	} catch (error) {
		if (sequence === diffSequence) message.error(t("code.gitDiffFailed"))
	} finally {
		if (sequence === diffSequence) diffLoading.value = false
	}
}

const selectHistoryCommit = (selection: CodeGitHistorySelection | null) => {
	if (!selection) return
	diffTitle.value = selection.title
	diffSubtitle.value = selection.subtitle
	diffContent.value = selection.content || ""
	diffTruncated.value = selection.truncated
	diffLoading.value = false
	diffVisible.value = true
}

const loadHistory = (sessionId: number) => getMobileGitHistory(sessionId)
const loadHistoryDiff = (sessionId: number, repositoryId: string, commit: string) =>
	getMobileGitHistoryDiff(sessionId, repositoryId, commit)

const updateStage = async (entry: CodeGitReviewEntry, staged: boolean) => {
	stagingKey.value = entry.key
	try {
		status.value = await updateMobileGitStage(props.session.id, entry.repository.id, [entry.file.path], staged)
		message.success(t("code.gitStageSuccess"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.gitStageFailed"))
	} finally {
		stagingKey.value = ""
	}
}

const saveChanges = async () => {
	if (!status.value?.files || saving.value) return
	saving.value = true
	try {
		await saveMobileGitChanges(props.session.id, commitMessage.value.trim())
		commitMessage.value = ""
		message.success(t("code.gitSaveSuccess"))
		revision.value++
		await loadStatus(true)
		emit("updated")
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.gitSaveFailed"))
	} finally {
		saving.value = false
	}
}

const refresh = () => {
	if (view.value === "history") historyRefreshKey.value++
	else void loadStatus()
}

const conflictResolutionCompleted = async () => {
	revision.value++
	await loadStatus(true)
	emit("updated")
}

watch(view, () => {
	selectedKey.value = ""
	loadError.value = ""
	if (props.show && view.value !== "history") void loadStatus()
})
watch(
	() => [props.show, props.session.id],
	([show]) => {
		if (!show) return
		view.value = "changes"
		status.value = null
		selectedKey.value = ""
		diffVisible.value = false
		void loadStatus()
	},
	{ immediate: true }
)
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="92dvh" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('code.gitReview')" closable body-content-style="padding: 0;">
			<div class="flex h-[calc(92dvh-58px)] min-h-0 flex-col bg-white">
				<CodeGitReviewHeader
					v-model="view"
					:status="status"
					:additions="totals.additions"
					:deletions="totals.deletions"
					:refreshing="refreshing"
					@refresh="refresh"
				/>
				<MobileGitCommitActions
					v-if="view === 'commit'"
					v-model:message="commitMessage"
					v-model:advanced="showAdvancedOperations"
					:session="session"
					:delivery="delivery"
					:status="status"
					:saving="saving"
					:active="show"
					:revision="revision"
					@save="saveChanges"
					@resolve="conflictResolverVisible = true"
					@synced="loadStatus(true)"
					@updated="emit('updated')"
				/>
				<CodeGitHistory
					v-if="view === 'history'"
					:session-id="session.id"
					:active="show"
					:refresh-key="historyRefreshKey"
					:load-history="loadHistory"
					:load-diff="loadHistoryDiff"
					:auto-select="false"
					@selected="selectHistoryCommit"
				/>
				<n-spin v-else :show="loading" class="min-h-0 flex-1">
					<div v-if="loadError" class="p-4">
						<n-alert type="error" :title="t('code.gitLoadFailed')">{{ loadError }}</n-alert>
					</div>
					<n-empty
						v-else-if="status && !status.available"
						:description="t(view === 'changes' ? 'code.gitResultUnavailable' : 'code.gitNoRepository')"
						class="mt-16"
					/>
					<n-empty
						v-else-if="status && !entries.length"
						:description="t(view === 'changes' ? 'code.gitResultEmpty' : 'code.gitNoChanges')"
						class="mt-16"
					/>
					<CodeGitRepositoryChanges
						v-else
						class="h-full min-h-0"
						:repositories="status?.repositories || []"
						:entries="entries"
						:scope="reviewScope"
						:selected-key="selectedKey"
						:staging-key="stagingKey"
						:show-advanced-operations="view === 'commit' && showAdvancedOperations"
						@select="loadDiff"
						@update-stage="({ entry, staged }) => updateStage(entry, staged)"
					/>
				</n-spin>
			</div>
		</n-drawer-content>
	</n-drawer>
	<MobileGitDiffDrawer
		v-model:show="diffVisible"
		:title="diffTitle"
		:subtitle="diffSubtitle"
		:content="diffContent"
		:truncated="diffTruncated"
		:loading="diffLoading"
	/>
	<CodeConflictResolverDrawer
		v-model:show="conflictResolverVisible"
		:session-id="session.id"
		:load-conflicts="getMobileDeliveryConflicts"
		:load-conflict-file="getMobileDeliveryConflictFile"
		:save-conflict-file="saveMobileDeliveryConflictFile"
		:complete-conflicts="completeMobileDeliveryConflicts"
		mobile
		@completed="conflictResolutionCompleted"
	/>
</template>

<style scoped>
:deep(.n-spin-container),
:deep(.n-spin-content) {
	height: 100%;
}
</style>
