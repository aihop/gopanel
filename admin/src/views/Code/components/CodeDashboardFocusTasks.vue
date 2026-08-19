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
  <section class="border-b border-slate-200/70 px-3 py-3 dark:border-white/10">
    <div class="mb-2 flex items-center justify-between gap-2 px-1">
      <div class="flex min-w-0 items-center gap-2 text-xs font-semibold text-[var(--n-text-color-2)]">
        <Icon
          name="mdi:history"
          :size="15"
          class="shrink-0 text-blue-500"
        />
        <span class="truncate">{{ t("code.dashboardRecentTitle") }}</span>
        <span class="rounded-full bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-500">
          {{ tasks.length }}
        </span>
      </div>
    </div>
    <div class="max-h-60 space-y-1 overflow-y-auto">
      <div
        v-for="task in tasks"
        :key="task.id"
        class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors"
        :class="
          task.id === selectedTaskId ? 'recent-task--active' : 'hover:bg-[var(--n-color-embedded)]'
        "
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
                boxShadow: `0 0 0 1px ${codeProjectColor(task.projectId)}38`,
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
        >
        <span
          v-else
          class="min-w-0 flex-1 truncate text-xs font-medium text-[var(--n-text-color)]"
          :title="task.title"
          @dblclick.stop="startEditing(task)"
        >
          {{ task.title }}
        </span>
        <TaskStatusBadge
          :status="codeDashboardRecentStatus(task)"
          compact
        />
      </div>
    </div>
  </section>
</template>

<style scoped>
/*
	选中态原本用的是 hover 那个 --n-color-embedded，两者同色，等于没做。
	这里换成主色的低透明度浅底：亮色主题下是一层淡蓝，暗色主题下也够分辨，
	而且跟着 naive 的主题色走，换皮肤不用再改。
	选中行不再叠 hover——底色本来就比 hover 重，再变一次只会闪。
*/
.recent-task--active {
	background: color-mix(in srgb, var(--n-primary-color) 12%, transparent);
}
</style>
