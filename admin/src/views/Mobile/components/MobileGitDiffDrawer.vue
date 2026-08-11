<script setup lang="ts">
import { useI18n } from "vue-i18n"
import CodeGitDiffViewer from "@/views/Code/components/CodeGitDiffViewer.vue"
import { codeGitReviewMessages } from "@/views/Code/codeGitReviewMessages"

defineProps<{
	show: boolean
	title: string
	subtitle: string
	content: string
	truncated: boolean
	loading: boolean
}>()
const emit = defineEmits<{ (event: "update:show", value: boolean): void }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="92dvh" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('code.gitReview')" closable body-content-style="padding: 0;">
			<div class="flex h-[calc(92dvh-58px)] min-h-0">
				<CodeGitDiffViewer
					:title="title"
					:subtitle="subtitle"
					:content="content"
					:truncated="truncated"
					:loading="loading"
					:empty-description="t('code.gitSelectFile')"
					:diff-empty-description="t('code.gitDiffEmpty')"
					:truncated-description="t('code.gitDiffTruncated')"
					:open-file-label="t('code.gitOpenFile')"
					:can-open-file="false"
				/>
			</div>
		</n-drawer-content>
	</n-drawer>
</template>
