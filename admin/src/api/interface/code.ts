export interface AIGroup {
	id: number
	createdAt: string
	name: string
	description: string
	workDir: string
	sourceDirs: string[]
	creatorId: number
	requireQualityGate: boolean
	monthlyTokenBudget: number
	memberCount?: number
	taskCount?: number
	executionSummary: AIProjectExecutionSummary
}

export interface AIProjectExecutionSummary {
	status: "idle" | "queued" | "running" | "pending_approval"
	activeTaskCount: number
	pendingApprovalCount: number
	currentSessionId: number
	currentTaskId: number
	currentTaskTitle: string
	currentStage: string
	updatedAt?: string
}

export interface AITask {
	id: number
	createdAt: string
	sessionId: number
	projectId: number
	title: string
	agentName: string
	workDir: string
	status: string
}

export interface CodeExecutor {
	id: string
	name: string
	description: string
	installed: boolean
	available: boolean
	version: string
	configured: boolean
	reason: string
	reasonCode: string
	capabilities: string[]
	approvalPolicies: CodeApprovalPolicy[]
	nativeTerminal: boolean
	customProviderConfigurable: boolean
	configSchema?: CodeExecutorConfigSchema
}

export interface CodeExecutorConfig {
	baseUrl: string
	apiKey: string
	model: string
}

export interface CodeExecutorConfigField {
	key: keyof CodeExecutorConfig
	type: "url" | "password" | "text"
	label: string
	placeholder: string
	required: boolean
}

export interface CodeExecutorConfigSchema {
	fields: CodeExecutorConfigField[]
}

export type CodeApprovalPolicy = "manual" | "safe_auto" | "full_auto"

export interface CodeSession {
	id: number
	createdAt: string
	projectId: number
	title: string
	currentTaskTitle?: string
	agentName: string
	workDir: string
	sourceWorkDir?: string
	worktreeBranch?: string
	targetBranch?: string
	baseCommit?: string
	remoteName?: string
	remoteBranch?: string
	remoteCommit?: string
	repositorySync?: "local" | "synced" | "fast_forwarded"
	isolationMode?: "single_worktree" | "multi_worktree"
	status: string
	currentStage: string
	approvalPolicy: CodeApprovalPolicy
	providerBaseUrl?: string
	providerModel?: string
}

export interface CodeWorktreeCapability {
	available: boolean
	reason: "" | "source_unavailable" | "not_git"
	sourceDir?: string
	sourceDirs?: string[]
	repositoryCount: number
}

export interface AIMessage {
	id: number
	createdAt: string
	sessionId: number
	taskId: number
	runId: number
	role: string
	content: string
}

export interface CodeExecutionRun {
	id: number
	createdAt: string
	sessionId: number
	taskId: number
	instructionId: number
	executorId: string
	model: string
	nativeSessionId: string
	prompt: string
	output: string
	rawOutput?: string
	status: string
	exitCode: number
	durationMs: number
	inputTokens: number
	outputTokens: number
	cachedInputTokens: number
	reasoningTokens: number
	totalTokens: number
	errorMessage: string
	startedAt: string
	completedAt?: string
}

export interface CodeTokenUsage {
	inputTokens: number
	outputTokens: number
	cachedInputTokens: number
	reasoningTokens: number
	totalTokens: number
	runs: number
}

export interface CodeDailyTokenUsage extends CodeTokenUsage {
	date: string
}

export interface CodeTokenUsageResponse {
	session: CodeTokenUsage
	project: CodeTokenUsage
	daily: CodeDailyTokenUsage[]
	models: Array<CodeTokenUsage & { model: string }>
	budget: {
		limitTokens: number
		usedTokens: number
		remainingTokens: number
		usagePercent: number
		exceeded: boolean
		unlimited: boolean
	}
}

export interface CodeSessionHistory {
	session: CodeSession
	messages: AIMessage[]
	runs: CodeExecutionRun[]
	total: number
	page: number
	limit: number
}

export interface CodeInstruction {
	id: number
	createdAt: string
	sessionId: number
	taskId: number
	content: string
	status: string
	autoPreview: boolean
}

export interface CodeApproval {
	id: number
	sessionId: number
	taskId: number
	title: string
	content: string
	riskLevel: string
	status: string
	createdAt: string
}

export interface CodePreview {
	id: number
	title: string
	url: string
	status: string
}

export interface CodeTimelineEvent {
	id: number
	createdAt: string
	title: string
	content: string
	stage: string
	status: string
}

export interface CodexRuntimeState {
	responseState: "idle" | "responding" | "needsInput" | "completed" | "failed"
	needsInput: boolean
	awaitingApproval: boolean
	model: string
	inputTokens: number
	outputTokens: number
	cachedInputTokens: number
	reasoningTokens: number
	totalTokens: number
	lastAssistantPreview: string
	updatedAt: string
	wasInterrupted: boolean
}

export interface CodeSessionState {
	session: CodeSession
	codexRuntime: CodexRuntimeState | null
	currentTask: AITask | null
	latestInstruction: CodeInstruction | null
	currentStage: string
	recentOutput: string
	recentMessages: AIMessage[]
	previews: CodePreview[]
	pendingApproval: CodeApproval | null
	timelineEvents: CodeTimelineEvent[]
	errorSummary: string
	changedFiles: string[]
	latestRun: CodeExecutionRun | null
	tokenUsage: CodeTokenUsageResponse
}

export type CodeQualityKind = "test" | "lint" | "typecheck" | "build"
export type CodeQualityStatus = "passed" | "failed" | "timed_out"

export interface CodeQualityCheckResult {
	checkId: string
	status: CodeQualityStatus
	exitCode: number
	durationMs: number
	output: string
	outputTruncated: boolean
	startedAt: string
	completedAt: string
	revision?: string
	current: boolean
}

export interface CodeQualityCheck {
	id: string
	kind: CodeQualityKind
	label: string
	command: string
	workDir: string
	lastResult?: CodeQualityCheckResult
}

export interface CodeAuditEvent {
	id: number
	createdAt: string
	userId: number
	projectId: number
	sessionId: number
	action: string
	status: string
	resource: string
	detail: string
	ip: string
	durationMs: number
	meta: string
}

export interface CodeInstructionResponse {
	session: CodeSession
	instruction: CodeInstruction
	task: AITask
	approval: CodeApproval | null
}

export interface CodeStructureEntry {
	name: string
	path: string
	isDir: boolean
	extension: string
}

export interface CodeStructureResult {
	path: string
	entries: CodeStructureEntry[]
	truncated: boolean
}

export interface CodeSessionFile {
	path: string
	content: string
	extension: string
	size: number
	version: string
}
