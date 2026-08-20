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
	CodeSessionImagePreview,
	CodeSessionState,
	CodeStructureResult,
	CodexRuntimeState,
	CodeTokenUsageResponse,
	CodeAttentionItem,
} from "../interface/code"
import type { CodeProjectBranches } from "../interface/codeBranches"
import type { CodeProjectCommitResult } from "../interface/codeProjectCommit"
import type { AIProviderAccount, AIProviderAccountInput } from "../interface/aiAccounts"
import type {
	CodeMemoryAuditEvent,
	CodeMemoryEntry,
	CodeMemoryExtractResult,
	CodeMemoryExtractionStatus,
	CodeMemoryList,
	CodeMemorySetting
} from "../interface/codeMemories"
import type { CodeResidueCleanupOutcome, CodeWorktreeResidueSummary } from "../interface/codeResidues"
import type { CodeTaskListItem } from "../interface/codeTasks"
import type { CodeDesktopSummary } from "../interface/codeDesktop"
import type { HostTerminalSession } from "../interface/hostTerminal"
import { waitForCodeSessionInitialization } from "./codeSessionInitialization"

// === Project APIs ===

export function getCodeDesktopSummary() {
	return http.get<CodeDesktopSummary>("/code/desktop-summary")
}

export function getAIProjects(params: { page: number; limit: number }) {
	return http.get<{ items: AIProject[]; total: number }>("/code/projects", params)
}

export function createAIProject(data: { name: string; description: string; sourceDirs: string[]; excludedRepositories?: string[]; primaryRepository: string; deliveryBranch: string; deliveryMode: string; gitCredentialId: number; requireQualityGate: boolean; qualityChecks: CodeProjectQualityCheck[]; monthlyTokenBudget: number }) {
	return http.post<AIProject>("/code/projects", data)
}

export function updateAIProject(id: number, data: { name: string; description: string; sourceDirs: string[]; excludedRepositories?: string[]; primaryRepository: string; deliveryBranch: string; deliveryMode: string; gitCredentialId: number; requireQualityGate: boolean; qualityChecks: CodeProjectQualityCheck[]; monthlyTokenBudget: number }) {
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

// 实连一次远端，超时给足：这是网络往返，不是本地校验。
export function verifyCodeGitCredential(data: { credentialId?: number; username: string; secret: string; remote: string }) {
	return http.post("/code/git/credentials/verify", data, 30000)
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

// 保存时后端会实连探测，比普通请求慢，给足超时。
export function getAIProviderAccounts() {
	return http.get<AIProviderAccount[]>("/code/ai-accounts")
}

export function saveAIProviderAccount(data: AIProviderAccountInput, id?: number) {
	return id
		? http.put<AIProviderAccount>(`/code/ai-accounts/${id}`, data)
		: http.post<AIProviderAccount>("/code/ai-accounts", data, 60000)
}

export function deleteAIProviderAccount(id: number) {
	return http.delete(`/code/ai-accounts/${id}`)
}

// 直接改用户源仓库，需要显式确认。
export function commitCodeProjectChanges(projectId: number, message: string) {
	return http.post<CodeProjectCommitResult[]>(`/code/projects/${projectId}/git/commit`, { message, confirm: true }, 60000)
}

export function getCodeMemorySetting() {
	return http.get<CodeMemorySetting>("/code/memory/setting")
}

export function saveCodeMemorySetting(data: { enabled: boolean; accountId: number; growthThreshold: number }) {
	return http.put<CodeMemorySetting>("/code/memory/setting", data)
}

export function getCodeMemories(projectId?: number) {
	return http.get<CodeMemoryList>("/code/memories", projectId ? { projectId } : undefined)
}

export function createCodeMemory(data: { content: string; projectId: number; allProjects: boolean }) {
	return http.post<CodeMemoryEntry>("/code/memories", data)
}

export function deleteCodeMemory(id: number) {
	return http.delete(`/code/memories/${id}`)
}

export function saveCodeMemorySummary(content: string) {
	return http.put<{ content: string }>("/code/memory/summary", { content })
}

export function deleteCodeMemorySummary() {
	return http.delete("/code/memory/summary")
}

export function getCodeMemoryAuditEvents() {
	return http.get<CodeMemoryAuditEvent[]>("/code/memory/audit-events")
}

export function getCodeSessionMemoryStatus(sessionId: number) {
	return http.get<CodeMemoryExtractionStatus>(`/code/sessions/${sessionId}/memory/status`)
}

export function extractCodeSessionMemory(sessionId: number) {
	return http.post<CodeMemoryExtractResult>(`/code/sessions/${sessionId}/memory/extract`)
}

// 扫描要逐个 worktree 跑 git status 和 merge-base，仓库多时会慢，给足超时。
export function getCodeWorktreeResidues() {
	return http.get<CodeWorktreeResidueSummary>("/code/worktree-residues", undefined, { timeout: 30000 })
}

export function cleanupCodeWorktreeResidues(sessionIds: number[]) {
	return http.post<CodeResidueCleanupOutcome[]>("/code/worktree-residues/cleanup", { sessionIds }, 60000)
}

const fetchCodeExecutors = () => http.get<CodeExecutor[]>("/code/executors")
type CodeExecutorsResponse = Awaited<ReturnType<typeof fetchCodeExecutors>>
let codeExecutorsCache: { response: CodeExecutorsResponse; expiresAt: number } | null = null
let codeExecutorsPending: Promise<CodeExecutorsResponse> | null = null

export function getCodeExecutors(force = false) {
	if (!force && codeExecutorsCache && Date.now() < codeExecutorsCache.expiresAt) {
		return Promise.resolve(codeExecutorsCache.response)
	}
	if (!force && codeExecutorsPending) return codeExecutorsPending
	const request = fetchCodeExecutors().then(response => {
		codeExecutorsCache = { response, expiresAt: Date.now() + 5 * 60 * 1000 }
		return response
	})
	codeExecutorsPending = request.then(
		response => {
			codeExecutorsPending = null
			return response
		},
		error => {
			codeExecutorsPending = null
			throw error
		}
	)
	return codeExecutorsPending
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
		providerAccountId?: number
		provider?: CodeExecutorConfig
	},
	messages: { initializationFailed: string; initializationTimedOut: string },
) {
	return http.post<CodeSession>("/code/sessions", data).then(async response => {
		if (response.data.status !== "initializing") return response
		await waitForCodeSessionInitialization(
			() => http.get<import("../interface/code").CodeSessionInitialization>(`/code/sessions/${response.data.id}/initialization`).then(result => result.data),
			{ failed: messages.initializationFailed, timedOut: messages.initializationTimedOut },
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
		taskId ? { taskId } : undefined,
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
		},
	}))
}

export function getCodeSessionStructure(sessionId: number, path = "") {
	return http.get<CodeStructureResult>(`/code/sessions/${sessionId}/structure`, { path }, { timeout: 10000 })
}

export function getCodeSessionFile(sessionId: number, path: string) {
	return http.get<CodeSessionFile>(`/code/sessions/${sessionId}/file`, { path }, { timeout: 10000 })
}

export function getCodeSessionImagePreview(sessionId: number, path: string) {
	return http.get<CodeSessionImagePreview>(`/code/sessions/${sessionId}/file-preview`, { path }, { timeout: 20000 })
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

export function getCodeAttention(limit = 8) {
	return http.get<{ items: CodeAttentionItem[]; total: number }>("/code/attention", { limit })
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

// 获取任务列表。archived 传 1 时只列归档的，不传则只列没归档的。
export function getAITasks(params: {
	page: number
	limit: number
	projectId?: number
	includeGit?: boolean
	gitScope?: "full" | "live"
	selectedTaskId?: number
	archived?: 1
	order?: "recent"
}) {
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

// 归档 / 取消归档。归档后任务不再出现在任务列表里，但数据保留，可以在归档列表里找回。
export function setAITaskArchived(taskId: number, archived: boolean) {
	return http.put(`/code/tasks/${taskId}`, { archived })
}

// 删除任务
export function deleteAITask(taskId: number) {
	return http.delete(`/code/tasks/${taskId}`)
}
