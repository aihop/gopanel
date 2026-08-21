<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getCodeGitStatus } from "@/api/modules/codeGit"
import type { CodeGitStatus } from "@/api/interface/codeGit"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	sessionId: number | null
	workDir: string
	fallbackBranch: string
	/** 会话是否正在执行。执行中才是变更量真正在变的时候，刷新频率跟着它走。 */
	running?: boolean
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const message = useMessage()
const branches = ref<string[]>([])
const additions = ref(0)
const deletions = ref(0)
const changedFiles = ref<ChangedFile[]>([])
const loading = ref(false)
const loadFailed = ref(false)
let requestVersion = 0
let refreshTimer: ReturnType<typeof setInterval> | null = null

type ChangedFile = {
	key: string
	repository: string
	path: string
	states: string[]
}

const directoryText = computed(() => props.workDir.trim() || t(loading.value ? "code.contextLoading" : "code.contextUnavailable"))
const branchText = computed(() => {
	if (branches.value.length) return branches.value.join(" · ")
	if (props.fallbackBranch.trim()) return props.fallbackBranch.trim()
	if (loading.value) return t("code.contextLoading")
	return t(loadFailed.value ? "code.contextUnavailable" : "code.noGitBranch")
})

const collectChangedFiles = (workspace: CodeGitStatus, result: CodeGitStatus) => {
	const files = new Map<string, ChangedFile>()
	const addFile = (repository: string, repositoryId: string, path: string, state: string) => {
		const key = `${repositoryId}:${path}`
		const existing = files.get(key)
		if (existing) {
			if (!existing.states.includes(state)) existing.states.push(state)
			return
		}
		files.set(key, { key, repository, path, states: [state] })
	}
	for (const repository of result.repositories || []) {
		for (const file of repository.files || []) {
			addFile(repository.name, repository.id, file.workspacePath || file.path, "result")
		}
	}
	for (const repository of workspace.repositories || []) {
		for (const file of repository.files || []) {
			const path = file.workspacePath || file.path
			if (file.staged) addFile(repository.name, repository.id, path, "staged")
			if (file.changed || file.untracked) addFile(repository.name, repository.id, path, "working")
		}
	}
	return [...files.values()].sort((left, right) =>
		left.repository.localeCompare(right.repository) || left.path.localeCompare(right.path),
	)
}

const loadBranches = async (sessionId: number | null, silent = false) => {
	const version = ++requestVersion
	if (!sessionId) return
	if (!silent) loading.value = true
	try {
		const [workspaceResponse, resultResponse] = await Promise.all([
			getCodeGitStatus(sessionId, "workspace"),
			getCodeGitStatus(sessionId, "result"),
		])
		if (version !== requestVersion) return
		const workspace = workspaceResponse.data
		const result = resultResponse.data
		branches.value = [...new Set((workspace.repositories || []).map(item => item.branch.trim()).filter(Boolean))]
		additions.value =
			(workspace.additions || 0) + (workspace.stagedAdditions || 0) + (result.additions || 0)
		deletions.value =
			(workspace.deletions || 0) + (workspace.stagedDeletions || 0) + (result.deletions || 0)
		changedFiles.value = collectChangedFiles(workspace, result)
		loadFailed.value = false
	} catch {
		if (version !== requestVersion) return
		loadFailed.value = true
		if (!silent) message.error(t("code.workspaceContextLoadFailed"))
	} finally {
		if (version === requestVersion) loading.value = false
	}
}

// 执行中 3 秒一轮：AI 正在写文件，+/- 停在十秒前的数字等于没显示。
// 空闲时退回 10 秒，免得对着不动的工作区一直跑 git。
const restartTimer = (sessionId: number | null) => {
	if (refreshTimer) clearInterval(refreshTimer)
	refreshTimer = sessionId
		? setInterval(() => void loadBranches(sessionId, true), props.running ? 3000 : 10000)
		: null
}

watch(
	() => props.sessionId,
	sessionId => {
		loading.value = false
		branches.value = []
		additions.value = 0
		deletions.value = 0
		changedFiles.value = []
		loadFailed.value = false
		void loadBranches(sessionId)
		restartTimer(sessionId)
	},
	{ immediate: true },
)

// 一轮执行刚起步和刚收尾都立刻补一次：收尾这一刻正是用户盯着变更量看的时候，
// 让它等下一个轮询周期才跳数就成了「写完了但没反应」。
watch(
	() => props.running,
	() => {
		restartTimer(props.sessionId)
		void loadBranches(props.sessionId, true)
	},
)
onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 px-1 pb-2 text-[11px] text-[var(--n-text-color-3)]">
    <div
      class="flex min-w-0 max-w-full items-center gap-1.5"
      :title="`${t('code.currentDirectory')}: ${directoryText}`"
    >
      <Icon
        name="mdi:folder-outline"
        :size="14"
        class="shrink-0"
      />
      <span class="shrink-0">{{ t("code.currentDirectory") }}</span>
      <span class="truncate font-mono text-[var(--n-text-color-2)]">{{ directoryText }}</span>
    </div>
    <div
      class="flex min-w-0 max-w-full items-center gap-1.5"
      :title="`${t('code.currentBranch')}: ${branchText}`"
    >
      <Icon
        name="mdi:source-branch"
        :size="14"
        class="shrink-0"
      />
      <span class="shrink-0">{{ t("code.currentBranch") }}</span>
      <span class="truncate font-mono text-[var(--n-text-color-2)]">{{ branchText }}</span>
      <n-popover
        trigger="click"
        placement="top"
        style="width: min(440px, calc(100vw - 24px)); padding: 0"
      >
        <template #trigger>
          <button
            type="button"
            class="flex shrink-0 items-center gap-1 rounded px-1 py-0.5 font-mono transition-colors hover:bg-slate-100 disabled:cursor-default disabled:hover:bg-transparent dark:hover:bg-white/10"
            :disabled="!changedFiles.length"
            :title="t('code.viewChangedFiles')"
          >
            <span class="text-emerald-600 dark:text-emerald-400">+{{ additions }}</span>
            <span class="text-rose-600 dark:text-rose-400">-{{ deletions }}</span>
          </button>
        </template>
        <div class="overflow-hidden">
          <div class="flex items-center justify-between gap-3 border-b border-slate-200 px-3 py-2 dark:border-white/10">
            <span class="font-medium text-[var(--n-text-color)]">{{ t("code.changedFileDetails") }}</span>
            <span class="shrink-0 font-mono text-xs">
              <span class="text-emerald-600 dark:text-emerald-400">+{{ additions }}</span>
              <span class="ml-1 text-rose-600 dark:text-rose-400">-{{ deletions }}</span>
            </span>
          </div>
          <div class="max-h-72 overflow-auto p-1.5">
            <div
              v-for="file in changedFiles"
              :key="file.key"
              class="flex min-w-0 items-center gap-2 rounded-lg px-2 py-1.5 hover:bg-slate-50 dark:hover:bg-white/5"
            >
              <Icon
                name="mdi:file-document-outline"
                :size="14"
                class="shrink-0 text-slate-400"
              />
              <div class="min-w-0 flex-1">
                <div
                  class="truncate font-mono text-xs text-[var(--n-text-color-2)]"
                  :title="file.path"
                >
                  {{ file.path }}
                </div>
                <div class="mt-0.5 flex items-center gap-1.5 text-[10px] text-[var(--n-text-color-3)]">
                  <span v-if="branches.length > 1">{{ file.repository }}</span>
                  <span
                    v-for="state in file.states"
                    :key="state"
                    class="rounded bg-slate-100 px-1 dark:bg-white/10"
                  >
                    {{ t(`code.changedFileState_${state}`) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </n-popover>
    </div>
  </div>
</template>
