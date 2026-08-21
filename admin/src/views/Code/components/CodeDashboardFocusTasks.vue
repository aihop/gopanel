<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { AIProject } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { updateAITask } from "@/api/modules/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { codeDashboardRecentStatus } from "../codeDashboardBuckets"
import { codeProjectColor } from "../projectColor"
import Icon from "@/components/common/Icon.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"

const props = defineProps<{ projects: AIProject[]; tasks: CodeTaskListItem[]; selectedTaskId: number | null }>()
const emit = defineEmits<{
	select: [task: CodeTaskListItem]
	renamed: [taskId: number, title: string]
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()
const projectsById = computed(() => new Map(props.projects.map(project => [project.id, project])))
const editingTaskId = ref<number | null>(null)
const editingTitle = ref("")
const savingTaskId = ref<number | null>(null)
const titleInput = ref<HTMLInputElement | null>(null)
const setTitleInput = (element: unknown) => {
	titleInput.value = element instanceof HTMLInputElement ? element : null
}
const projectStatusLabel = (projectId: number) => {
	const status = projectsById.value.get(projectId)?.executionSummary.status || "idle"
	const key = status === "pending_approval" ? "pendingApproval" : status
	return t(`code.projectStatus_${key}`)
}

const startEditing = async (task: CodeTaskListItem) => {
	if (savingTaskId.value) return
	editingTaskId.value = task.id
	editingTitle.value = task.title
	await nextTick()
	titleInput.value?.focus()
	titleInput.value?.select()
}

const cancelEditing = () => {
	if (savingTaskId.value) return
	editingTaskId.value = null
	editingTitle.value = ""
}

const saveTitle = async (task: CodeTaskListItem) => {
	if (editingTaskId.value !== task.id || savingTaskId.value) return
	const title = editingTitle.value.trim()
	if (!title || title === task.title) {
		cancelEditing()
		return
	}
	savingTaskId.value = task.id
	try {
		const response = await updateAITask(task.id, title)
		if (response.code !== 0) throw new Error(response.message)
		emit("renamed", task.id, title)
		editingTaskId.value = null
		editingTitle.value = ""
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.taskRenameFailed"))
	} finally {
		savingTaskId.value = null
	}
}
</script>

<template>
	<section class="border-b border-slate-200/70 px-3 pb-3 pt-4 dark:border-white/10">
		<div class="mb-2 flex items-center gap-2 px-2">
			<Icon name="mdi:clock-outline" :size="14" class="shrink-0 text-[var(--n-text-color-3)] opacity-70" />
			<span class="text-xs font-medium tracking-[0.04em] text-[var(--n-text-color-3)]">
				{{ t("code.dashboardRecentTitle") }}
			</span>
		</div>
		<div class="max-h-60 space-y-0.5 overflow-y-auto">
			<div
				v-for="task in tasks"
				:key="task.id"
				class="recent-task flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
				:class="task.id === selectedTaskId ? 'recent-task--active' : 'hover:bg-[var(--n-color-embedded)]'"
				role="button"
				tabindex="0"
				@click="emit('select', task)"
				@keydown.enter.self="editingTaskId !== task.id && emit('select', task)"
				@keydown.space.self.prevent="editingTaskId !== task.id && emit('select', task)"
			>
				<n-tooltip>
					<template #trigger>
						<span
							class="h-2.5 w-2.5 shrink-0 rounded-[2px]"
							:style="{
								backgroundColor: codeProjectColor(task.projectId),
								boxShadow: `0 0 0 1px ${codeProjectColor(task.projectId)}38`
							}"
						/>
					</template>
					<span class="font-medium">
						{{ projectsById.get(task.projectId)?.name || t("code.projectFallback") }}
					</span>
					<span class="ml-1 opacity-70">· {{ projectStatusLabel(task.projectId) }}</span>
				</n-tooltip>
				<input
					v-if="editingTaskId === task.id"
					:ref="setTitleInput"
					v-model="editingTitle"
					type="text"
					class="min-w-0 flex-1 rounded border border-[var(--n-primary-color)] bg-transparent px-1 py-0.5 text-xs font-medium text-[var(--n-text-color)] outline-none"
					:disabled="savingTaskId === task.id"
					@click.stop
					@dblclick.stop
					@keydown.enter.stop.prevent="saveTitle(task)"
					@keydown.esc.stop.prevent="cancelEditing"
					@blur="saveTitle(task)"
				/>
				<span
					v-else
					class="min-w-0 flex-1 truncate text-xs font-medium text-[var(--n-text-color)]"
					:title="task.title"
					@dblclick.stop="startEditing(task)"
				>
					{{ task.title }}
				</span>
				<TaskStatusBadge :status="codeDashboardRecentStatus(task)" compact />
			</div>
		</div>
	</section>
</template>

<style scoped>
.recent-task--active {
	background: color-mix(in srgb, var(--primary-color) 10%, transparent);
	box-shadow: inset 2px 0 0 color-mix(in srgb, var(--primary-color) 76%, transparent);
}

.recent-task:not(.recent-task--active):hover {
	background: color-mix(in srgb, var(--n-text-color) 3%, transparent);
}
</style>
