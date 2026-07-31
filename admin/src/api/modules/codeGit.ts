import http from "@/api"
import type { CodeGitDeliveryResult, CodeGitDiff, CodeGitDiffKind, CodeGitStatus } from "../interface/codeGit"

export function getCodeGitStatus(sessionId: number) {
	return http.get<CodeGitStatus>(`/code/sessions/${sessionId}/git/status`, undefined, { timeout: 15000 })
}

export function commitCodeGitChanges(sessionId: number, repositoryId: string, message: string) {
	return http.post<CodeGitDeliveryResult>(`/code/sessions/${sessionId}/git/commit`, { repositoryId, message })
}

export function mergeCodeSessionWorktree(sessionId: number) {
	return http.post<CodeGitDeliveryResult>(`/code/sessions/${sessionId}/worktree/merge`, { confirm: true })
}

export function getCodeGitDiff(sessionId: number, repositoryId: string, path: string, kind: CodeGitDiffKind) {
	return http.get<CodeGitDiff>(
		`/code/sessions/${sessionId}/git/diff`,
		{ repositoryId, path, kind },
		{ timeout: 15000 }
	)
}

export function updateCodeGitStage(sessionId: number, repositoryId: string, paths: string[], staged: boolean) {
	return http.put<CodeGitStatus>(`/code/sessions/${sessionId}/git/stage`, { repositoryId, paths, staged })
}
