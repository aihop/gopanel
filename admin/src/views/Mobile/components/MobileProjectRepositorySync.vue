<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeProjectRepositorySyncStatus, CodeProjectSyncStatus } from "@/api/interface/codeOverview"
import { getMobileProjectSyncStatus, syncMobileProject } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileRepositorySyncMessages } from "../mobileRepositorySyncMessages"

const props = defineProps<{ projectId: number | null }>()
const { t } = useI18n({ messages: mobileRepositorySyncMessages })
const dialog = useDialog()
const message = useMessage()
const state = ref<CodeProjectSyncStatus | null>(null)
const loading = ref(false)
const syncing = ref(false)
const loadError = ref(false)
let requestId = 0

const statusLabel = (status: CodeProjectRepositorySyncStatus) => t(`mobileSync.status_${status}`)
const statusType = computed(() => {
	if (state.value?.status === "synced" || state.value?.status === "local") return "success" as const
	if (state.value?.status === "behind") return "info" as const
	if (state.value?.status === "ahead") return "warning" as const
	return "error" as const
})
const summary = computed(() => {
	const repositories = state.value?.repositories || []
	return t("mobileSync.summary", {
		count: repositories.length,
		ahead: repositories.reduce((total, repository) => total + repository.ahead, 0),
		behind: repositories.reduce((total, repository) => total + repository.behind, 0)
	})
})

const load = async () => {
	if (!props.projectId || syncing.value) return
	const projectId = props.projectId
	const currentRequest = ++requestId
	if (!state.value) loading.value = true
	try {
		const result = await getMobileProjectSyncStatus(projectId)
		if (currentRequest !== requestId || projectId !== props.projectId) return
		state.value = result
		loadError.value = false
	} catch (error) {
		if (currentRequest !== requestId || projectId !== props.projectId) return
		loadError.value = true
	} finally {
		if (currentRequest === requestId) loading.value = false
	}
}

const requestSync = () => {
	if (!props.projectId || !state.value?.canSync || syncing.value) return
	dialog.warning({
		title: t("mobileSync.title"),
		content: t("mobileSync.confirm"),
		positiveText: t("mobileSync.action"),
		negativeText: t("mobileSync.cancel"),
		onPositiveClick: async () => {
			syncing.value = true
			try {
				state.value = await syncMobileProject(props.projectId as number)
				message.success(t("mobileSync.success"))
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("mobileSync.failed"))
			} finally {
				syncing.value = false
			}
		}
	})
}

watch(
	() => props.projectId,
	() => {
		requestId++
		state.value = null
		loadError.value = false
		void load()
	},
	{ immediate: true }
)
useIntervalFn(() => void load(), 10000)
</script>

<template>
	<section v-if="projectId" class="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm">
		<div class="flex items-center justify-between gap-3">
			<div class="min-w-0">
				<div class="flex items-center gap-2 text-sm font-semibold text-slate-900">
					<Icon name="mdi:source-repository" :size="18" class="text-blue-600" />
					{{ t("mobileSync.title") }}
				</div>
				<p v-if="state" class="mt-1 truncate text-xs text-slate-500">{{ summary }}</p>
			</div>
			<n-button
				size="small"
				type="primary"
				secondary
				:disabled="!state?.canSync"
				:loading="syncing"
				@click="requestSync"
			>
				{{ t("mobileSync.action") }}
			</n-button>
		</div>
		<div v-if="loading" class="flex justify-center py-3"><n-spin size="small" /></div>
		<n-alert v-else-if="loadError" class="mt-3" type="error" :show-icon="false">
			<div class="flex items-center justify-between gap-2">
				<span class="text-xs">{{ t("mobileSync.loadFailed") }}</span>
				<n-button size="tiny" @click="load">{{ t("mobileSync.retry") }}</n-button>
			</div>
		</n-alert>
		<div v-else-if="state" class="mt-3 flex items-center justify-between gap-3">
			<n-tag size="small" :type="statusType" :bordered="false">{{ statusLabel(state.status) }}</n-tag>
			<div class="min-w-0 truncate text-xs text-slate-400">
				{{
					state.repositories
						.map(repository => `${repository.name}:${statusLabel(repository.status)}`)
						.join(" · ")
				}}
			</div>
		</div>
	</section>
</template>
