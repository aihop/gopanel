<script setup lang="ts">
import type { AIProject, CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import { mobileAlignmentMessages } from "@/i18n/locales/mobileAlignment"
import { useI18n } from "vue-i18n"
import MobileRecentTasks from "./MobileRecentTasks.vue"
import MobileSessionBrowser from "./MobileSessionBrowser.vue"
import MobileTerminal from "./MobileTerminal.vue"

defineProps<{
	remoteNode: boolean
	taskDetail: boolean
	projects: AIProject[]
	sessions: CodeSession[]
	selectedProjectId: number | null
	selectedSessionId: number
	selectedTaskId: number
	sessionsLoading: boolean
	projectTerminalSession: HostTerminalSession | null
	projectTerminalProject: AIProject | null
	selectedSession: CodeSession | null
	selectedTaskName: string
	selectedProjectName: string
}>()
const emit = defineEmits<{
	"new-session": []
	"project-terminal": []
	"select-project": [projectId: number]
	"select-session": [session: CodeSession]
	"open-task": [task: CodeTaskListItem]
	back: []
	"open-files": []
	"open-status": []
	renamed: []
}>()
const { t } = useI18n({ messages: mobileAlignmentMessages })
</script>

<template>
	<div :class="taskDetail ? '' : 'space-y-4'">
		<n-alert v-if="remoteNode" type="info" :title="t('mobile.remoteCodeUnavailable')">
			{{ t("mobile.remoteCodeControllerHint") }}
		</n-alert>
		<MobileSessionBrowser
			v-else-if="!taskDetail"
			:projects="projects"
			:sessions="sessions"
			:selected-project-id="selectedProjectId"
			:selected-session-id="selectedSessionId"
			:loading="sessionsLoading"
			@update:selected-project-id="emit('select-project', $event)"
			@new-session="emit('new-session')"
			@project-terminal="emit('project-terminal')"
			@select-session="emit('select-session', $event)"
		>
			<template #tasks>
				<MobileRecentTasks
					:projects="projects"
					:selected-task-id="selectedTaskId"
					@open="emit('open-task', $event)"
				/>
			</template>
		</MobileSessionBrowser>
		<MobileTerminal
			v-if="projectTerminalSession && projectTerminalProject"
			:session-id="projectTerminalSession.id"
			:task-name="t('mobile.projectTerminal')"
			:project-name="projectTerminalProject.name"
			mode="native"
			@back="emit('back')"
		/>
		<MobileTerminal
			v-else-if="selectedSession"
			:session-id="selectedSessionId"
			:task-name="selectedTaskName"
			:project-name="selectedProjectName"
			@back="emit('back')"
			@open-files="emit('open-files')"
			@open-status="emit('open-status')"
			@renamed="emit('renamed')"
		/>
	</div>
</template>
