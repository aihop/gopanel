<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { checkCodeSessionGitSync, syncCodeSessionGitRepository } from "@/api/modules/codeGit"
import type { CodeSessionGitSyncRepository, CodeSessionGitSyncStatus } from "@/api/interface/codeGit"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

const props = defineProps<{ sessionId: number; disabled: boolean }>()
const emit = defineEmits<{ (event: "synced"): void }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const dialog = useDialog()
const message = useMessage()
const loading = ref(false)
const syncingId = ref("")
const loadError = ref("")
const result = ref<CodeSessionGitSyncStatus | null>(null)

const hasResult = computed(() => Boolean(result.value?.repositories.length))
const statusLabel = (repository: CodeSessionGitSyncRepository) =>
	t(`code.gitSyncStatus_${repository.status}`, { count: repository.behind })
const reasonLabel = (repository: CodeSessionGitSyncRepository) =>
	repository.reason ? t(`code.gitSyncReason_${repository.reason}`) : ""
const statusType = (repository: CodeSessionGitSyncRepository) => {
	if (repository.status === "synced" || repository.updated) return "success" as const
	if (repository.status === "integrated") return "warning" as const
	if (repository.status === "behind") return "info" as const
	if (["dirty", "diverged", "remote_behind", "blocked"].includes(repository.status)) return "warning" as const
	return "default" as const
}

const checkUpdates = async () => {
	if (loading.value || syncingId.value || props.disabled) return
	loading.value = true
	loadError.value = ""
	try {
		result.value = (await checkCodeSessionGitSync(props.sessionId)).data
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.gitSyncCheckFailed")
		message.error(loadError.value)
	} finally {
		loading.value = false
	}
}

const syncRepository = (repository: CodeSessionGitSyncRepository) => {
	if (!repository.canSync || props.disabled || syncingId.value) return
	dialog.warning({
		title: t("code.gitSyncTitle"),
		content: t(repository.status === "integrated" ? "code.gitSyncIntegratedConfirm" : "code.gitSyncConfirm", {
			repository: repository.name,
			count: repository.behind
		}),
		positiveText: t("code.gitSyncAction"),
		negativeText: t("code.gitCancel"),
		onPositiveClick: async () => {
			syncingId.value = repository.id
			try {
				result.value = (await syncCodeSessionGitRepository(props.sessionId, repository.id)).data
				message.success(
					t(repository.status === "integrated" ? "code.gitSyncIntegratedSuccess" : "code.gitSyncSuccess", {
						repository: repository.name
					})
				)
				emit("synced")
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("code.gitSyncFailed"))
				syncingId.value = ""
				await checkUpdates()
			} finally {
				syncingId.value = ""
			}
		}
	})
}

watch(
	() => props.sessionId,
	() => {
		result.value = null
		loadError.value = ""
	}
)
</script>

<template>
	<div class="space-y-2 border-b border-slate-200 p-3">
		<div class="flex items-center justify-between gap-2">
			<div>
				<div class="text-xs font-semibold text-slate-700">{{ t("code.gitSyncPanel") }}</div>
				<div class="mt-0.5 text-[11px] text-slate-400">{{ t("code.gitSyncHint") }}</div>
			</div>
			<n-button
				size="tiny"
				secondary
				:loading="loading"
				:disabled="disabled || Boolean(syncingId)"
				@click="checkUpdates"
			>
				{{ t(hasResult ? "code.gitSyncCheckAgain" : "code.gitSyncCheck") }}
			</n-button>
		</div>
		<n-alert v-if="loadError" type="error" :show-icon="false">
			<div class="text-xs">{{ loadError }}</div>
		</n-alert>
		<div
			v-for="repository in result?.repositories || []"
			:key="repository.id"
			class="rounded-lg border border-slate-200 bg-white p-2"
		>
			<div class="flex items-center gap-2">
				<div class="min-w-0 flex-1">
					<div class="truncate text-xs font-medium text-slate-700">{{ repository.name }}</div>
					<div class="truncate text-[11px] text-slate-400">
						{{
							repository.remote
								? `${repository.remote}/${repository.remoteBranch}`
								: t("code.gitSyncLocalOnly")
						}}
					</div>
				</div>
				<n-tag size="small" :type="statusType(repository)" :bordered="false">
					{{ statusLabel(repository) }}
				</n-tag>
			</div>
			<div v-if="reasonLabel(repository)" class="mt-1 text-[11px] leading-4 text-slate-500">
				{{ reasonLabel(repository) }}
			</div>
			<n-button
				v-if="repository.canSync"
				class="mt-2"
				size="tiny"
				type="primary"
				secondary
				block
				:loading="syncingId === repository.id"
				:disabled="disabled || Boolean(syncingId)"
				@click="syncRepository(repository)"
			>
				{{
					t(repository.status === "integrated" ? "code.gitSyncFinish" : "code.gitSyncCommits", {
						count: repository.behind
					})
				}}
			</n-button>
		</div>
	</div>
</template>
