import http from "@/api"
import type {
	AIProject,
	CodeGitCredential,
	CodeGitCredentialInput,
	AITask,
	AIMessage,
	CodeExecutor,
	CodeExecutorConfig,
	CodeExecutionRun,
	CodeInstruction,
	CodeInstructionResponse,
	CodeApproval,
	CodeApprovalPolicy,
	CodeAuditEvent,
	CodeQualityCheck,
	CodeProjectQualityCheck,
	CodeQualityPreflight,
	CodeQualityCheckResult,
	CodeWorktreeCapability,
	CodeProjectRepositoryOption,
	CodeSession,
	CodeSessionFile,
	CodeSessionHistory,
	CodeSessionState,
	CodeStructureResult,
	CodexRuntimeState,
	CodeTokenUsageResponse,
} from "../interface/code"
import type { CodeProjectBranches } from "../interface/codeBranches"
import type { CodeTaskListItem } from "../interface/codeTasks"
import type { HostTerminalSession } from "../interface/hostTerminal"
import { waitForCodeSessionInitialization } from "./codeSessionInitialization"

// === Project APIs ===

export function getAIProjects(params: { page: number; limit: number }) {
	return http.get<{ items: AIProject[]; total: number }>("/code/projects", params)
}

export function createAIProject(data: { name: string; description: string; sourceDirs: string[]; primaryRepository: string; deliveryBranch: string; deliveryMode: string; gitCredentialId: number; requireQualityGate: boolean; qualityChecks: CodeProjectQualityCheck[]; monthlyTokenBudget: number }) {
	return http.post<AIProject>("/code/projects", data)
}

export function updateAIProject(id: number, data: { name: string; description: string; sourceDirs: string[]; primaryRepository: string; deliveryBranch: string; deliveryMode: string; gitCredentialId: number; requireQualityGate: boolean; qualityChecks: CodeProjectQualityCheck[]; monthlyTokenBudget: number }) {
	return http.put<AIProject>(`/code/projects/${id}`, data)
}

export function preflightCodeProjectQualityChecks(sourceDirs: string[], qualityChecks: CodeProjectQualityCheck[]) {
	return http.post<CodeQualityPreflight>("/code/projects/quality-checks/preflight", { sourceDirs, qualityChecks })
}

export function discoverCodeProjectRepositories(sourceDirs: string[]) {
	return http.post<CodeProjectRepositoryOption[]>("/code/projects/repositories/discover", { sourceDirs })
}

export function getCodeGitCredentials() {
	return http.get<CodeGitCredential[]>("/code/git/credentials")
}

export function saveCodeGitCredential(data: CodeGitCredentialInput, id?: number) {
	return id
		? http.put<CodeGitCredential>(`/code/git/credentials/${id}`, data)
		: http.post<CodeGitCredential>("/code/git/credentials", data)
}

export function deleteCodeGitCredential(id: number) {
	return http.delete(`/code/git/credentials/${id}`)
}

export function getCodeProjectBranches(projectId: number) {
	return http.get<CodeProjectBranches>(`/code/projects/${projectId}/git/branches`, undefined, { timeout: 15000 })
}

export function deleteCodeProjectBranch(projectId: number, repositoryPath: string, branch: string, force: boolean) {
	return http.delete(`/code/projects/${projectId}/git/branches`, { repositoryPath, branch, force })
}

export function getCodeExecutors() {
	return http.get<CodeExecutor[]>("/code/executors")
}

export function getCodeWorktreeCapability(projectId: number) {
	return http.get<CodeWorktreeCapability>(`/code/projects/${projectId}/worktree-capability`)
}

export function openCodeProjectTerminal(projectId: number, sessionId?: number) {
	const query = sessionId ? `?session_id=${sessionId}` : ""
	return http.post<HostTerminalSession>(`/code/projects/${projectId}/terminal${query}`)
}

export function createCodeSession(
	data: {
		title: string
		workDir: string
		projectId: number
		executorId: string
		approvalPolicy: CodeApprovalPolicy
		isolated: boolean
		includeUncommitted: boolean
		provider?: CodeExecutorConfig
	},
	messages: { initializationFailed: string; initializationTimedOut: string }
) {
	return http.post<CodeSession>("/code/sessions", data).then(async response => {
		if (response.data.status !== "initializing") return response
		await waitForCodeSessionInitialization(
			() => http.get<import("../interface/code").CodeSessionInitialization>(`/code/sessions/${response.data.id}/initialization`).then(result => result.data),
			{ failed: messages.initializationFailed, timedOut: messages.initializationTimedOut }
		)
		const initialized = await getCodeSession(response.data.id)
		return { ...response, data: initialized.data.session }
	})
}

export function getCodeSession(sessionId: number) {
	return http.get<{ session: CodeSession }>(`/code/sessions/${sessionId}`)
}

export function updateCodeSessionApprovalPolicy(sessionId: number, approvalPolicy: CodeApprovalPolicy) {
	return http.put<CodeSession>(`/code/sessions/${sessionId}/approval-policy`, { approvalPolicy })
}

/** taskId 可选：传入后只返回该任务的对话，不传则返回整个会话的。 */
export function getCodeSessionHistory(sessionId: number, taskId?: number) {
	return http.get<CodeSessionHistory>(
		`/code/sessions/${sessionId}/history`,
		taskId ? { taskId } : undefined
	)
}

export function getCodeExecutionRun(runId: number) {
	return http.get<CodeExecutionRun>(`/code/runs/${runId}`)
}

export function getCodeSessionState(sessionId: number) {
	return http.get<CodeSessionState>(`/code/sessions/${sessionId}/state`, undefined, { timeout: 10000 }).then(response => ({
		...response,
		data: {
			...response.data,
			recentMessages: response.data.recentMessages || [],
			previews: response.data.previews || [],
			timelineEvents: response.data.timelineEvents || [],
			changedFiles: response.data.changedFiles || [],
		}
	}))
}

export function getCodeSessionStructure(sessionId: number, path = "") {
	return http.get<CodeStructureResult>(`/code/sessions/${sessionId}/structure`, { path }, { timeout: 10000 })
}

export function getCodeSessionFile(sessionId: number, path: string) {
	return http.get<CodeSessionFile>(`/code/sessions/${sessionId}/file`, { path }, { timeout: 10000 })
}

export function saveCodeSessionFile(sessionId: number, path: string, content: string, baseVersion: string) {
	return http.put<{ path: string; size: number; version: string }>(`/code/sessions/${sessionId}/file`, { path, content, baseVersion })
}

export function getCodexRuntimeState(sessionId: number) {
	return http.get<CodexRuntimeState | null>(`/code/sessions/${sessionId}/codex-runtime`, undefined, {
		timeout: 10000,
	})
}

export function getCodeTokenUsage(sessionId: number) {
	return http.get<CodeTokenUsageResponse>(`/code/sessions/${sessionId}/token-usage`, undefined, { timeout: 10000 })
}

export function getCodeAuditEvents(sessionId: number) {
	return http.get<{ items: CodeAuditEvent[]; total: number }>(`/code/sessions/${sessionId}/audit-events`, { page: 1, limit: 20 }, { timeout: 10000 })
}

export function getCodeQualityChecks(sessionId: number) {
	return http.get<{ items: CodeQualityCheck[] }>(`/code/sessions/${sessionId}/quality-checks`, undefined, {
		timeout: 15000,
	})
}

export function runCodeQualityCheck(sessionId: number, checkId: string) {
	return http.post<CodeQualityCheckResult>(`/code/sessions/${sessionId}/quality-checks/run`, { checkId }, 630000)
}

export function createCodeInstruction(sessionId: number, content: string) {
	return http.post<CodeInstructionResponse>(`/code/sessions/${sessionId}/instructions`, {
		content,
		allowCode: true,
		autoPreview: true,
	})
}

export function approveCodeInstruction(approvalId: number) {
	return http.post(`/code/approvals/${approvalId}/approve`, {})
}

export function getCodeApprovals(status = "pending") {
	return http.get<{ items: CodeApproval[]; total: number }>("/code/approvals", { status, limit: 50 })
}

export function rejectCodeInstruction(approvalId: number) {
	return http.post(`/code/approvals/${approvalId}/reject`, {})
}

export function stopCodeSession(sessionId: number) {
	return http.post(`/code/sessions/${sessionId}/stop`, {})
}

export function retryCodeInstruction(instructionId: number) {
	return http.post<CodeInstruction>(`/code/instructions/${instructionId}/retry`, {})
}

// === Task APIs ===

// 获取任务列表
export function getAITasks(params: { page: number; limit: number; projectId?: number; includeGit?: boolean }) {
	return http.get<{ items: CodeTaskListItem[]; total: number }>("/code/tasks", params)
}

// 获取某个任务的消息记录
export function getAITaskMessages(taskId: number) {
	return http.get<AIMessage[]>(`/code/tasks/${taskId}/messages`)
}

// 重命名任务
export function updateAITask(taskId: number, title: string) {
	return http.put(`/code/tasks/${taskId}`, { title })
}

// 删除任务
export function deleteAITask(taskId: number) {
	return http.delete(`/code/tasks/${taskId}`)
}
