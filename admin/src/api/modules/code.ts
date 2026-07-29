import http from "@/api"
import type {
	AIGroup,
	AITask,
	AIMessage,
	CodeExecutor,
	CodeExecutionRun,
	CodeInstruction,
	CodeInstructionResponse,
	CodeApprovalPolicy,
	CodeWorktreeCapability,
	CodeSession,
	CodeSessionHistory,
	CodeSessionState,
	CodexRuntimeState
} from "../interface/code"

// === Group APIs ===

export function getAIGroups(params: { page: number; limit: number }) {
	return http.get<{ items: AIGroup[]; total: number }>("/code/groups", params)
}

export function createAIGroup(data: { name: string; description: string; sourceDirs: string[] }) {
	return http.post<AIGroup>("/code/groups", data)
}

export function updateAIGroup(id: number, data: { name: string; description: string; sourceDirs: string[] }) {
	return http.put<AIGroup>(`/code/groups/${id}`, data)
}

export function getCodeExecutors() {
	return http.get<CodeExecutor[]>("/code/executors")
}

export function getCodeWorktreeCapability(projectId: number) {
	return http.get<CodeWorktreeCapability>(`/code/groups/${projectId}/worktree-capability`)
}

export function createCodeSession(data: {
	title: string
	workDir: string
	projectId: number
	executorId: string
	approvalPolicy: CodeApprovalPolicy
	isolated: boolean
}) {
	return http.post<CodeSession>("/code/sessions", data)
}

export function getCodeSession(sessionId: number) {
	return http.get<{ session: CodeSession }>(`/code/sessions/${sessionId}`)
}

export function updateCodeSessionApprovalPolicy(sessionId: number, approvalPolicy: CodeApprovalPolicy) {
	return http.put<CodeSession>(`/code/sessions/${sessionId}/approval-policy`, { approvalPolicy })
}

export function getCodeSessionHistory(sessionId: number) {
	return http.get<CodeSessionHistory>(`/code/sessions/${sessionId}/history`)
}

export function getCodeExecutionRun(runId: number) {
	return http.get<CodeExecutionRun>(`/code/runs/${runId}`)
}

export function getCodeSessionState(sessionId: number) {
	return http.get<CodeSessionState>(`/code/sessions/${sessionId}/state`, undefined, { timeout: 10000 })
}

export function getCodexRuntimeState(sessionId: number) {
	return http.get<CodexRuntimeState | null>(`/code/sessions/${sessionId}/codex-runtime`, undefined, { timeout: 10000 })
}

export function createCodeInstruction(sessionId: number, content: string) {
	return http.post<CodeInstructionResponse>(`/code/sessions/${sessionId}/instructions`, {
		content,
		allowCode: true,
		autoPreview: true
	})
}

export function approveCodeInstruction(approvalId: number) {
	return http.post(`/code/approvals/${approvalId}/approve`, {})
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
export function getAITasks(params: { page: number; limit: number; projectId?: number }) {
	return http.get<{ items: AITask[]; total: number }>("/code/tasks", params)
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
