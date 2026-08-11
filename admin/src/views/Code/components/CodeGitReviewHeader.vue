<script setup lang="ts">
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import type { CodeGitStatus } from "@/api/interface/codeGit"
import type { CodeGitReviewView } from "../codeGitReviewView"
import { codeGitReviewHeaderMessages } from "../codeGitReviewHeaderMessages"

defineProps<{
	status: CodeGitStatus | null
	additions: number
	deletions: number
	refreshing: boolean
}>()

const view = defineModel<CodeGitReviewView>({ required: true })
defineEmits<{ (event: "refresh"): void }>()
const { t } = useI18n({ messages: codeGitReviewHeaderMessages })
</script>

<template>
	<div class="border-b border-slate-200 p-3">
		<div class="flex items-center justify-between gap-3">
			<div>
				<div class="text-sm font-semibold text-slate-800">{{ t("code.gitReview") }}</div>
				<div v-if="status?.available" class="mt-1 flex items-center gap-2 text-xs text-slate-500">
					<span>{{ t("code.gitSummary", { files: status.files }) }}</span>
					<span class="text-emerald-600">+{{ additions }}</span>
					<span class="text-rose-500">-{{ deletions }}</span>
				</div>
			</div>
			<n-button
				circle
				quaternary
				size="small"
				:loading="refreshing"
				:title="t('code.gitRefresh')"
				@click="$emit('refresh')"
			>
				<template #icon><Icon name="mdi:refresh" :size="17" /></template>
			</n-button>
		</div>
		<n-radio-group v-model:value="view" size="small" class="mt-3 w-full">
			<n-radio-button value="changes" class="w-1/3 text-center">
				{{ t("code.gitUnsavedChanges") }}
			</n-radio-button>
			<n-radio-button value="commit" class="w-1/3 text-center">{{ t("code.gitTaskChanges") }}</n-radio-button>
			<n-radio-button value="history" class="w-1/3 text-center">{{ t("code.gitCommitHistory") }}</n-radio-button>
		</n-radio-group>
	</div>
</template>
