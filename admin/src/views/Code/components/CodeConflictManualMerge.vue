<script setup lang="ts">
import { ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeDeliveryJob, CodeRepositoryDeliveryResult } from "@/api/interface/codeGit"
import { confirmManualCodeDeliveryConflict } from "@/api/modules/codeGit"
import { codeConflictManualMergeMessages } from "../codeConflictManualMergeMessages"

const props = defineProps<{
	repositories: CodeRepositoryDeliveryResult[]
	sessionId?: number | null
	confirmManual?: (sessionId: number) => Promise<CodeDeliveryJob>
}>()
const emit = defineEmits<{ (event: "resolve"): void; (event: "completed"): void }>()
const { t } = useI18n({ messages: codeConflictManualMergeMessages })
const message = useMessage()
const confirming = ref(false)

const confirmManualMerge = async () => {
	if (!props.sessionId || confirming.value) return
	confirming.value = true
	try {
		if (props.confirmManual) await props.confirmManual(props.sessionId)
		else await confirmManualCodeDeliveryConflict(props.sessionId)
		message.success(t("code.conflictManualConfirmed"))
		emit("completed")
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.conflictManualConfirmFailed"))
	} finally {
		confirming.value = false
	}
}
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
		<n-button size="tiny" type="primary" ghost @click="emit('resolve')">
			{{ t("code.conflictResolveOnline") }}
		</n-button>
		<n-button
			v-if="sessionId"
			size="tiny"
			quaternary
			:loading="confirming"
			:title="t('code.conflictManualConfirmHint')"
			@click="confirmManualMerge"
		>
			{{ t("code.conflictManualConfirm") }}
		</n-button>
	</div>
</template>
