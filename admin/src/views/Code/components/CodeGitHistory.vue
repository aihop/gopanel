<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { getCodeGitHistory, getCodeGitHistoryDiff } from "@/api/modules/codeGit"
import type {
	CodeGitHistory,
	CodeGitHistoryCommit,
	CodeGitHistoryRepository,
	CodeGitHistorySelection
} from "@/api/interface/codeGit"
import Icon from "@/components/common/Icon.vue"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

const props = withDefaults(defineProps<{
	sessionId: number | null
	active: boolean
	refreshKey: number
	loadHistory?: (sessionId: number) => Promise<CodeGitHistory>
	loadDiff?: (sessionId: number, repositoryId: string, commit: string) => Promise<CodeGitHistoryDiff>
	autoSelect?: boolean
}>(), { autoSelect: true })
const emit = defineEmits<{ selected: [selection: CodeGitHistorySelection | null] }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const history = ref<CodeGitHistory | null>(null)
const loading = ref(false)
const loadError = ref("")
const selectedKey = ref("")

const requestHistory = async (sessionId: number) =>
	props.loadHistory ? props.loadHistory(sessionId) : (await getCodeGitHistory(sessionId)).data
const requestDiff = async (sessionId: number, repositoryId: string, commit: string) =>
	props.loadDiff
		? props.loadDiff(sessionId, repositoryId, commit)
		: (await getCodeGitHistoryDiff(sessionId, repositoryId, commit)).data

const selectCommit = async (repository: CodeGitHistoryRepository, commit: CodeGitHistoryCommit) => {
	const key = `${repository.id}:${commit.commit}`
	selectedKey.value = key
	try {
		const response = await requestDiff(props.sessionId as number, repository.id, commit.commit)
		if (selectedKey.value !== key) return
		const repositoryLabels = repository.branch ? [repository.branch, repository.name] : [repository.name]
		emit("selected", {
			...response,
			title: commit.subject,
			subtitle: [...repositoryLabels, commit.shortCommit, commit.author].join(" · ")
		})
	} catch (error) {
		if (selectedKey.value === key) emit("selected", null)
	}
}

const loadHistory = async () => {
	if (!props.sessionId || !props.active) return
	loading.value = true
	try {
		history.value = await requestHistory(props.sessionId)
		loadError.value = ""
		const repository = history.value.repositories.find(item => item.commits.length)
		if (repository && props.autoSelect) await selectCommit(repository, repository.commits[0])
		else emit("selected", null)
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.gitHistoryLoadFailed")
		emit("selected", null)
	} finally {
		loading.value = false
	}
}

watch(
	() => [props.sessionId, props.active, props.refreshKey],
	() => void loadHistory(),
	{ immediate: true }
)
</script>

<template>
	<n-spin :show="loading" class="min-h-0 flex-1">
		<div v-if="loadError" class="p-4">
			<n-alert type="error" :title="t('code.gitHistoryLoadFailed')">{{ loadError }}</n-alert>
		</div>
		<n-empty v-else-if="history && !history.commits" :description="t('code.gitHistoryEmpty')" class="mt-16" />
		<n-scrollbar v-else class="h-full">
			<div class="space-y-4 p-2.5">
				<section v-for="repository in history?.repositories || []" :key="repository.id">
					<div class="mb-2 flex items-start gap-2 px-2 text-slate-700">
						<Icon name="mdi:source-repository" :size="16" />
						<div class="min-w-0 flex-1">
							<div class="truncate font-mono text-xs font-semibold" :title="repository.branch">
								{{ repository.branch || repository.name }}
							</div>
							<div
								v-if="repository.branch"
								class="mt-0.5 truncate text-[11px] font-normal text-slate-400"
							>
								{{ repository.name }}
							</div>
						</div>
						<span class="font-mono text-[11px] text-slate-400">{{ repository.commits.length }}</span>
					</div>
					<button
						v-for="commit in repository.commits"
						:key="commit.commit"
						type="button"
						class="mb-1 w-full rounded-lg px-2 py-2 text-left hover:bg-slate-200/70"
						:class="
							selectedKey === `${repository.id}:${commit.commit}`
								? 'bg-blue-50 text-blue-700'
								: 'text-slate-600'
						"
						@click="selectCommit(repository, commit)"
					>
						<div class="flex items-center gap-2">
							<div class="min-w-0 flex-1 truncate text-xs font-medium" :title="commit.subject">
								{{ commit.subject }}
							</div>
							<span
								v-if="repository.targetBranch"
								class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium"
								:class="commit.merged ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'"
							>
								{{
									t(commit.merged ? "code.gitHistoryMerged" : "code.gitHistoryUnmerged", {
										branch: repository.targetBranch
									})
								}}
							</span>
						</div>
						<div class="mt-1 flex items-center justify-between gap-2 text-[11px] text-slate-400">
							<span class="truncate">{{ commit.author }}</span>
							<span class="shrink-0 font-mono">{{ commit.shortCommit }}</span>
						</div>
						<div class="mt-0.5 text-[11px] text-slate-400">
							{{ new Date(commit.authoredAt).toLocaleString() }}
						</div>
					</button>
				</section>
			</div>
		</n-scrollbar>
	</n-spin>
</template>

<style scoped>
:deep(.n-spin-container),
:deep(.n-spin-content) {
	height: 100%;
}
</style>
