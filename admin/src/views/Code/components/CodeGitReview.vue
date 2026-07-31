<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useDialog, useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import CodeDeliveryPush from "./CodeDeliveryPush.vue"
import { commitCodeGitChanges, getCodeDeliveryJob, getCodeGitDiff, getCodeGitStatus, mergeCodeSessionWorktree, updateCodeGitStage } from "@/api/modules/codeGit"
import { getCodeSession } from "@/api/modules/code"
import type { CodeDeliveryJob, CodeGitDiffKind, CodeGitFile, CodeGitRepository, CodeGitStatus } from "@/api/interface/codeGit"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

const props = defineProps<{ sessionId: number | null; active: boolean }>()
const emit = defineEmits<{ (event: "open-file", file: { path: string; extension: string }): void }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const message = useMessage()
const dialog = useDialog()
const status = ref<CodeGitStatus | null>(null)
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
const deliveryLoading = ref(false)
const deliveryPushKey = ref(0)
const deliveryJob = ref<CodeDeliveryJob | null>(null)
let statusPending = false
let diffSequence = 0

interface GitReviewEntry {
	repository: CodeGitRepository
	file: CodeGitFile
	kind: CodeGitDiffKind
	key: string
}

const entries = computed<GitReviewEntry[]>(() => {
	const result: GitReviewEntry[] = []
	for (const repository of status.value?.repositories || []) {
		for (const file of repository.files) {
			if (file.staged) {
				result.push({ repository, file, kind: "staged", key: `${repository.id}:staged:${file.path}` })
			}
			if (file.changed || file.untracked) {
				result.push({ repository, file, kind: "working", key: `${repository.id}:working:${file.path}` })
			}
		}
	}
	return result
})
const selectedEntry = computed(() => entries.value.find(entry => entry.key === selectedKey.value) || null)
const hasChanges = computed(() => entries.value.length > 0)
const totalAdditions = computed(() => (status.value?.additions || 0) + (status.value?.stagedAdditions || 0))
const totalDeletions = computed(() => (status.value?.deletions || 0) + (status.value?.stagedDeletions || 0))
const isolatedRepositories = computed(() => (status.value?.repositories || []).filter(repository => repository.isolated))
const commitRepository = computed(() => {
	if (selectedEntry.value?.repository.stagedCount) return selectedEntry.value.repository
	return isolatedRepositories.value.find(repository => repository.stagedCount > 0) || null
})
const hasIsolation = computed(() => Boolean(worktreeBranch.value || isolationMode.value === "multi_worktree"))
const deliveryActive = computed(() => ["queued", "running"].includes(deliveryJob.value?.status || ""))
const canCommit = computed(() => Boolean(hasIsolation.value && !deliveryActive.value && commitRepository.value && commitMessage.value.trim()))
const canMerge = computed(() => Boolean(hasIsolation.value && !deliveryActive.value && status.value && status.value.files === 0))
const deliveryStatusLabel = computed(() => {
	if (!deliveryJob.value) return ""
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
const diffLines = computed(() => diffContent.value.split("\n"))
const diffLineClass = (line: string) => {
	if (line.startsWith("+++") || line.startsWith("---")) return "text-slate-400"
	if (line.startsWith("+")) return "bg-emerald-500/10 text-emerald-300"
	if (line.startsWith("-")) return "bg-rose-500/10 text-rose-300"
	if (line.startsWith("@@")) return "text-sky-300"
	return "text-slate-300"
}

const entriesFor = (repositoryId: string, kind: CodeGitDiffKind | "untracked") =>
	entries.value.filter(entry => {
		if (entry.repository.id !== repositoryId) return false
		if (kind === "untracked") return entry.kind === "working" && entry.file.untracked
		if (kind === "working") return entry.kind === "working" && entry.file.changed
		return entry.kind === "staged"
	})

const loadDiff = async (entry: GitReviewEntry, preserveContent = false) => {
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
			entry.kind
		)
		if (sequence !== diffSequence || selectedKey.value !== entry.key) return
		diffContent.value = response.data.content || ""
		diffTruncated.value = response.data.truncated
	} catch (error) {
		if (sequence !== diffSequence) return
		message.error(error instanceof Error ? error.message : t("code.gitDiffFailed"))
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
	if (!props.sessionId || statusPending) return
	statusPending = true
	if (!silent) loading.value = true
	else refreshing.value = true
	try {
		const [response, sessionResponse, deliveryResponse] = await Promise.all([
			getCodeGitStatus(props.sessionId), getCodeSession(props.sessionId), getCodeDeliveryJob(props.sessionId)
		])
		status.value = response.data
		deliveryJob.value = deliveryResponse.data
		worktreeBranch.value = sessionResponse.data.session.worktreeBranch || ""
		isolationMode.value = sessionResponse.data.session.isolationMode || ""
		loadError.value = ""
		await reconcileSelection()
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.gitLoadFailed")
		if (!silent) message.error(loadError.value)
	} finally {
		statusPending = false
		loading.value = false
		refreshing.value = false
	}
}

const commitChanges = async () => {
	if (!props.sessionId || !canCommit.value) return
	deliveryLoading.value = true
	try {
		await commitCodeGitChanges(props.sessionId, commitRepository.value?.id || "", commitMessage.value.trim())
		commitMessage.value = ""
		message.success(t("code.gitCommitSuccess"))
		await loadStatus(true)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.gitCommitFailed"))
	} finally {
		deliveryLoading.value = false
	}
}

const mergeWorktree = () => {
	if (!props.sessionId || !canMerge.value) return
	dialog.warning({
		title: t("code.gitMergeTitle"),
		content: t(isolationMode.value === "multi_worktree" ? "code.gitMultiMergeConfirm" : "code.gitMergeConfirm", {
			branch: worktreeBranch.value, count: isolatedRepositories.value.length
		}),
		positiveText: t("code.gitMerge"),
		negativeText: t("code.gitCancel"),
		onPositiveClick: async () => {
			deliveryLoading.value = true
			try {
				const response = await mergeCodeSessionWorktree(props.sessionId as number)
				deliveryJob.value = response.data
				message.success(t("code.gitDeliveryQueued"))
				await loadStatus(true)
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("code.gitMergeFailed"))
			} finally {
				deliveryLoading.value = false
			}
		}
	})
}

const updateStage = async (entry: GitReviewEntry, staged: boolean) => {
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
		message.error(error instanceof Error ? error.message : t("code.gitStageFailed"))
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

watch(
	() => props.sessionId,
	() => {
		status.value = null
		deliveryJob.value = null
		selectedKey.value = ""
		diffContent.value = ""
		loadError.value = ""
		worktreeBranch.value = ""
		isolationMode.value = ""
		commitMessage.value = ""
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
		<section class="flex min-w-0 flex-1 flex-col bg-[#0f172a] text-slate-100">
			<div
				v-if="selectedEntry"
				class="flex h-12 shrink-0 items-center justify-between gap-3 border-b border-slate-700 px-4"
			>
				<div class="min-w-0">
					<div class="truncate text-sm font-semibold">{{ selectedEntry.file.path }}</div>
					<div class="text-[11px] text-slate-400">
						{{ t(selectedEntry.kind === "staged" ? "code.gitStagedDiff" : "code.gitWorkingDiff") }}
					</div>
				</div>
				<n-button
					v-if="selectedEntry.file.indexStatus !== 'D' && selectedEntry.file.worktreeStatus !== 'D'"
					size="small"
					secondary
					@click="openSelectedFile"
				>
					{{ t("code.gitOpenFile") }}
				</n-button>
			</div>
			<n-spin :show="diffLoading" class="min-h-0 flex-1">
				<div v-if="!selectedEntry" class="flex h-full items-center justify-center">
					<n-empty :description="t('code.gitSelectFile')" />
				</div>
				<div v-else-if="!diffLoading && !diffContent" class="flex h-full items-center justify-center">
					<n-empty :description="t('code.gitDiffEmpty')" />
				</div>
				<div v-else class="flex h-full min-h-0 flex-col">
					<n-alert v-if="diffTruncated" type="warning" :show-icon="false" class="m-3 mb-0">
						{{ t("code.gitDiffTruncated") }}
					</n-alert>
					<pre class="min-h-0 flex-1 overflow-auto p-4 font-mono text-xs leading-5"><span
						v-for="(line, index) in diffLines"
						:key="index"
						class="block min-w-max px-1"
						:class="diffLineClass(line)"
					>{{ line || " " }}</span></pre>
				</div>
			</n-spin>
		</section>

		<aside class="flex w-80 shrink-0 flex-col border-l border-slate-200 bg-slate-50/70">
			<div class="border-b border-slate-200 p-3">
				<div class="flex items-center justify-between gap-3">
					<div>
						<div class="text-sm font-semibold text-slate-800">{{ t("code.gitReview") }}</div>
						<div v-if="status?.available" class="mt-1 flex items-center gap-2 text-xs text-slate-500">
							<span>{{ t("code.gitSummary", { files: status.files }) }}</span>
							<span class="text-emerald-600">+{{ totalAdditions }}</span>
							<span class="text-rose-500">-{{ totalDeletions }}</span>
						</div>
					</div>
					<n-button
						circle
						quaternary
						size="small"
						:loading="refreshing"
						:title="t('code.gitRefresh')"
						@click="loadStatus()"
					>
						<template #icon><Icon name="mdi:refresh" :size="17" /></template>
					</n-button>
				</div>
			</div>
			<div v-if="hasIsolation" class="space-y-2 border-b border-slate-200 p-3">
				<div class="truncate text-xs text-slate-500" :title="deliveryLabel">{{ deliveryLabel }}</div>
				<div v-if="commitRepository" class="truncate text-[11px] text-slate-400">
					{{ t("code.gitCommitRepository", { repository: commitRepository.name }) }}
				</div>
				<n-alert v-if="deliveryJob" :type="deliveryJob.status === 'failed' || deliveryJob.status === 'conflict' ? 'error' : deliveryJob.status === 'completed' ? 'success' : 'info'" :show-icon="false">
					<div class="flex items-center justify-between gap-2 text-xs">
						<span>{{ deliveryStatusLabel }}</span>
						<span v-if="deliveryActive">{{ t(`code.gitDeliveryStage_${deliveryJob.stage}`) }}</span>
					</div>
					<n-progress v-if="deliveryActive" class="mt-2" type="line" :percentage="deliveryJob.progress" :show-indicator="false" />
					<div v-if="deliveryJob.errorMessage" class="mt-2 break-words text-xs">{{ deliveryJob.errorMessage }}</div>
				</n-alert>
				<n-input v-model:value="commitMessage" size="small" :placeholder="t('code.gitCommitPlaceholder')" :disabled="deliveryLoading" @keyup.enter="commitChanges" />
				<div class="grid grid-cols-2 gap-2">
					<n-button size="small" type="primary" :disabled="!canCommit" :loading="deliveryLoading" @click="commitChanges">{{ t("code.gitCommit") }}</n-button>
					<n-button size="small" type="success" secondary :disabled="!canMerge" :loading="deliveryLoading" @click="mergeWorktree">{{ t("code.gitMerge") }}</n-button>
				</div>
				<p class="text-[11px] leading-4 text-slate-400">{{ t(canMerge ? "code.gitMergeReady" : "code.gitMergeHint") }}</p>
			</div>
			<CodeDeliveryPush v-if="!hasIsolation && sessionId" :session-id="sessionId" :refresh-key="deliveryPushKey" />
			<n-spin :show="loading" class="min-h-0 flex-1">
				<div v-if="loadError" class="p-4">
					<n-alert type="error" :title="t('code.gitLoadFailed')">{{ loadError }}</n-alert>
				</div>
				<n-empty
					v-else-if="status && !status.available"
					:description="t('code.gitNoRepository')"
					class="mt-16"
				/>
				<n-empty v-else-if="status && !hasChanges" :description="t('code.gitNoChanges')" class="mt-16" />
				<n-scrollbar v-else class="h-full">
					<div class="space-y-4 p-2.5">
						<section v-for="repository in status?.repositories || []" :key="repository.id">
							<div class="mb-2 flex items-center gap-2 px-2">
								<Icon name="mdi:source-repository" :size="16" />
								<span class="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
									{{ repository.name }}
								</span>
								<span class="truncate text-[11px] text-slate-400">
									{{ repository.branch || t("code.gitBranchDetached") }}
								</span>
							</div>
							<template
								v-for="group in [
									{ kind: 'staged', label: t('code.gitStaged') },
									{ kind: 'working', label: t('code.gitChanged') },
									{ kind: 'untracked', label: t('code.gitUntracked') }
								]"
								:key="group.kind"
							>
								<div
									v-if="entriesFor(repository.id, group.kind as CodeGitDiffKind | 'untracked').length"
									class="mb-2"
								>
									<div
										class="px-2 py-1 text-[11px] font-semibold uppercase tracking-wider text-slate-400"
									>
										{{ group.label }}
									</div>
									<button
										v-for="entry in entriesFor(
											repository.id,
											group.kind as CodeGitDiffKind | 'untracked'
										)"
										:key="entry.key"
										type="button"
										class="group flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left hover:bg-slate-200/70"
										:class="
											selectedKey === entry.key ? 'bg-blue-50 text-blue-700' : 'text-slate-600'
										"
										@click="loadDiff(entry)"
									>
										<span class="w-4 shrink-0 text-center text-xs font-semibold">
											{{
												entry.file.untracked
													? "U"
													: entry.kind === "staged"
														? entry.file.indexStatus
														: entry.file.worktreeStatus
											}}
										</span>
										<span class="min-w-0 flex-1 truncate text-xs" :title="entry.file.path">
											{{ entry.file.path }}
										</span>
										<n-button
											text
											size="tiny"
											:loading="stagingKey === entry.key"
											@click.stop="updateStage(entry, entry.kind !== 'staged')"
										>
											{{ t(entry.kind === "staged" ? "code.gitUnstage" : "code.gitStage") }}
										</n-button>
									</button>
								</div>
							</template>
							<n-alert v-if="repository.truncated" type="warning" :show-icon="false" class="mx-2 mt-2">
								{{ t("code.gitFilesTruncated") }}
							</n-alert>
						</section>
					</div>
				</n-scrollbar>
			</n-spin>
		</aside>
	</div>
</template>

<style scoped>
:deep(.n-spin-container),
:deep(.n-spin-content) {
	height: 100%;
}
</style>
