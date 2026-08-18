<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { AIProject } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { getMobileTasks, setMobileTaskArchived } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileAlignmentMessages } from "@/i18n/locales/mobileAlignment"
import { mobileTaskDisplayStatus, sortMobileRecentTasks } from "../mobileRecentTaskOrder"

const props = defineProps<{ projects: AIProject[]; selectedTaskId: number }>()
const emit = defineEmits<{ open: [task: CodeTaskListItem] }>()
const { t } = useI18n({ messages: mobileAlignmentMessages })
const message = useMessage()
const tasks = ref<CodeTaskListItem[]>([])
const loading = ref(false)
const loadError = ref("")
const archived = ref(false)
const archivingId = ref(0)
const projectsById = computed(() => new Map(props.projects.map(project => [project.id, project.name])))
const sortedTasks = computed(() => sortMobileRecentTasks(tasks.value))

function displayStatus(task: CodeTaskListItem) {
	return mobileTaskDisplayStatus(task)
}

function statusType(task: CodeTaskListItem) {
	const status = displayStatus(task)
	if (["completed"].includes(status)) return "success" as const
	if (["failed"].includes(status)) return "error" as const
	if (["pending_approval", "awaiting_approval", "approval_rejected", "cancelled"].includes(status))
		return "warning" as const
	if (
		["queued", "running", "delivering", "active", "interactive", "instruction_queued", "executing"].includes(status)
	)
		return "info" as const
	return "default" as const
}

function statusLabel(task: CodeTaskListItem) {
	const status = displayStatus(task)
	const known = [
		"active",
		"idle",
		"interactive",
		"task_ready",
		"instruction_queued",
		"awaiting_approval",
		"approval_rejected",
		"executing",
		"preview_ready",
		"queued",
		"running",
		"pending_approval",
		"delivering",
		"completed",
		"failed",
		"cancelled"
	]
	return t(`mobile.taskStatus_${known.includes(status) ? status : "unknown"}`)
}

function formatTime(value?: string) {
	if (!value) return "-"
	return new Date(value).toLocaleString(undefined, {
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit"
	})
}

async function load() {
	loading.value = true
	try {
		tasks.value = (await getMobileTasks(archived.value)).items
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.taskListLoadFailed")
	} finally {
		loading.value = false
	}
}

async function toggleArchived(task: CodeTaskListItem) {
	archivingId.value = task.id
	try {
		await setMobileTaskArchived(task.id, !archived.value)
		message.success(t(archived.value ? "mobile.taskRestored" : "mobile.taskArchived"))
		await load()
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.taskArchiveFailed"))
	} finally {
		archivingId.value = 0
	}
}

async function toggleView() {
	archived.value = !archived.value
	await load()
}

onMounted(load)
</script>

<template>
	<section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
		<header class="flex items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
			<div class="flex min-w-0 items-center gap-2 font-semibold text-slate-900">
				<Icon :name="archived ? 'mdi:archive-outline' : 'mdi:history'" class="text-blue-600" />
				<span>{{ t(archived ? "mobile.archivedTasks" : "mobile.recentTasks") }}</span>
			</div>
			<div class="flex items-center gap-1">
				<n-button size="tiny" quaternary @click="toggleView">
					{{ t(archived ? "mobile.activeTasks" : "mobile.archivedTasks") }}
				</n-button>
				<n-button circle quaternary size="tiny" :loading="loading" @click="load">
					<template #icon><Icon name="mdi:refresh" /></template>
				</n-button>
			</div>
		</header>
		<n-alert v-if="loadError" type="error" :show-icon="false" class="m-3">
			<div class="flex items-center justify-between gap-3">
				<span>{{ loadError }}</span>
				<n-button text type="primary" @click="load">{{ t("mobile.retry") }}</n-button>
			</div>
		</n-alert>
		<n-spin v-else :show="loading">
			<n-empty
				v-if="!loading && !tasks.length"
				size="small"
				:description="t(archived ? 'mobile.noArchivedTasks' : 'mobile.noRecentTasks')"
				class="py-8"
			/>
			<div v-else class="divide-y divide-slate-100">
				<article
					v-for="task in sortedTasks"
					:key="task.id"
					class="relative flex items-center gap-3 px-4 py-3 transition-colors"
					:class="task.id === selectedTaskId ? 'bg-blue-50 ring-1 ring-inset ring-blue-200' : ''"
				>
					<span
						v-if="task.id === selectedTaskId"
						class="absolute inset-y-2 left-0 w-1 rounded-r bg-blue-600"
					/>
					<button
						type="button"
						class="min-w-0 flex-1 rounded-lg text-left outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
						:aria-current="task.id === selectedTaskId ? 'true' : undefined"
						@click="emit('open', task)"
					>
						<div class="flex items-center gap-2">
							<span class="truncate text-sm font-semibold text-slate-900">{{ task.title }}</span>
							<n-tag size="tiny" :type="statusType(task)" :bordered="false">
								{{ statusLabel(task) }}
							</n-tag>
						</div>
						<div class="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-slate-500">
							<span class="truncate">
								{{ projectsById.get(task.projectId) || t("mobile.unlinkedProject") }}
							</span>
							<span>·</span>
							<span class="shrink-0">
								{{ formatTime(task.summary.lastActivityAt || task.updatedAt || task.createdAt) }}
							</span>
						</div>
					</button>
					<n-button
						circle
						quaternary
						size="small"
						:loading="archivingId === task.id"
						:title="t(archived ? 'mobile.restoreTask' : 'mobile.archiveTask')"
						@click="toggleArchived(task)"
					>
						<template #icon>
							<Icon
								:name="archived ? 'mdi:archive-arrow-up-outline' : 'mdi:archive-arrow-down-outline'"
							/>
						</template>
					</n-button>
				</article>
			</div>
		</n-spin>
	</section>
</template>
