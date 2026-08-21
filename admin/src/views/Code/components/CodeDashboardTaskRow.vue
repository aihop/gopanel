<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { updateAITask } from "@/api/modules/code"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import TaskApprovalAction from "./TaskApprovalAction.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"

const props = defineProps<{
	task: CodeTaskListItem
	projectName: string
	showProject?: boolean
	selected?: boolean
	archived?: boolean
	archiving?: boolean
}>()
const emit = defineEmits<{
	open: [task: CodeTaskListItem]
	archive: [task: CodeTaskListItem]
	openWorkspace: [task: CodeTaskListItem]
	renamed: [taskId: number, title: string]
	refresh: []
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()

const deliveryNeedsAttention = computed(() =>
	["failed", "partial", "conflict"].includes(props.task.summary.deliveryStatus || "")
)

// 改名就地进行：任务名是这一行唯一的身份标识，弹窗改名会把它从上下文里摘出来，
// 反而看不清自己在改哪一条。
const editing = ref(false)
const editingTitle = ref("")
const saving = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)

const startEditing = async () => {
	if (saving.value) return
	editing.value = true
	editingTitle.value = props.task.title
	await nextTick()
	titleInput.value?.focus()
	titleInput.value?.select()
}

const cancelEditing = () => {
	if (saving.value) return
	editing.value = false
	editingTitle.value = ""
}

const saveTitle = async () => {
	if (!editing.value || saving.value) return
	const title = editingTitle.value.trim()
	// 空标题和原样提交都按取消处理：清空标题只会让列表里多出一行没有名字的任务。
	if (!title || title === props.task.title) {
		cancelEditing()
		return
	}
	saving.value = true
	try {
		const response = await updateAITask(props.task.id, title)
		if (response.code !== 0) throw new Error(response.message)
		emit("renamed", props.task.id, title)
		editing.value = false
		editingTitle.value = ""
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.taskRenameFailed"))
	} finally {
		saving.value = false
	}
}
</script>

<template>
	<div
		class="dashboard-task-row group/row relative mb-0.5 flex cursor-pointer items-center gap-2 rounded-md py-1.5 pl-2 pr-2 transition-colors"
		:class="selected ? 'dashboard-task-row--selected' : ''"
		role="button"
		tabindex="0"
		@click="emit('open', task)"
		@keydown.enter.self="emit('open', task)"
		@keydown.space.self.prevent="emit('open', task)"
	>
		<span
			class="dashboard-task-row__marker absolute -left-[7px] bottom-1.5 top-1.5 w-0.5 rounded-full"
			:class="selected ? 'dashboard-task-row__marker--selected' : 'bg-transparent'"
		/>
		<TaskStatusBadge class="shrink-0" :status="task.status" compact />
		<span
			v-if="showProject !== false"
			class="max-w-[88px] shrink-0 truncate text-[11px] text-[var(--n-text-color-3)]"
			:title="projectName"
		>
			{{ projectName }}
		</span>
		<input
			v-if="editing"
			ref="titleInput"
			v-model="editingTitle"
			type="text"
			class="min-w-0 flex-1 rounded border border-[var(--primary-color)] bg-transparent px-1 py-0.5 text-[13px] tracking-[0.01em] text-[var(--n-text-color)] outline-none"
			:disabled="saving"
			@click.stop
			@keydown.stop.enter.prevent="saveTitle"
			@keydown.stop.esc.prevent="cancelEditing"
			@blur="saveTitle"
		/>
		<span
			v-else
			class="min-w-0 flex-1 truncate text-[13px] tracking-[0.01em]"
			:class="selected ? 'font-semibold text-[var(--primary-color)]' : 'font-normal text-[var(--n-text-color-2)]'"
			:title="task.title"
			@dblclick.stop="startEditing"
		>
			{{ task.title }}
		</span>
		<n-button
			v-if="!editing"
			quaternary
			circle
			size="tiny"
			:title="t('code.taskRename')"
			class="shrink-0 opacity-0 transition-opacity group-hover/row:opacity-100"
			@click.stop="startEditing"
		>
			<template #icon>
				<Icon name="mdi:pencil-outline" :size="14" />
			</template>
		</n-button>
		<n-button
			v-if="!editing"
			quaternary
			circle
			size="tiny"
			:loading="archiving"
			:title="archived ? t('code.taskUnarchive') : t('code.taskArchive')"
			class="shrink-0 opacity-0 transition-opacity group-hover/row:opacity-100"
			:class="archiving ? '!opacity-100' : ''"
			@click.stop="emit('archive', task)"
		>
			<template #icon>
				<Icon :name="archived ? 'mdi:archive-arrow-up-outline' : 'mdi:archive-arrow-down-outline'" :size="14" />
			</template>
		</n-button>
		<TaskApprovalAction class="shrink-0" :task="task" compact hide-status @click.stop @approved="emit('refresh')" />
		<Icon
			v-if="deliveryNeedsAttention"
			name="mdi:alert-circle-outline"
			:size="14"
			class="shrink-0 text-amber-500"
			:title="t('code.dashboardDeliveryAttention')"
		/>
	</div>
</template>

<style scoped>
.dashboard-task-row--selected {
	background: color-mix(in srgb, var(--primary-color) 14%, transparent);
	box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--primary-color) 24%, transparent);
}

.dashboard-task-row__marker--selected {
	background: var(--primary-color);
}

.dashboard-task-row:not(.dashboard-task-row--selected):hover {
	background: color-mix(in srgb, var(--n-text-color) 3%, transparent);
}

:global(.theme-dark) .dashboard-task-row--selected {
	background: color-mix(in srgb, var(--primary-color) 18%, transparent);
}
</style>
