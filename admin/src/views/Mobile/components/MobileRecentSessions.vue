<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { AIProject, CodeSession } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"

defineProps<{ sessions: CodeSession[]; projects: AIProject[]; pendingCount: number }>()
const emit = defineEmits<{ open: [session: CodeSession]; showAll: [] }>()
const { t } = useI18n({ messages: mobileMessages })

function normalizedProjectPath(value?: string) {
	return (value || "").replace(/\\/g, "/").replace(/\/+$/, "")
}

function sessionProjectName(session: CodeSession, projects: AIProject[]) {
	const projectById = projects.find(project => project.id === session.projectId)
	if (projectById) return projectById.name
	const sessionPaths = [session.sourceWorkDir, session.workDir].map(normalizedProjectPath).filter(Boolean)
	return projects.find(project => {
		const projectPaths = [project.workDir, ...(project.sourceDirs || [])].map(normalizedProjectPath).filter(Boolean)
		return projectPaths.some(path => sessionPaths.includes(path))
	})?.name || t("mobile.unlinkedProject")
}

function sessionStageLabel(session: CodeSession) {
	const knownStages = ["idle", "interactive", "task_ready", "instruction_queued", "awaiting_approval", "executing", "completed", "preview_ready", "failed", "cancelled", "approval_rejected"]
	return t(`mobile.stage_${knownStages.includes(session.currentStage) ? session.currentStage : "unknown"}`)
}

function sessionStageType(session: CodeSession) {
	if (session.currentStage === "failed") return "error"
	if (["awaiting_approval", "approval_rejected", "cancelled"].includes(session.currentStage)) return "warning"
	if (["completed", "preview_ready"].includes(session.currentStage)) return "success"
	if (["executing", "instruction_queued", "interactive"].includes(session.currentStage)) return "info"
	return "default"
}

function sessionApprovalLabel(session: CodeSession) {
	return t({ manual: "mobile.approvalManual", safe_auto: "mobile.approvalSafe", full_auto: "mobile.approvalFull" }[session.approvalPolicy])
}

function formatSessionTime(value: string) {
	return new Date(value).toLocaleString(undefined, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })
}
</script>

<template>
	<section>
		<div class="my-6 flex items-center justify-between">
			<div class="flex items-center gap-2">
				<h2 class="text-xl">{{ t("mobile.recentSessions") }}</h2>
				<n-tag v-if="pendingCount" size="small" type="warning" :bordered="false">{{ t("mobile.pendingCount", { count: pendingCount }) }}</n-tag>
			</div>
			<n-button size="small" text type="primary" @click="emit('showAll')">{{ t("mobile.viewProjectSessions") }}</n-button>
		</div>
		<n-empty v-if="sessions.length === 0" size="small" :description="t('mobile.noSessions')" />
		<div v-else class="space-y-3">
			<button v-for="session in sessions" :key="session.id" class="w-full rounded-2xl border border-slate-200 bg-white p-4 text-left shadow-sm transition active:scale-[0.99]" @click="emit('open', session)">
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0 flex-1">
						<div class="truncate font-semibold text-slate-900">{{ session.title }}</div>
						<div class="mt-1.5 flex items-center gap-1.5 text-sm font-medium text-blue-600"><Icon name="mdi:folder-outline" :size="16" /><span class="truncate">{{ sessionProjectName(session, projects) }}</span></div>
						<div class="mt-1 flex items-center gap-1.5 text-xs text-slate-500"><Icon name="mdi:robot-outline" :size="14" /><span>{{ session.agentName }}</span><span v-if="session.providerModel" class="truncate">· {{ session.providerModel }}</span></div>
					</div>
					<n-tag size="small" :type="sessionStageType(session)" :bordered="false" round>{{ sessionStageLabel(session) }}</n-tag>
				</div>
				<div class="mt-3 flex items-center gap-2 text-xs text-slate-500"><Icon name="mdi:source-branch" :size="15" class="shrink-0" /><span class="min-w-0 flex-1 truncate">{{ session.workDir }}</span></div>
				<div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-slate-100 pt-3 text-xs text-slate-400">
					<span>{{ sessionApprovalLabel(session) }}</span><span v-if="session.worktreeBranch" class="max-w-full truncate">{{ session.worktreeBranch }}</span><span class="ml-auto">{{ formatSessionTime(session.createdAt) }}</span>
				</div>
			</button>
		</div>
	</section>
</template>
