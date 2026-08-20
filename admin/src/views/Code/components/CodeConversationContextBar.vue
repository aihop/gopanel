<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getCodeGitStatus } from "@/api/modules/codeGit"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	sessionId: number | null
	workDir: string
	fallbackBranch: string
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const message = useMessage()
const branches = ref<string[]>([])
const additions = ref(0)
const deletions = ref(0)
const loading = ref(false)
const loadFailed = ref(false)
let requestVersion = 0
let refreshTimer: ReturnType<typeof setInterval> | null = null

const directoryText = computed(() => props.workDir.trim() || t(loading.value ? "code.contextLoading" : "code.contextUnavailable"))
const branchText = computed(() => {
	if (branches.value.length) return branches.value.join(" · ")
	if (props.fallbackBranch.trim()) return props.fallbackBranch.trim()
	if (loading.value) return t("code.contextLoading")
	return t(loadFailed.value ? "code.contextUnavailable" : "code.noGitBranch")
})

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
		loadFailed.value = false
	} catch {
		if (version !== requestVersion) return
		loadFailed.value = true
		if (!silent) message.error(t("code.workspaceContextLoadFailed"))
	} finally {
		if (version === requestVersion) loading.value = false
	}
}

watch(
	() => props.sessionId,
	sessionId => {
		if (refreshTimer) clearInterval(refreshTimer)
		loading.value = false
		branches.value = []
		additions.value = 0
		deletions.value = 0
		loadFailed.value = false
		void loadBranches(sessionId)
		refreshTimer = sessionId ? setInterval(() => void loadBranches(sessionId, true), 10000) : null
	},
	{ immediate: true },
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
      <span class="shrink-0 font-mono text-emerald-600 dark:text-emerald-400">+{{ additions }}</span>
      <span class="shrink-0 font-mono text-rose-600 dark:text-rose-400">-{{ deletions }}</span>
    </div>
  </div>
</template>
