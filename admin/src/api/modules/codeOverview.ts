import http from "@/api"
import type { CodeProjectOverview } from "../interface/codeOverview"

export function getCodeProjectOverview(projectId: number) {
	return http.get<CodeProjectOverview>(`/code/projects/${projectId}/overview`, undefined, { timeout: 10000 })
}
