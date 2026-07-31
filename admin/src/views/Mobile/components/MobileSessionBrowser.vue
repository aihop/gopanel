<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { AIProject, CodeSession } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import MobileProjectRepositorySync from "./MobileProjectRepositorySync.vue"

const props = defineProps<{
	projects: AIProject[]
	sessions: CodeSession[]
	selectedProjectId: number | null
	selectedSessionId: number
	loading: boolean
}>()
const emit = defineEmits<{
	"update:selectedProjectId": [projectId: number]
	"new-session": []
	"project-terminal": []
	"select-session": [session: CodeSession]
}>()
const { t } = useI18n({ messages: mobileMessages })

const projectStatus = (project: AIProject) => {
	const status = project.executionSummary?.status || "idle"
	return t(
		`mobile.projectStatus_${["idle", "queued", "running", "delivering", "pending_approval"].includes(status) ? status : "unknown"}`
	)
}

const sessionStatus = (session: CodeSession) => {
	if (session.status === "delivering") return t("mobile.deliveryStatus_running")
	if (session.status === "delivered") return t("mobile.deliveryStatus_completed")
	const stage = session.currentStage || "idle"
	const known = [
		"idle",
		"interactive",
		"task_ready",
		"instruction_queued",
		"awaiting_approval",
		"executing",
		"completed",
		"preview_ready",
		"failed",
		"cancelled",
		"approval_rejected"
	]
	return t(`mobile.stage_${known.includes(stage) ? stage : "unknown"}`)
}

const sessionStatusType = (session: CodeSession) => {
	if (session.status === "delivered") return "success" as const
	if (session.status === "delivering") return "info" as const
	if (session.currentStage === "failed") return "error" as const
	if (["awaiting_approval", "approval_rejected", "cancelled"].includes(session.currentStage))
		return "warning" as const
	if (["completed", "preview_ready"].includes(session.currentStage)) return "success" as const
	if (["executing", "instruction_queued", "interactive"].includes(session.currentStage)) return "info" as const
	return "default" as const
}

const formatTime = (value?: string) =>
	value
		? new Date(value).toLocaleString(undefined, {
				month: "2-digit",
				day: "2-digit",
				hour: "2-digit",
				minute: "2-digit"
			})
		: t("mobile.noActivityTime")
</script>

<template>
	<section class="space-y-4">
		<div class="grid grid-cols-2 gap-2">
			<n-button type="primary" secondary class="!h-11 !rounded-xl" @click="emit('new-session')">
				<template #icon><Icon name="mdi:robot-outline" /></template>
				{{ t("mobile.newSession") }}
			</n-button>
			<n-button secondary class="!h-11 !rounded-xl" @click="emit('project-terminal')">
				<template #icon><Icon name="mdi:console-line" /></template>
				{{ t("mobile.projectTerminal") }}
			</n-button>
		</div>
		<n-empty
			v-if="projects.length === 0"
			:description="t('mobile.noProjects')"
			class="rounded-2xl bg-white py-16"
		/>
		<div v-else class="space-y-3">
			<section
				v-for="project in projects"
				:key="project.id"
				class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
			>
				<button
					type="button"
					class="flex w-full items-center gap-3 p-4 text-left transition active:bg-slate-50"
					:class="selectedProjectId === project.id ? 'bg-blue-50/60' : ''"
					@click="emit('update:selectedProjectId', project.id)"
				>
					<span
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600"
					>
						<Icon name="mdi:folder-code-outline" :size="21" />
					</span>
					<span class="min-w-0 flex-1">
						<span class="block truncate text-sm font-semibold text-slate-900">{{ project.name }}</span>
						<span class="mt-1 flex items-center gap-1.5 text-xs text-slate-500">
							<span>{{ t("mobile.projectTaskCount", { count: project.taskCount || 0 }) }}</span>
							<span>·</span>
							<span>
								{{
									formatTime(
										project.executionSummary?.updatedAt || project.updatedAt || project.createdAt
									)
								}}
							</span>
						</span>
					</span>
					<n-tag
						size="small"
						:type="
							project.executionSummary?.status === 'running'
								? 'info'
								: project.executionSummary?.status === 'pending_approval'
									? 'warning'
									: project.executionSummary?.status === 'delivering'
										? 'success'
										: 'default'
						"
						:bordered="false"
					>
						{{ projectStatus(project) }}
					</n-tag>
					<Icon
						:name="selectedProjectId === project.id ? 'mdi:chevron-up' : 'mdi:chevron-down'"
						class="shrink-0 text-slate-400"
					/>
				</button>
				<div
					v-if="selectedProjectId === project.id"
					class="space-y-3 border-t border-slate-100 bg-slate-50/60 p-3"
				>
					<MobileProjectRepositorySync :project-id="project.id" />
					<div v-if="loading" class="flex justify-center py-10"><n-spin size="small" /></div>
					<n-empty
						v-else-if="sessions.length === 0"
						size="small"
						:description="t('mobile.noProjectSessions')"
						class="rounded-xl bg-white py-10"
					>
						<template #extra>
							<n-button size="small" type="primary" @click="emit('new-session')">
								{{ t("mobile.newSession") }}
							</n-button>
						</template>
					</n-empty>
					<div
						v-for="session in sessions"
						v-else
						:key="session.id"
						class="flex w-full min-w-0 items-center gap-3 rounded-xl border border-slate-200 bg-white p-3 text-left transition active:scale-[0.99]"
						:class="selectedSessionId === session.id ? 'border-blue-300 bg-blue-50' : ''"
					>
						<button
							type="button"
							class="flex min-w-0 flex-1 items-center gap-3 text-left"
							@click="emit('select-session', session)"
						>
							<span
								class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600"
							>
								<Icon name="mdi:robot-outline" :size="19" />
							</span>
							<span class="min-w-0 flex-1">
								<span class="block truncate text-sm font-semibold text-slate-900">
									{{ session.currentTaskTitle || session.title }}
								</span>
								<span class="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-slate-500">
									<span class="truncate">{{ session.agentName }}</span>
									<span>·</span>
									<span class="shrink-0">
										{{ formatTime(session.updatedAt || session.createdAt) }}
									</span>
								</span>
							</span>
						</button>
						<div class="flex shrink-0 flex-col items-end gap-1.5">
							<n-tag size="tiny" :type="sessionStatusType(session)" :bordered="false">
								{{ sessionStatus(session) }}
							</n-tag>
							<Icon name="mdi:chevron-right" class="text-slate-400" />
						</div>
					</div>
				</div>
			</section>
		</div>
	</section>
</template>
