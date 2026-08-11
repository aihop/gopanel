import http from "@/api"
import type {
	CodeDeliveryConflictFile,
	CodeDeliveryConflictResolution,
	CodeDeliveryConflicts,
	CodeDeliveryJob,
	CodeDeliveryPushResult,
	CodeGitDeliveryResult,
	CodeGitDiff,
	CodeGitDiffKind,
	CodeGitHistory,
	CodeGitHistoryDiff,
	CodeGitScope,
	CodeGitStatus,
	CodeSessionGitSyncStatus
} from "../interface/codeGit"

export function getCodeGitStatus(sessionId: number, scope: CodeGitScope = "workspace") {
	return http.get<CodeGitStatus>(`/code/sessions/${sessionId}/git/status`, { scope }, { timeout: 15000 })
}

export function checkCodeSessionGitSync(sessionId: number) {
	return http.post<CodeSessionGitSyncStatus>(`/code/sessions/${sessionId}/git/sync/check`, {}, 70000)
}

export function syncCodeSessionGitRepository(sessionId: number, repositoryId: string) {
	return http.post<CodeSessionGitSyncStatus>(
		`/code/sessions/${sessionId}/git/sync`,
		{ repositoryId, confirm: true },
		70000
	)
}

export function commitCodeGitChanges(sessionId: number, repositoryId: string, message: string) {
	return http.post<CodeGitDeliveryResult>(`/code/sessions/${sessionId}/git/commit`, { repositoryId, message })
}

export function saveCodeGitChanges(sessionId: number, message: string) {
	return http.post<CodeGitDeliveryResult>(`/code/sessions/${sessionId}/git/save`, { message })
}

export function mergeCodeSessionWorktree(sessionId: number, reviewRevision?: string) {
	return http.post<CodeDeliveryJob>(`/code/sessions/${sessionId}/worktree/merge`, {
		confirm: true,
		reviewRevision
	})
}

export function getCodeDeliveryJob(sessionId: number) {
	return http.get<CodeDeliveryJob | null>(`/code/sessions/${sessionId}/delivery`, undefined, { timeout: 10000 })
}

export function getCodeDeliveryConflicts(sessionId: number) {
	return http.get<CodeDeliveryConflicts>(`/code/sessions/${sessionId}/delivery/conflicts`, undefined, {
		timeout: 15000
	})
}

export function getCodeDeliveryConflictFile(sessionId: number, repositoryId: string, path: string) {
	return http.get<CodeDeliveryConflictFile>(
		`/code/sessions/${sessionId}/delivery/conflicts/file`,
		{ repositoryId, path },
		{ timeout: 15000 }
	)
}

export function saveCodeDeliveryConflictFile(
	sessionId: number,
	repositoryId: string,
	path: string,
	resolution: CodeDeliveryConflictResolution,
	content: string,
	baseVersion: string
) {
	return http.put<CodeDeliveryConflictFile>(`/code/sessions/${sessionId}/delivery/conflicts/file`, {
		repositoryId,
		path,
		resolution,
		content,
		baseVersion
	})
}

export function completeCodeDeliveryConflicts(sessionId: number) {
	return http.post<CodeDeliveryJob>(`/code/sessions/${sessionId}/delivery/conflicts/complete`, {}, 70000)
}

export function confirmManualCodeDeliveryConflict(sessionId: number) {
	return http.post<CodeDeliveryJob>(`/code/sessions/${sessionId}/delivery/conflicts/manual-confirm`, {}, 70000)
}

export function getCodeDeliveryPushStatus(sessionId: number) {
	return http.get<CodeDeliveryPushResult>(`/code/sessions/${sessionId}/delivery/push`, undefined, { timeout: 15000 })
}

export function pushCodeSessionDelivery(sessionId: number) {
	return http.post<CodeDeliveryPushResult>(`/code/sessions/${sessionId}/delivery/push`, { confirm: true }, 70000)
}

export function getCodeGitDiff(
	sessionId: number,
	repositoryId: string,
	path: string,
	kind: CodeGitDiffKind,
	scope: CodeGitScope = "workspace"
) {
	return http.get<CodeGitDiff>(
		`/code/sessions/${sessionId}/git/diff`,
		{ repositoryId, path, kind, scope },
		{ timeout: 15000 }
	)
}

export function getCodeGitHistory(sessionId: number) {
	return http.get<CodeGitHistory>(`/code/sessions/${sessionId}/git/history`, undefined, { timeout: 15000 })
}

export function getCodeGitHistoryDiff(sessionId: number, repositoryId: string, commit: string) {
	return http.get<CodeGitHistoryDiff>(
		`/code/sessions/${sessionId}/git/history/diff`,
		{ repositoryId, commit },
		{ timeout: 15000 }
	)
}

export function updateCodeGitStage(sessionId: number, repositoryId: string, paths: string[], staged: boolean) {
	return http.put<CodeGitStatus>(`/code/sessions/${sessionId}/git/stage`, { repositoryId, paths, staged })
}
