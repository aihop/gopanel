<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeProjectRepositorySyncStatus, CodeProjectSyncStatus } from "@/api/interface/codeOverview"
import { getCodeProjectSyncStatus, syncCodeProject } from "@/api/modules/codeOverview"
import Icon from "@/components/common/Icon.vue"
import { projectOverviewMessages } from "../projectOverviewMessages"

const props = defineProps<{ projectId: number }>()
const { t } = useI18n({ messages: projectOverviewMessages })
const dialog = useDialog()
const message = useMessage()
const state = ref<CodeProjectSyncStatus | null>(null)
const loading = ref(false)
const syncing = ref(false)
const loadError = ref(false)
let pending = false
let requestId = 0

const statusType = computed(() => {
	if (state.value?.status === "synced" || state.value?.status === "local") return "success" as const
	if (state.value?.status === "behind") return "info" as const
	if (state.value?.status === "ahead") return "warning" as const
	return "error" as const
})

const statusLabel = (status: CodeProjectRepositorySyncStatus) => t(`code.repositorySyncStatus_${status}`)
const syncSummary = computed(() => {
	const repositories = state.value?.repositories || []
	const ahead = repositories.reduce((total, repository) => total + repository.ahead, 0)
	const behind = repositories.reduce((total, repository) => total + repository.behind, 0)
	return t("code.repositorySyncSummary", { count: repositories.length, ahead, behind })
})

const load = async (notify = false) => {
	if (!props.projectId || pending || syncing.value) return
	const projectId = props.projectId
	const currentRequest = ++requestId
	pending = true
	if (!state.value) loading.value = true
	try {
		const response = await getCodeProjectSyncStatus(projectId)
		if (response.code !== 0) throw new Error(response.message)
		if (currentRequest !== requestId || projectId !== props.projectId) return
		state.value = response.data
		loadError.value = false
	} catch (error) {
		if (currentRequest !== requestId || projectId !== props.projectId) return
		loadError.value = true
	} finally {
		if (currentRequest === requestId) {
			pending = false
			loading.value = false
		}
	}
}

const requestSync = () => {
	if (!state.value?.canSync || syncing.value) return
	dialog.warning({
		title: t("code.repositorySyncTitle"),
		content: t("code.repositorySyncConfirm"),
		positiveText: t("code.repositorySyncAction"),
		negativeText: t("code.repositorySyncCancel"),
		onPositiveClick: async () => {
			syncing.value = true
			try {
				const response = await syncCodeProject(props.projectId)
				if (response.code !== 0) throw new Error(response.message)
				state.value = response.data
				message.success(t("code.repositorySyncSuccess"))
			} catch (error) {
				// 错误提示由请求拦截器统一处理
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
		pending = false
		state.value = null
		loadError.value = false
		void load()
	},
	{ immediate: true }
)
useIntervalFn(() => void load(), 10000)
</script>

<template>
	<section
		class="rounded-2xl border border-slate-200 bg-white p-5 dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
	>
		<div class="flex items-center justify-between gap-3">
			<div>
				<h3 class="font-semibold">{{ t("code.repositorySyncTitle") }}</h3>
				<p v-if="state" class="mt-1 text-xs text-[var(--n-text-color-3)]">{{ syncSummary }}</p>
			</div>
			<div class="flex items-center gap-2">
				<n-tag v-if="state" :type="statusType" :bordered="false" size="small">
					{{ statusLabel(state.status) }}
				</n-tag>
				<n-button
					size="small"
					type="primary"
					secondary
					:disabled="!state?.canSync"
					:loading="syncing"
					@click="requestSync"
				>
					<template #icon><Icon name="mdi:source-pull" /></template>
					{{ t("code.repositorySyncAction") }}
				</n-button>
			</div>
		</div>
		<n-spin :show="loading">
			<n-alert v-if="loadError && !state" class="mt-4" type="error" :show-icon="false">
				<div class="flex items-center justify-between gap-3">
					<span>{{ t("code.repositorySyncLoadFailed") }}</span>
					<n-button size="tiny" @click="load(true)">{{ t("code.retry") }}</n-button>
				</div>
			</n-alert>
			<n-empty
				v-else-if="state && !state.repositories.length"
				class="py-4"
				size="small"
				:description="t('code.repositorySyncEmpty')"
			/>
			<div v-else class="mt-4 grid gap-2 md:grid-cols-2">
				<div
					v-for="repository in state?.repositories || []"
					:key="repository.path"
					class="rounded-xl bg-slate-50 p-3 dark:bg-white/5"
				>
					<div class="flex items-center justify-between gap-3">
						<span class="min-w-0 truncate text-sm font-medium">{{ repository.name }}</span>
						<n-tag size="tiny" :bordered="false">{{ statusLabel(repository.status) }}</n-tag>
					</div>
					<div class="mt-1 truncate text-xs text-[var(--n-text-color-3)]">
						{{ repository.branch }}
						<template v-if="repository.remote">
							→ {{ repository.remote }}/{{ repository.remoteBranch }}
						</template>
					</div>
					<div v-if="repository.ahead || repository.behind" class="mt-2 text-xs text-[var(--n-text-color-3)]">
						{{ t("code.repositorySyncDifference", { ahead: repository.ahead, behind: repository.behind }) }}
					</div>
				</div>
			</div>
		</n-spin>
	</section>
</template>
