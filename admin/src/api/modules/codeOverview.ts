import http from "@/api"
import type { CodeProjectOverview, CodeProjectSyncStatus } from "../interface/codeOverview"

export function getCodeProjectOverview(projectId: number) {
	return http.get<CodeProjectOverview>(`/code/projects/${projectId}/overview`, undefined, { timeout: 10000 })
}

export function getCodeProjectSyncStatus(projectId: number) {
	return http.get<CodeProjectSyncStatus>(`/code/projects/${projectId}/git/sync`, undefined, { timeout: 15000 })
}

export function syncCodeProject(projectId: number) {
	return http.post<CodeProjectSyncStatus>(`/code/projects/${projectId}/git/sync`, { confirm: true }, 70000)
}
