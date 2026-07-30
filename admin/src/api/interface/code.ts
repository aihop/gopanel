export interface AIGroup {
	id: number
	createdAt: string
	name: string
	description: string
	workDir: string
	sourceDirs: string[]
	creatorId: number
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
	agentName: string
	workDir: string
	sourceWorkDir?: string
	worktreeBranch?: string
	status: string
	currentStage: string
	approvalPolicy: CodeApprovalPolicy
	providerBaseUrl?: string
	providerModel?: string
}

export interface CodeWorktreeCapability {
	available: boolean
	reason: "" | "multi_source" | "source_unavailable" | "not_git" | "not_git_root"
	sourceDir?: string
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
	nativeSessionId: string
	prompt: string
	output: string
	rawOutput?: string
	status: string
	exitCode: number
	durationMs: number
	errorMessage: string
	startedAt: string
	completedAt?: string
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
}
