<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { CodeProjectBranch, CodeProjectBranchRepository } from "@/api/interface/codeBranches"
import Icon from "@/components/common/Icon.vue"
import { projectBranchMessages } from "../projectBranchMessages"

defineProps<{
	show: boolean
	repositories: CodeProjectBranchRepository[]
	loading: boolean
	loadFailed: boolean
	deletingBranchKey: string
}>()
const emit = defineEmits<{
	"update:show": [show: boolean]
	refresh: []
	deleteBranch: [repositoryPath: string, branch: CodeProjectBranch]
}>()
const { t } = useI18n({ messages: projectBranchMessages })

const branchScopeLabel = (branch: CodeProjectBranch) =>
	t(branch.scope === "remote" ? "code.remoteBranch" : "code.localBranch")
const branchDeleteBlockLabel = (branch: CodeProjectBranch) =>
	branch.deleteBlockReason ? t(`code.branchDeleteBlocked_${branch.deleteBlockReason}`) : ""
</script>

<template>
	<n-drawer :show="show" placement="right" style="width: min(680px, 100vw)" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('code.branchManagementTitle')" closable body-content-style="padding: 16px;">
			<div class="mb-4 flex items-center justify-between gap-3">
				<p class="text-xs text-slate-400">{{ t("code.branchManagementHint") }}</p>
				<n-button quaternary circle size="small" :loading="loading" :aria-label="t('code.refreshBranches')" @click="emit('refresh')">
					<template #icon><Icon name="mdi:refresh" :size="16" /></template>
				</n-button>
			</div>
			<div v-if="loading && repositories.length === 0" class="flex min-h-[240px] items-center justify-center">
				<n-spin size="small" />
			</div>
			<div v-else-if="loadFailed && repositories.length === 0" class="flex min-h-[240px] flex-col items-center justify-center gap-2 text-xs text-red-500">
				<span>{{ t("code.branchLoadFailed") }}</span>
				<n-button text type="primary" size="tiny" @click="emit('refresh')">{{ t("code.retry") }}</n-button>
			</div>
			<n-empty v-else-if="repositories.length === 0" :description="t('code.noGitRepositories')" />
			<div v-else class="space-y-4">
				<section v-for="repository in repositories" :key="repository.path" class="rounded-xl border border-slate-200/80 p-3">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<div class="flex items-center gap-1.5 text-sm font-semibold text-slate-700" :title="repository.path">
								<Icon name="mdi:source-repository" :size="15" class="shrink-0 text-slate-400" />
								<span class="truncate">{{ repository.name }}</span>
								<span v-if="repository.excluded" class="shrink-0 rounded bg-amber-50 px-1.5 py-0.5 text-[10px] text-amber-700">
									{{ t("code.excludedRepository") }}
								</span>
							</div>
							<p v-if="repository.excluded" class="mt-1 text-[11px] text-amber-600">{{ t("code.removedRepositoryCleanupHint") }}</p>
						</div>
						<span v-if="repository.dirty" class="shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-700">
							{{ t("code.changedFiles", { count: repository.changedFiles }) }}
						</span>
					</div>
					<div class="mt-3 divide-y divide-slate-100">
						<div v-for="branch in repository.branches" :key="branch.ref" class="py-2.5">
							<div class="flex items-center justify-between gap-3">
								<div class="flex min-w-0 items-center gap-1.5 text-xs font-medium" :class="branch.current ? 'text-blue-700' : 'text-slate-700'" :title="branch.name">
									<Icon :name="branch.current ? 'mdi:source-branch-check' : 'mdi:source-branch'" :size="14" :class="branch.current ? 'text-blue-600' : 'text-slate-400'" />
									<span class="truncate font-mono">{{ branch.name }}</span>
									<span v-if="branch.taskBranch" class="shrink-0 rounded bg-violet-50 px-1.5 py-0.5 text-[10px] text-violet-600">{{ t("code.taskBranch") }}</span>
									<span v-else-if="branch.managed" class="shrink-0 rounded bg-blue-50 px-1.5 py-0.5 text-[10px] text-blue-600">{{ t("code.managedBranch") }}</span>
								</div>
								<div class="flex shrink-0 items-center gap-1.5">
									<span class="text-[10px] uppercase tracking-wide text-slate-400">{{ branchScopeLabel(branch) }}</span>
									<n-tooltip v-if="branch.scope === 'local'" trigger="hover">
										<template #trigger>
											<n-button text circle size="tiny" type="error" :disabled="!branch.deletable" :loading="deletingBranchKey === `${repository.path}:${branch.ref}`" :aria-label="t('code.deleteBranch')" @click="emit('deleteBranch', repository.path, branch)">
												<template #icon><Icon name="mdi:delete-outline" :size="14" /></template>
											</n-button>
										</template>
										{{ branch.deletable ? t("code.deleteBranch") : branchDeleteBlockLabel(branch) }}
									</n-tooltip>
								</div>
							</div>
							<div class="mt-1.5 flex items-center gap-2 text-[10px] text-slate-400">
								<span class="font-mono">{{ branch.commit }}</span>
								<span class="font-medium text-emerald-600">+{{ branch.additions }}</span>
								<span class="font-medium text-red-500">-{{ branch.deletions }}</span>
								<span class="min-w-0 truncate" :title="branch.subject">{{ branch.subject }}</span>
								<span v-if="branch.scope === 'local' && branch.merged" class="ml-auto shrink-0 text-emerald-600">{{ t("code.branchMerged") }}</span>
							</div>
						</div>
					</div>
				</section>
			</div>
		</n-drawer-content>
	</n-drawer>
</template>
