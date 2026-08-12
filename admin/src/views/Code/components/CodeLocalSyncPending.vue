<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeDeliveryLocalSyncRepository } from "@/api/interface/codeGit"
import { syncCodeSessionDeliveryLocal } from "@/api/modules/codeGit"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

interface LocalSyncRepository {
	repositoryName: string
	commit?: string
	localSynced: boolean
	localSyncError?: string
	localSyncCommand?: string
}

const props = defineProps<{ sessionId: number; repositories: LocalSyncRepository[] }>()
const emit = defineEmits<{ synced: []; conflict: [] }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const dialog = useDialog()
const message = useMessage()
const syncing = ref(false)
const fallbackVisible = ref(false)
const resultRepositories = ref<CodeDeliveryLocalSyncRepository[]>([])
const repositories = computed(() => (resultRepositories.value.length ? resultRepositories.value : props.repositories))
const pending = computed(() =>
	repositories.value.filter(item => !item.localSynced && (item.localSyncError || item.commit))
)
const commands = computed(() =>
	pending.value
		.map(item => item.localSyncCommand)
		.filter(Boolean)
		.join("\n")
)

watch(
	() => props.repositories.map(item => `${item.repositoryName}:${item.localSynced}`).join("|"),
	() => {
		resultRepositories.value = []
		if (!props.repositories.some(item => !item.localSynced && item.commit)) fallbackVisible.value = false
	}
)

async function syncLocal() {
	if (syncing.value || pending.value.length === 0) return
	syncing.value = true
	try {
		const response = await syncCodeSessionDeliveryLocal(props.sessionId)
		resultRepositories.value = response.data.repositories
		if (response.data.status === "conflict") {
			fallbackVisible.value = false
			message.warning(t("code.gitLocalSyncConflict"))
			emit("conflict")
			return
		}
		if (response.data.status === "completed") {
			fallbackVisible.value = false
			message.success(t("code.gitLocalSyncSuccess"))
			emit("synced")
			return
		}
		fallbackVisible.value = true
		message.warning(t("code.gitLocalSyncBlocked"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.gitLocalSyncBlocked"))
	} finally {
		syncing.value = false
	}
}

function confirmSync() {
	dialog.warning({
		title: t("code.gitLocalSyncConfirmTitle"),
		content: t("code.gitLocalSyncConfirm", { count: pending.value.length }),
		positiveText: t("code.gitLocalSyncAction"),
		negativeText: t("code.gitCancel"),
		onPositiveClick: syncLocal
	})
}

async function copyCommands() {
	if (!commands.value) return
	try {
		await navigator.clipboard.writeText(commands.value)
		message.success(t("code.gitLocalSyncCopied"))
	} catch {
		message.error(t("code.gitPushStatusFailed"))
	}
}
</script>

<template>
	<n-alert v-if="pending.length" type="info" :show-icon="false" class="mt-2">
		<div class="space-y-2">
			<p class="text-xs font-medium text-slate-700">
				{{ t("code.gitLocalSyncPending", { count: pending.length }) }}
			</p>
			<p v-if="!fallbackVisible" class="text-[11px] leading-5 text-slate-500">
				{{ t("code.gitLocalSyncActionHint") }}
			</p>
			<n-button size="tiny" type="primary" secondary :loading="syncing" @click="confirmSync">
				{{ t("code.gitLocalSyncAction") }}
			</n-button>

			<template v-if="fallbackVisible">
				<p class="text-[11px] leading-5 text-slate-500">
					{{ t("code.gitLocalSyncHint") }}
				</p>
				<ul class="space-y-1">
					<li v-for="item in pending" :key="item.repositoryName" class="text-[11px] leading-5 text-slate-500">
						<span class="font-medium text-slate-600">{{ item.repositoryName }}</span>
						<span class="mx-1 text-slate-300">·</span>
						<span>{{ item.localSyncError || t("code.gitLocalSyncUnknownReason") }}</span>
					</li>
				</ul>
				<pre
					v-if="commands"
					class="overflow-x-auto whitespace-pre rounded-lg bg-slate-900 p-2 font-mono text-[11px] leading-5 text-slate-100"
					>{{ commands }}</pre
				>
				<n-button v-if="commands" size="tiny" secondary @click="copyCommands">
					{{ t("code.gitLocalSyncCopy") }}
				</n-button>
			</template>
		</div>
	</n-alert>
</template>
