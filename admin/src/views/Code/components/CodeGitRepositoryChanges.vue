<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import type { CodeGitDiffKind, CodeGitFile, CodeGitRepository, CodeGitScope } from "@/api/interface/codeGit"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

export interface GitRepositoryChangeEntry {
	repository: CodeGitRepository
	file: CodeGitFile
	kind: CodeGitDiffKind
	key: string
}

const props = defineProps<{
	repositories: CodeGitRepository[]
	entries: GitRepositoryChangeEntry[]
	scope: CodeGitScope
	selectedKey: string
	stagingKey: string
	showAdvancedOperations: boolean
}>()

defineEmits<{
	(event: "select", entry: GitRepositoryChangeEntry): void
	(event: "update-stage", payload: { entry: GitRepositoryChangeEntry; staged: boolean }): void
}>()

const { t } = useI18n({ messages: codeGitReviewMessages })
const groups = computed(() =>
	props.scope === "result"
		? [{ kind: "result" as const, label: t("code.gitResultFiles") }]
		: [
				{ kind: "staged" as const, label: t("code.gitStaged") },
				{ kind: "working" as const, label: t("code.gitChanged") },
				{ kind: "untracked" as const, label: t("code.gitUntracked") }
			]
)
const visibleRepositories = computed(() =>
	props.repositories.filter(repository => props.entries.some(entry => entry.repository.id === repository.id))
)
const entriesFor = (repositoryId: string, kind: CodeGitDiffKind | "untracked") =>
	props.entries.filter(entry => {
		if (entry.repository.id !== repositoryId) return false
		if (kind === "result") return entry.kind === "result"
		if (kind === "untracked") return entry.kind === "working" && entry.file.untracked
		if (kind === "working") return entry.kind === "working" && entry.file.changed
		return entry.kind === "staged"
	})
</script>

<template>
	<n-scrollbar class="h-full min-h-0">
		<div class="space-y-4 p-2.5">
			<section v-for="repository in visibleRepositories" :key="repository.id">
				<div class="mb-2 flex items-center gap-2 px-2">
					<Icon name="mdi:source-repository" :size="16" />
					<span class="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
						{{ repository.name }}
					</span>
					<span class="truncate text-[11px] text-slate-400">
						<template v-if="scope === 'result'">
							{{ repository.baseCommit?.slice(0, 8) }} → {{ repository.resultCommit?.slice(0, 8) }}
						</template>
						<template v-else>{{ repository.branch || t("code.gitBranchDetached") }}</template>
					</span>
				</div>
				<template v-for="group in groups" :key="group.kind">
					<div v-if="entriesFor(repository.id, group.kind).length" class="mb-2">
						<div class="px-2 py-1 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
							{{ group.label }}
						</div>
						<button
							v-for="entry in entriesFor(repository.id, group.kind)"
							:key="entry.key"
							type="button"
							class="group flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left hover:bg-slate-200/70"
							:class="selectedKey === entry.key ? 'bg-blue-50 text-blue-700' : 'text-slate-600'"
							@click="$emit('select', entry)"
						>
							<span class="w-4 shrink-0 text-center text-xs font-semibold">
								{{
									entry.kind === "result"
										? entry.file.resultStatus
										: entry.file.untracked
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
								v-if="scope === 'workspace' && showAdvancedOperations"
								text
								size="tiny"
								:loading="stagingKey === entry.key"
								@click.stop="$emit('update-stage', { entry, staged: entry.kind !== 'staged' })"
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
</template>
