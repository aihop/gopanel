<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeSession } from "@/api/interface/code"
import type { CodeDeliveryJob, CodeGitStatus } from "@/api/interface/codeGit"
import {
	checkMobileSessionGitSync,
	confirmMobileManualDeliveryConflict,
	syncMobileSessionGitRepository
} from "@/api/modules/mobile"
import CodeConflictManualMerge from "@/views/Code/components/CodeConflictManualMerge.vue"
import CodeDeliveryFacts from "@/views/Code/components/CodeDeliveryFacts.vue"
import CodeSessionRepositorySync from "@/views/Code/components/CodeSessionRepositorySync.vue"
import { codeGitReviewMessages } from "@/views/Code/codeGitReviewMessages"
import { mobileTaskDeliveryMessages } from "../mobileTaskDeliveryMessages"
import MobileDeliveryPush from "./MobileDeliveryPush.vue"
import MobileTaskDeliveryButton from "./MobileTaskDeliveryButton.vue"

const props = defineProps<{
	session: CodeSession
	delivery?: CodeDeliveryJob | null
	status: CodeGitStatus | null
	saving: boolean
	active: boolean
	revision: number
}>()
const emit = defineEmits<{
	(event: "save"): void
	(event: "resolve"): void
	(event: "synced"): void
	(event: "updated"): void
}>()
const commitMessage = defineModel<string>("message", { required: true })
const advanced = defineModel<boolean>("advanced", { required: true })
const messages = {
	zh: { code: codeGitReviewMessages.zh.code, mobile: mobileTaskDeliveryMessages.zh.mobile },
	en: { code: codeGitReviewMessages.en.code, mobile: mobileTaskDeliveryMessages.en.mobile }
}
const { t } = useI18n({ messages })

const deliveryActive = computed(() => ["queued", "running"].includes(props.delivery?.status || ""))
const conflictRepositories = computed(() => {
	if (props.delivery?.status !== "conflict") return []
	return (props.delivery.repositories || []).filter(repository => repository.status === "conflict")
})
const hasChanges = computed(() => Boolean(props.status?.files))
const deliveryConflict = computed(() => props.delivery?.status === "conflict")
const isolatedRepositories = computed(() =>
	(props.status?.repositories || []).filter(repository => repository.isolated)
)
const deliveryLabel = computed(() =>
	props.session.isolationMode === "multi_worktree"
		? t("code.gitMultiWorktree", { count: isolatedRepositories.value.length })
		: t("code.gitWorktreeBranch", { branch: props.session.worktreeBranch || "-" })
)
const deliveryStatusLabel = computed(() => {
	if (!props.delivery) return ""
	if (props.delivery.status === "completed" && props.delivery.hasUncommittedChanges) {
		return t("code.gitDeliveryStatus_unsaved")
	}
	if (props.delivery.status === "completed" && props.delivery.hasPendingCommits) {
		return t("code.gitDeliveryStatus_pending")
	}
	return t(`code.gitDeliveryStatus_${props.delivery.status}`, {
		position: props.delivery.queuePosition,
		progress: props.delivery.progress
	})
})

const checkSync = (sessionId: number) => checkMobileSessionGitSync(sessionId)
const runSync = (sessionId: number, repositoryId: string) => syncMobileSessionGitRepository(sessionId, repositoryId)
</script>

<template>
	<div class="max-h-[46dvh] overflow-y-auto border-b border-slate-200 bg-slate-50/70">
		<CodeSessionRepositorySync
			:session-id="session.id"
			:disabled="deliveryActive"
			:check="checkSync"
			:sync="runSync"
			@synced="emit('synced')"
		/>
		<div class="space-y-3 p-3">
			<div class="truncate text-xs text-slate-500" :title="deliveryLabel">{{ deliveryLabel }}</div>
			<n-alert
				v-if="delivery"
				:type="
					delivery.status === 'failed' || delivery.status === 'conflict' || delivery.status === 'partial'
						? 'error'
						: delivery.status === 'completed'
							? 'success'
							: 'info'
				"
				:show-icon="false"
			>
				<div class="flex items-center justify-between gap-2 text-xs">
					<span>{{ deliveryStatusLabel }}</span>
					<span v-if="deliveryActive">{{ t(`code.gitDeliveryStage_${delivery.stage}`) }}</span>
				</div>
				<n-progress
					v-if="deliveryActive"
					class="mt-2"
					type="line"
					:percentage="delivery.progress"
					:show-indicator="false"
				/>
				<div v-if="delivery.errorMessage && !conflictRepositories.length" class="mt-2 break-words text-xs">
					{{ delivery.errorMessage }}
				</div>
				<CodeConflictManualMerge
					:repositories="conflictRepositories"
					:session-id="session.id"
					:confirm-manual="confirmMobileManualDeliveryConflict"
					@resolve="emit('resolve')"
					@completed="emit('updated')"
				/>
				<CodeDeliveryFacts :facts="delivery.facts" :job-status="delivery.status" />
			</n-alert>
			<n-input
				v-model:value="commitMessage"
				size="large"
				:placeholder="t('code.gitSavePlaceholder')"
				:disabled="saving || deliveryActive || deliveryConflict"
				@keyup.enter="emit('save')"
			/>
			<n-button
				block
				size="large"
				type="primary"
				class="!rounded-xl"
				:disabled="!hasChanges || deliveryActive || deliveryConflict"
				:loading="saving"
				@click="emit('save')"
			>
				{{ t("code.gitSave") }}
			</n-button>
			<p class="text-[11px] leading-5 text-slate-500">
				{{ t(hasChanges ? "code.gitSaveHint" : "code.gitMergeReady") }}
			</p>
			<n-button text size="small" @click="advanced = !advanced">
				{{ t(advanced ? "code.gitAdvancedHide" : "code.gitAdvancedShow") }}
			</n-button>
			<MobileTaskDeliveryButton
				:session="session"
				:delivery="delivery"
				:active="active"
				:revision="String(revision)"
				block
				@updated="emit('updated')"
			/>
			<MobileDeliveryPush
				:session-id="session.id"
				:active="active"
				:refresh-key="`${delivery?.status || ''}:${delivery?.resultCommit || ''}:${revision}`"
			/>
		</div>
	</div>
</template>
