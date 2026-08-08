<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { deleteCodeProjectBranch, getCodeProjectBranches } from "@/api/modules/code"
import type { CodeProjectBranch, CodeProjectBranches } from "@/api/interface/codeBranches"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{ projectId: number }>()
const { t } = useI18n({ messages: codeWorkspaceMessages })
const dialog = useDialog()
const message = useMessage()
const branchState = ref<CodeProjectBranches>({ repositories: [], totalBranches: 0 })
const loading = ref(false)
const loadFailed = ref(false)
const deletingBranchKey = ref("")
let refreshTimer: ReturnType<typeof setInterval> | undefined

const repositoryCountLabel = computed(() =>
	t("code.repositoryCount", { count: branchState.value.repositories.length })
)

const fetchBranches = async (silent = false) => {
	if (!props.projectId || loading.value) return
	loading.value = true
	if (!silent) loadFailed.value = false
	try {
		const response = await getCodeProjectBranches(props.projectId)
		if (response.code !== 0) throw new Error(response.message || t("code.branchLoadFailed"))
		branchState.value = response.data
		loadFailed.value = false
	} catch {
		loadFailed.value = true
	} finally {
		loading.value = false
	}
}

const branchScopeLabel = (branch: CodeProjectBranch) =>
	t(branch.scope === "remote" ? "code.remoteBranch" : "code.localBranch")

const branchDeleteBlockLabel = (branch: CodeProjectBranch) =>
	branch.deleteBlockReason ? t(`code.branchDeleteBlocked_${branch.deleteBlockReason}`) : ""

const confirmDeleteBranch = (repositoryPath: string, branch: CodeProjectBranch) => {
	if (!branch.deletable) return
	const force = !branch.merged
	dialog.warning({
		title: t(force ? "code.forceDeleteBranchTitle" : "code.deleteBranchTitle"),
		content: t(force ? "code.forceDeleteBranchConfirm" : "code.deleteBranchConfirm", { branch: branch.name }),
		positiveText: t(force ? "code.confirmForceDeleteBranch" : "code.confirmDeleteBranch"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			const key = `${repositoryPath}:${branch.ref}`
			deletingBranchKey.value = key
			try {
				await deleteCodeProjectBranch(props.projectId, repositoryPath, branch.name, force)
				message.success(t("code.branchDeleted", { branch: branch.name }))
				await fetchBranches(true)
			} catch (error) {
				throw error
			} finally {
				deletingBranchKey.value = ""
			}
		}
	})
}

onMounted(() => {
	void fetchBranches()
	refreshTimer = setInterval(() => void fetchBranches(true), 60000)
})
onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
watch(
	() => props.projectId,
	() => {
		branchState.value = { repositories: [], totalBranches: 0 }
		void fetchBranches()
	}
)
</script>

<template>
	<section
		class="ai-workspace-branches flex max-h-[42%] min-h-[168px] shrink-0 flex-col border-t border-slate-200/80 bg-white/65"
	>
		<div class="flex shrink-0 items-center justify-between gap-2 px-4 py-2.5">
			<div class="mt-0.5 min-w-0 text-[11px] text-slate-400">
				{{
					t("code.branchSummary", {
						branches: branchState.totalBranches,
						repositories: repositoryCountLabel
					})
				}}
			</div>
			<n-button
				quaternary
				circle
				size="tiny"
				:loading="loading"
				:aria-label="t('code.refreshBranches')"
				@click="fetchBranches()"
			>
				<template #icon><Icon name="mdi:refresh" :size="15" /></template>
			</n-button>
		</div>
		<div v-if="loading && branchState.repositories.length === 0" class="flex min-h-0 flex-1 items-center justify-center">
			<n-spin size="small" />
		</div>
		<div
			v-else-if="loadFailed && branchState.repositories.length === 0"
			class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-4 text-center text-xs text-red-500"
		>
			<span>{{ t("code.branchLoadFailed") }}</span>
			<n-button text type="primary" size="tiny" @click="fetchBranches()">{{ t("code.retry") }}</n-button>
		</div>
		<div
			v-else-if="branchState.repositories.length === 0"
			class="flex min-h-0 flex-1 items-center justify-center px-4 text-center text-xs text-slate-400"
		>
			{{ t("code.noGitRepositories") }}
		</div>
		<n-scrollbar v-else trigger="none" class="ai-workspace-branch-scrollbar min-h-0 flex-1">
			<div class="space-y-3 px-4 pb-3">
				<div v-for="repository in branchState.repositories" :key="repository.path">
					<div class="flex items-center justify-between gap-2">
						<div class="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-slate-700" :title="repository.path">
							<Icon name="mdi:source-repository" :size="14" class="shrink-0 text-slate-400" />
							<span class="truncate">{{ repository.name }}</span>
						</div>
						<span
							v-if="repository.dirty"
							class="shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-700"
						>
							{{ t("code.changedFiles", { count: repository.changedFiles }) }}
						</span>
					</div>
					<div class="mt-1.5 space-y-0.5 pl-1">
						<div
							v-for="branch in repository.branches"
							:key="branch.ref"
							class="py-1.5 pl-4"
							:class="branch.current ? 'text-blue-600' : ''"
						>
							<div class="flex items-center justify-between gap-2">
								<div
									class="flex min-w-0 items-center gap-1.5 text-[11px] font-medium"
									:class="branch.current ? 'text-blue-700' : 'text-slate-700'"
									:title="branch.name"
								>
									<Icon
										:name="branch.current ? 'mdi:source-branch-check' : 'mdi:source-branch'"
										:size="13"
										:class="branch.current ? 'text-blue-600' : 'text-slate-400'"
									/>
									<span class="truncate">{{ branch.name }}</span>
									<span v-if="branch.managed" class="shrink-0 rounded bg-blue-50 px-1 text-[9px] text-blue-600">
										{{ t("code.managedBranch") }}
									</span>
								</div>
								<div class="flex shrink-0 items-center gap-1">
									<span class="text-[9px] uppercase tracking-wide text-slate-400">{{ branchScopeLabel(branch) }}</span>
									<n-tooltip v-if="branch.scope === 'local'" trigger="hover">
										<template #trigger>
											<n-button
												text
												circle
												size="tiny"
												type="error"
												:disabled="!branch.deletable"
												:loading="deletingBranchKey === `${repository.path}:${branch.ref}`"
												:aria-label="t('code.deleteBranch')"
												@click="confirmDeleteBranch(repository.path, branch)"
											>
												<template #icon><Icon name="mdi:delete-outline" :size="13" /></template>
											</n-button>
										</template>
										{{ branch.deletable ? t("code.deleteBranch") : branchDeleteBlockLabel(branch) }}
									</n-tooltip>
								</div>
							</div>
							<div class="mt-1 flex items-center gap-1.5 text-[10px] text-slate-400">
								<span class="font-mono">{{ branch.commit }}</span>
								<span class="font-medium text-emerald-600">+{{ branch.additions }}</span>
								<span class="font-medium text-red-500">-{{ branch.deletions }}</span>
								<span class="truncate" :title="branch.subject">{{ branch.subject }}</span>
								<span v-if="branch.scope === 'local' && branch.merged" class="ml-auto shrink-0 text-emerald-600">
									{{ t("code.branchMerged") }}
								</span>
							</div>
						</div>
					</div>
				</div>
			</div>
		</n-scrollbar>
	</section>
</template>

<style scoped>
.ai-workspace-branch-scrollbar :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical) {
	right: 3px !important;
	width: 3px !important;
}

.ai-workspace-branch-scrollbar :deep(.n-scrollbar-rail__scrollbar) {
	width: 3px !important;
	background-color: rgba(100, 116, 139, 0.28) !important;
}

:global(.theme-dark) .ai-workspace-branches {
	border-top-color: color-mix(in srgb, var(--border-color) 80%, transparent);
	background-color: color-mix(in srgb, var(--bg-default-color) 76%, transparent);
}
</style>
