<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { CodeRepositoryDeliveryResult } from "@/api/interface/codeGit"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

defineProps<{ repositories: CodeRepositoryDeliveryResult[] }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
</script>

<template>
	<div v-if="repositories.length" class="mt-2 space-y-1 text-xs">
		<p v-for="repository in repositories" :key="repository.repositoryId" class="break-words">
			{{
				t("code.gitConflictManualMerge", {
					repository: repository.repositoryName,
					branch: repository.branch,
					target: repository.targetBranch
				})
			}}
		</p>
	</div>
</template>
