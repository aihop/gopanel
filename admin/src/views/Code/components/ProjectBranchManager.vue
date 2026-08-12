<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { deleteCodeProjectBranch, getCodeProjectBranches } from "@/api/modules/code"
import type { CodeProjectBranch, CodeProjectBranchRepository, CodeProjectBranches } from "@/api/interface/codeBranches"
import Icon from "@/components/common/Icon.vue"
import { projectBranchMessages } from "../projectBranchMessages"
import ProjectBranchManagementDrawer from "./ProjectBranchManagementDrawer.vue"
import CodeResidueManager from "./CodeResidueManager.vue"

const props = defineProps<{ projectId: number }>()
const { t } = useI18n({ messages: projectBranchMessages })
const dialog = useDialog()
const message = useMessage()
const branchState = ref<CodeProjectBranches>({ repositories: [], totalBranches: 0 })
const loading = ref(false)
const loadFailed = ref(false)
const showBranchManager = ref(false)
const deletingBranchKey = ref("")
let refreshTimer: ReturnType<typeof setInterval> | undefined

const activeRepositories = computed(() => branchState.value.repositories.filter(repository => !repository.excluded))
const statusBranches = (repository: CodeProjectBranchRepository) =>
	repository.branches.filter(branch => branch.scope === "local" && !branch.taskBranch)
const currentBranch = (repository: CodeProjectBranchRepository) =>
	statusBranches(repository).find(branch => branch.current)
const otherBranches = (repository: CodeProjectBranchRepository) =>
	statusBranches(repository).filter(branch => !branch.current)

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

const confirmDeleteBranch = (repositoryPath: string, branch: CodeProjectBranch) => {
	if (!branch.deletable) return
	const force = !branch.merged
	dialog.warning({
		title: t(force ? "code.forceDeleteBranchTitle" : "code.deleteBranchTitle"),
		content: t(force ? "code.forceDeleteBranchConfirm" : "code.deleteBranchConfirm", { branch: branch.name }),
		positiveText: t(force ? "code.confirmForceDeleteBranch" : "code.confirmDeleteBranch"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			deletingBranchKey.value = `${repositoryPath}:${branch.ref}`
			try {
				await deleteCodeProjectBranch(props.projectId, repositoryPath, branch.name, force)
				message.success(t("code.branchDeleted", { branch: branch.name }))
				await fetchBranches(true)
			} catch {
				message.error(t("code.branchDeleteFailed"))
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
		showBranchManager.value = false
		void fetchBranches()
	}
)
</script>

<template>
	<section class="ai-workspace-repositories flex max-h-[32%] min-h-[132px] shrink-0 flex-col border-t border-slate-200/80 bg-white/65">
		<div class="flex shrink-0 items-center justify-between gap-2 px-4 py-2.5">
			<div class="min-w-0">
				<div class="text-xs font-semibold text-slate-600">{{ t("code.repositoryStatus") }}</div>
				<div class="mt-0.5 text-[10px] text-slate-400">
					{{ t("code.repositoryCount", { count: activeRepositories.length }) }}
				</div>
			</div>
			<div class="flex shrink-0 items-center gap-1">
				<CodeResidueManager />
				<n-button text size="tiny" :disabled="branchState.repositories.length === 0" @click="showBranchManager = true">
					<template #icon><Icon name="mdi:source-branch" :size="14" /></template>
					{{ t("code.manageBranches") }}
				</n-button>
				<n-button quaternary circle size="tiny" :loading="loading" :aria-label="t('code.refreshRepositories')" @click="fetchBranches()">
					<template #icon><Icon name="mdi:refresh" :size="15" /></template>
				</n-button>
			</div>
		</div>
		<div v-if="loading && activeRepositories.length === 0" class="flex min-h-0 flex-1 items-center justify-center">
			<n-spin size="small" />
		</div>
		<div v-else-if="loadFailed && activeRepositories.length === 0" class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-4 text-center text-xs text-red-500">
			<span>{{ t("code.repositoryLoadFailed") }}</span>
			<n-button text type="primary" size="tiny" @click="fetchBranches()">{{ t("code.retry") }}</n-button>
		</div>
		<div v-else-if="activeRepositories.length === 0" class="flex min-h-0 flex-1 items-center justify-center px-4 text-center text-xs text-slate-400">
			{{ t("code.noGitRepositories") }}
		</div>
		<n-scrollbar v-else trigger="none" class="ai-workspace-repository-scrollbar min-h-0 flex-1">
			<div class="space-y-2 px-4 pb-3">
				<div v-for="repository in activeRepositories" :key="repository.path" class="rounded-lg bg-slate-50/75 px-2.5 py-2">
					<div class="flex items-center justify-between gap-2">
						<div class="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-slate-700" :title="repository.path">
							<Icon name="mdi:source-repository" :size="14" class="shrink-0 text-slate-400" />
							<span class="truncate">{{ repository.name }}</span>
						</div>
						<span v-if="repository.dirty" class="shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-700">
							{{ t("code.changedFiles", { count: repository.changedFiles }) }}
						</span>
					</div>
					<div class="mt-1.5 flex items-center gap-1.5 text-[11px] text-slate-500">
						<Icon name="mdi:source-branch-check" :size="13" class="shrink-0 text-blue-500" />
						<span>{{ t("code.currentBranchLabel") }}</span>
						<span class="min-w-0 truncate font-mono text-slate-700">{{ currentBranch(repository)?.name || repository.currentBranch || t("code.detachedHead") }}</span>
						<template v-if="repository.dirty">
							<span class="ml-auto shrink-0 font-medium text-emerald-600">+{{ repository.additions }}</span>
							<span class="shrink-0 font-medium text-red-500">-{{ repository.deletions }}</span>
						</template>
					</div>
					<div v-if="otherBranches(repository).length" class="mt-1 flex min-w-0 items-start gap-1.5 text-[10px] text-slate-400">
						<span class="shrink-0">{{ t("code.otherBranches") }}</span>
						<span class="min-w-0 break-all font-mono text-slate-500">{{ otherBranches(repository).map(branch => branch.name).join(", ") }}</span>
					</div>
				</div>
			</div>
		</n-scrollbar>
	</section>

	<ProjectBranchManagementDrawer
		v-model:show="showBranchManager"
		:repositories="branchState.repositories"
		:loading="loading"
		:load-failed="loadFailed"
		:deleting-branch-key="deletingBranchKey"
		@refresh="fetchBranches()"
		@delete-branch="confirmDeleteBranch"
	/>
</template>

<style scoped>
.ai-workspace-repository-scrollbar :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical) {
	right: 3px !important;
	width: 3px !important;
}

.ai-workspace-repository-scrollbar :deep(.n-scrollbar-rail__scrollbar) {
	width: 3px !important;
	background-color: rgba(100, 116, 139, 0.28) !important;
}

:global(.theme-dark) .ai-workspace-repositories {
	border-top-color: color-mix(in srgb, var(--border-color) 80%, transparent);
	background-color: color-mix(in srgb, var(--bg-default-color) 76%, transparent);
}
</style>
