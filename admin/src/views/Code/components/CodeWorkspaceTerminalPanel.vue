<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import CodeTerminal from "./CodeTerminal.vue"
import ProjectNativeTerminalPanel from "./ProjectNativeTerminalPanel.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	active: boolean
	isProjectTerminalActive: boolean
	projectTerminalSessionId: number | null
	projectTerminalWorkDir: string
	projectTerminalOpening: boolean
	taskId: number | null
	sessionId: number | null
	taskTitle: string
	taskWorkDir: string
	sessionWorkDir: string
	terminalTakeoverRequested: boolean
	terminalKey: number
}>()

const emit = defineEmits<{
	openProjectTerminal: []
	reopenProjectTerminal: []
	closeProjectTerminal: []
	takeTaskTerminal: []
	taskCreated: [taskId: number]
	newSession: []
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })

const canTaskTerminal = computed(() => props.sessionId !== null || props.taskId !== null)
const taskTerminalLabel = computed(() =>
	props.taskId ? t("code.taskTerminal") : props.sessionId ? t("code.sessionTerminal") : t("code.terminalSession"),
)
const activeDirectory = computed(() => {
	if (props.isProjectTerminalActive) return props.projectTerminalWorkDir
	return props.taskWorkDir || props.sessionWorkDir || ""
})

function onProjectTabClick() {
	if (!props.isProjectTerminalActive) emit("openProjectTerminal")
}

function onTaskTabClick() {
	if (props.isProjectTerminalActive) emit("takeTaskTerminal")
}
</script>

<template>
	<div
		v-show="active"
		class="ai-workspace-terminal-panel flex min-h-0 flex-1 flex-col overflow-hidden rounded border border-slate-700 bg-[#1e1e1e] shadow-lg"
	>
			<div class="flex shrink-0 items-center gap-2 border-b border-slate-700 bg-slate-900 px-3 py-1.5">
			<button
				class="flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors"
				:class="
					isProjectTerminalActive
						? 'bg-blue-600 text-white'
						: 'text-slate-300 hover:bg-slate-800 hover:text-white'
				"
				@click="onProjectTabClick"
			>
				<Icon
					name="mdi:console-line"
					:size="14"
				/>
				<span>{{ t("code.projectTerminal") }}</span>
			</button>
			<button
				v-if="canTaskTerminal"
				class="flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors"
				:class="
					!isProjectTerminalActive
						? 'bg-blue-600 text-white'
						: 'text-slate-300 hover:bg-slate-800 hover:text-white'
				"
				@click="onTaskTabClick"
			>
				<Icon
					name="mdi:robot-outline"
					:size="14"
				/>
				<span>{{ taskTerminalLabel }}</span>
			</button>
			<div
				v-if="activeDirectory"
				class="ml-auto min-w-0 truncate text-xs text-slate-400"
				:title="activeDirectory"
			>
				{{ t("code.currentDirectory") }}: {{ activeDirectory }}
			</div>
		</div>

		<div class="min-h-0 flex-1">
			<ProjectNativeTerminalPanel
				v-if="isProjectTerminalActive"
				:session-id="projectTerminalSessionId"
				:opening="projectTerminalOpening"
				@closed="emit('closeProjectTerminal')"
				@reopen="emit('reopenProjectTerminal')"
			/>
			<CodeTerminal
				v-else-if="canTaskTerminal"
				:key="terminalKey"
				:task-id="taskId"
				:session-id="sessionId"
				:auto-take-control="terminalTakeoverRequested"
				@task-created="emit('taskCreated', $event)"
				@new-session="emit('newSession')"
			/>
		</div>
	</div>
</template>
