<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { deleteCodeProjectBranch, getCodeProjectBranches } from "@/api/modules/code"
import type { CodeProjectBranch, CodeProjectBranches } from "@/api/interface/codeBranches"
import type { CodeTaskRepositorySummary } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { taskRepositoryMessages } from "../taskRepositoryMessages"
import CodeTaskDeliveryButton from "./CodeTaskDeliveryButton.vue"

const props = defineProps<{
	projectId: number
	sessionId: number
	repositories: CodeTaskRepositorySummary[]
	branch?: string
	additions: number
	deletions: number
	changedFiles: number
	unsavedAdditions: number
	unsavedDeletions: number
	unsavedFiles: number
	hasUnsavedChanges: boolean
	statusIcon?: string
	statusColor?: string
}>()

const { t } = useI18n({ messages: taskRepositoryMessages })
const dialog = useDialog()
const message = useMessage()
const show = ref(false)
const branchState = ref<CodeProjectBranches>({ repositories: [], totalBranches: 0 })
const branchStateLoading = ref(false)
const branchStateFailed = ref(false)
const deletingBranchKey = ref("")
const involvedRepositories = computed(() => {
	if (props.repositories.length) return props.repositories
	if (!props.branch) return []
	return [{
		name: t("code.taskRepositoryFallback"), branch: props.branch,
		additions: props.additions, deletions: props.deletions,
		changedFiles: props.changedFiles, hasDiff: props.changedFiles > 0
	}]
})

const loadBranchState = async () => {
	if (!props.projectId || branchStateLoading.value) return
	branchStateLoading.value = true
	branchStateFailed.value = false
	try {
		const response = await getCodeProjectBranches(props.projectId)
		if (response.code !== 0) throw new Error(response.message || t("code.zombieBranchStateFailed"))
		branchState.value = response.data
	} catch (error) {
		branchStateFailed.value = true
		message.error(error instanceof Error ? error.message : t("code.zombieBranchStateFailed"))
	} finally {
		branchStateLoading.value = false
	}
}

const repositoryBranchState = (repository: CodeTaskRepositorySummary) => {
	const candidates = branchState.value.repositories.filter(item =>
		item.excluded &&
		(repository.repositoryPath ? item.path === repository.repositoryPath : item.name === repository.name) &&
		item.branches.some(branch => branch.scope === "local" && branch.name === repository.branch)
	)
	if (candidates.length !== 1) return undefined
	const branchRepository = candidates[0]
	return {
		repository: branchRepository,
		branch: branchRepository.branches.find(branch => branch.scope === "local" && branch.name === repository.branch)
	}
}

const branchDeleteBlockLabel = (branch: CodeProjectBranch) =>
	branch.deleteBlockReason ? t(`code.branchDeleteBlocked_${branch.deleteBlockReason}`) : ""

const confirmDeleteBranch = (repository: CodeTaskRepositorySummary) => {
	const state = repositoryBranchState(repository)
	if (!state?.branch?.deletable) return
	const force = !state.branch.merged
	dialog.warning({
		title: t(force ? "code.forceDeleteBranchTitle" : "code.deleteBranchTitle"),
		content: t(force ? "code.forceDeleteBranchConfirm" : "code.deleteBranchConfirm", { branch: state.branch.name }),
		positiveText: t(force ? "code.confirmForceDeleteBranch" : "code.confirmDeleteBranch"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			const key = `${state.repository.path}:${state.branch.name}`
			deletingBranchKey.value = key
			try {
				await deleteCodeProjectBranch(props.projectId, state.repository.path, state.branch.name, force)
				message.success(t("code.zombieBranchDeleted", { branch: state.branch.name }))
				await loadBranchState()
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("code.branchDeleteFailed"))
				throw error
			} finally {
				deletingBranchKey.value = ""
			}
		}
	})
}

watch(show, opened => {
	if (opened) void loadBranchState()
})
</script>

<template>
	<n-popover v-model:show="show" trigger="click" placement="bottom-start" style="max-width: 480px">
		<template #trigger>
			<button
				type="button"
				class="flex items-center gap-1 hover:text-blue-500"
				:class="hasUnsavedChanges ? 'text-amber-600' : ''"
				:title="t(hasUnsavedChanges ? 'code.taskUnsavedChanges' : 'code.taskGitDetails')"
				@click.stop
			>
				<Icon :name="statusIcon || 'mdi:source-branch'" :size="13" :class="statusColor" />
				<template v-if="hasUnsavedChanges">
					<span class="font-medium">{{ t("code.taskUnsaved") }}</span>
					<span class="font-medium text-emerald-600">+{{ unsavedAdditions }}</span>
					<span class="font-medium text-red-500">-{{ unsavedDeletions }}</span>
					<span>{{ t("code.taskChangedFiles", { count: unsavedFiles }) }}</span>
				</template>
			</button>
		</template>
		<div class="min-w-[320px] space-y-2">
			<div class="flex items-center gap-2">
				<p class="text-xs font-medium text-slate-700">{{ t("code.taskRepositoryBranches") }}</p>
				<n-spin v-if="branchStateLoading" :size="12" />
			</div>
			<p v-if="branchStateFailed" class="text-[11px] text-red-500">{{ t("code.zombieBranchStateFailed") }}</p>
			<div v-if="hasUnsavedChanges" class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700">
				<div class="font-medium">{{ t("code.taskUnsavedChanges") }}</div>
				<div class="mt-1 flex items-center gap-1.5">
					<span class="font-medium text-emerald-600">+{{ unsavedAdditions }}</span>
					<span class="font-medium text-red-500">-{{ unsavedDeletions }}</span>
					<span>{{ t("code.taskChangedFiles", { count: unsavedFiles }) }}</span>
				</div>
			</div>
			<p v-if="involvedRepositories.length" class="text-[11px] font-medium text-slate-500">
				{{ t("code.taskCumulativeOutput") }}
			</p>
			<div
				v-for="repository in involvedRepositories"
				:key="`${repository.name}:${repository.branch}`"
				class="rounded-lg px-3 py-2"
				:class="repositoryBranchState(repository) ? 'border border-amber-200 bg-amber-50/70' : 'bg-slate-50'"
			>
				<div class="flex items-center justify-between gap-3 text-xs">
					<div class="flex min-w-0 items-center gap-1.5">
						<span class="truncate font-medium text-slate-700">{{ repository.name }}</span>
						<span
							v-if="repositoryBranchState(repository)"
							class="shrink-0 rounded bg-amber-100 px-1.5 py-0.5 text-[9px] text-amber-700"
						>
							{{ t("code.removedTaskRepository") }}
						</span>
					</div>
					<span class="shrink-0">
						<span class="font-medium text-emerald-600">+{{ repository.additions }}</span>
						<span class="ml-1 font-medium text-red-500">-{{ repository.deletions }}</span>
						<span class="ml-1 text-slate-400">{{ t("code.taskChangedFiles", { count: repository.changedFiles }) }}</span>
					</span>
				</div>
				<div class="mt-1 flex items-end justify-between gap-2">
					<div class="flex min-w-0 items-center gap-1 font-mono text-[11px] text-slate-500">
						<span class="min-w-0 break-all">{{ repository.branch }}</span>
						<template v-if="repository.targetBranch">
							<span class="shrink-0 text-slate-300">→</span>
							<span class="shrink-0 text-blue-600">{{ repository.targetBranch }}</span>
						</template>
					</div>
					<n-tooltip v-if="repositoryBranchState(repository)?.branch" trigger="hover">
						<template #trigger>
							<n-button
								size="tiny"
								type="error"
								secondary
								:disabled="!repositoryBranchState(repository)?.branch?.deletable"
								:loading="deletingBranchKey === `${repositoryBranchState(repository)?.repository.path}:${repository.branch}`"
								@click="confirmDeleteBranch(repository)"
							>
								{{ t("code.deleteZombieBranch") }}
							</n-button>
						</template>
						{{
							repositoryBranchState(repository)?.branch?.deletable
								? t("code.deleteZombieBranch")
								: branchDeleteBlockLabel(repositoryBranchState(repository)!.branch!)
						}}
					</n-tooltip>
				</div>
			</div>
			<div class="flex items-center justify-between gap-3 border-t border-slate-100 pt-2">
				<p class="text-[11px] text-slate-400">{{ t("code.taskDeliveryAllRepositories") }}</p>
				<CodeTaskDeliveryButton v-if="show" :session-id="sessionId" />
			</div>
		</div>
	</n-popover>
</template>
