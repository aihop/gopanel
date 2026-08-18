import type { AIProject, CodeSession } from "@/api/interface/code"
import { computed, type Ref } from "vue"

function normalizedProjectPath(value?: string) {
	return (value || "").replace(/\\/g, "/").replace(/\/+$/, "")
}

export function findMobileSessionProject(session: CodeSession, projects: AIProject[]) {
	const projectById = projects.find(item => item.id === session.projectId)
	if (projectById) return projectById
	const sessionPaths = [session.sourceWorkDir, session.workDir].map(normalizedProjectPath).filter(Boolean)
	return projects.find(project => {
		const projectPaths = [project.workDir, ...(project.sourceDirs || [])].map(normalizedProjectPath).filter(Boolean)
		return projectPaths.some(path => sessionPaths.includes(path))
	})
}

export function mobileSessionTaskTitle(session: CodeSession) {
	return session.currentTaskTitle || session.title
}

export function useMobileSessionDisplay(
	sessions: Ref<CodeSession[]>,
	selectedSessionId: Ref<number>,
	projects: Ref<AIProject[]>,
	unlinkedProject: () => string
) {
	const selectedSession = computed(() => sessions.value.find(item => item.id === selectedSessionId.value) || null)
	const selectedTaskName = computed(() =>
		selectedSession.value ? mobileSessionTaskTitle(selectedSession.value) : ""
	)
	const selectedProjectName = computed(() => {
		if (!selectedSession.value) return ""
		return findMobileSessionProject(selectedSession.value, projects.value)?.name || unlinkedProject()
	})
	return { selectedSession, selectedTaskName, selectedProjectName }
}
