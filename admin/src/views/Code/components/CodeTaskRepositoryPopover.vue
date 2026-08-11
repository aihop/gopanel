<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeTaskRepositorySummary } from "@/api/interface/codeTasks"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	repositories: CodeTaskRepositorySummary[]
	branch?: string
	additions: number
	deletions: number
	changedFiles: number
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const involvedRepositories = computed(() => {
	const repositories = props.repositories.filter(repository => repository.hasDiff)
	if (repositories.length) return repositories
	if (props.repositories.length) return props.repositories
	if (!props.branch) return []
	return [{
		name: t("code.taskRepositoryFallback"), branch: props.branch,
		additions: props.additions, deletions: props.deletions,
		changedFiles: props.changedFiles, hasDiff: props.changedFiles > 0
	}]
})
</script>

<template>
	<n-popover trigger="click" placement="bottom-start" style="max-width: 480px">
		<template #trigger>
			<button type="button" class="flex items-center gap-1 hover:text-blue-500" @click.stop>
				<span class="font-medium text-emerald-600">+{{ additions }}</span>
				<span class="font-medium text-red-500">-{{ deletions }}</span>
				<span>{{ t("code.taskChangedFiles", { count: changedFiles }) }}</span>
			</button>
		</template>
		<div class="min-w-[320px] space-y-2">
			<p class="text-xs font-medium text-slate-700">{{ t("code.taskRepositoryBranches") }}</p>
			<div
				v-for="repository in involvedRepositories"
				:key="`${repository.name}:${repository.branch}`"
				class="rounded-lg bg-slate-50 px-3 py-2"
			>
				<div class="flex items-center justify-between gap-3 text-xs">
					<span class="font-medium text-slate-700">{{ repository.name }}</span>
					<span class="shrink-0">
						<span class="font-medium text-emerald-600">+{{ repository.additions }}</span>
						<span class="ml-1 font-medium text-red-500">-{{ repository.deletions }}</span>
						<span class="ml-1 text-slate-400">{{ t("code.taskChangedFiles", { count: repository.changedFiles }) }}</span>
					</span>
				</div>
				<p class="mt-1 break-all font-mono text-[11px] text-slate-500">{{ repository.branch }}</p>
			</div>
		</div>
	</n-popover>
</template>
