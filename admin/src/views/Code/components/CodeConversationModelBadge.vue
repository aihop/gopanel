<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { getCodexRuntimeState } from "@/api/modules/code"
import type { CodeExecutionRun, CodeSession, CodexRuntimeState } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	sessionId: number | null
	session: CodeSession | null
	runs: CodeExecutionRun[]
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const runtime = ref<CodexRuntimeState | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null
let requestVersion = 0
const latestRun = computed(() => props.runs.find(item => item.model.trim()) || null)
const model = computed(
	() => runtime.value?.model.trim() || props.session?.providerModel?.trim() || latestRun.value?.model.trim() || props.session?.agentName || t("code.modelUnavailable"),
)
const reasoningEffort = computed(() => {
	const value = runtime.value?.reasoningEffort?.trim().toLowerCase()
	return value ? t(`code.reasoningEffort_${value}`) : t("code.reasoningEffort_default")
})

const loadRuntime = async (sessionId: number) => {
	const version = ++requestVersion
	try {
		const response = await getCodexRuntimeState(sessionId)
		if (version === requestVersion) runtime.value = response.data
	} catch {
		if (version === requestVersion) runtime.value = null
	}
}

watch(
	() => props.sessionId,
	sessionId => {
		if (refreshTimer) clearInterval(refreshTimer)
		refreshTimer = null
		requestVersion++
		runtime.value = null
		if (!sessionId) return
		void loadRuntime(sessionId)
		refreshTimer = setInterval(() => void loadRuntime(sessionId), 3000)
	},
	{ immediate: true },
)
onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <div
    class="pointer-events-none absolute bottom-1.5 right-2 flex max-w-[calc(100%-1rem)] items-center gap-1 text-[10px] text-[var(--n-text-color-3)]"
    :title="`${t('code.currentModel')}: ${model} · ${t('code.reasoningEffort')}: ${reasoningEffort}`"
  >
    <Icon
      name="mdi:brain"
      :size="12"
      class="shrink-0"
    />
    <span class="truncate font-mono">{{ model }}</span>
    <span class="shrink-0">· {{ reasoningEffort }}</span>
  </div>
</template>
