<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { CodeRuntimeProgress } from "@/api/interface/code"
import { codeTerminalMessages } from "../codeTerminalMessages"

defineProps<{ progress: CodeRuntimeProgress }>()
const { t } = useI18n({ messages: codeTerminalMessages })
</script>

<template>
	<div class="flex min-w-0 items-center gap-2 text-xs text-slate-300">
		<span v-if="progress.totalSteps" class="shrink-0 font-medium text-sky-300">
			{{ t("code.runtimeStep", { current: progress.currentStep, total: progress.totalSteps }) }}
		</span>
		<span v-if="progress.totalSteps && progress.stepTitle" class="truncate text-slate-400">
			{{ progress.stepTitle }}
		</span>
		<span v-if="progress.changedFiles" class="shrink-0 text-emerald-300">
			{{ t("code.runtimeChangedFiles", { count: progress.changedFiles }) }}
		</span>
		<span v-if="progress.additions || progress.deletions" class="hidden shrink-0 sm:inline">
			<span class="text-emerald-400">+{{ progress.additions }}</span>
			<span class="ml-1 text-rose-400">-{{ progress.deletions }}</span>
		</span>
	</div>
</template>
