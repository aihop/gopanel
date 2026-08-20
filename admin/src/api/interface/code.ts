export interface AIProject {
	id: number
	createdAt: string
	updatedAt?: string
	name: string
	description: string
	workDir: string
	sourceDirs: string[]
	/** 不参与开发的仓库（绝对路径）。不进快照、不建 worktree、不参与交付和本地主仓同步。 */
	excludedRepositories?: string[]
	creatorId: number
	primaryRepository?: string
	deliveryBranch: string
	deliveryMode: "direct" | "branch"
	gitCredentialId: number
	requireQualityGate: boolean
	qualityChecks: CodeProjectQualityCheck[]
	monthlyTokenBudget: number
	memberCount?: number
	taskCount?: number
	executionSummary: AIProjectExecutionSummary
}

export type CodeQualityKind = "test" | "lint" | "typecheck" | "build"

export interface CodeProjectQualityCheck {
	name: string
	kind: CodeQualityKind
	repository: string
	workDir: string
	command: string
}

export interface CodeQualityPreflightItem {
	id: string
	kind: CodeQualityKind
	label: string
	command: string
	workDir: string
	available: boolean
	reason?: string
}

export interface CodeQualityPreflight {
	ready: boolean
	items: CodeQualityPreflightItem[]
}

export interface CodeGitCredential {
	id: number
	name: string
	username: string
	hasSecret: boolean
}

export interface CodeGitCredentialInput {
	name: string
	username: string
	secret: string
	// 可选：填了就在保存前实连一次这个仓库，连不上则拒绝保存。
	verifyRemote?: string
}

export interface AIProjectExecutionSummary {
	status: "idle" | "queued" | "running" | "delivering" | "pending_approval"
	activeTaskCount: number
	pendingApprovalCount: number
	currentSessionId: number
	currentTaskId: number
	currentTaskTitle: string
	currentStage: string
	updatedAt?: string
}

export interface CodeAttentionItem {
	id: string
	type: "approval" | "initialization_failed" | "delivery_failed" | "execution_failed"
	severity: "warning" | "error"
	title: string
	summary: string
	projectId: number
	sessionId: number
	taskId?: number
	approvalId?: number
	updatedAt: string
}

export interface AITask {
	id: number
	createdAt: string
	/** 后端一直有下发，老接口没声明；开发面板按它判断「今天」。 */
	updatedAt?: string
	sessionId: number
	projectId: number
	title: string
	agentName: string
	workDir: string
	status: string
}

export interface CodeSessionInitialization {
	id: number
	status: "initializing" | "active" | "failed"
	currentStage: string
	initializationError?: string
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
	updatedAt?: string
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
	repositorySync?: "local" | "local_only" | "synced" | "fast_forwarded"
	isolationMode?: "direct" | "single_worktree" | "multi_worktree"
	includeUncommitted?: boolean
	status: string
	initializationError?: string
	deliveredAt?: string
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
	dirtyRepositories?: string[]
	snapshotSupported: boolean
}

export interface CodeProjectRepositoryOption {
	name: string
	path: string
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
	tokenUsageStatus: "pending" | "recorded" | "recovered" | "unavailable"
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
	recordedRuns: number
	recoveredRuns: number
	unavailableRuns: number
	pendingRuns: number
	complete: boolean
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
		complete: boolean
		unavailableRuns: number
		pendingRuns: number
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
	reasoningEffort?: string
	inputTokens: number
	outputTokens: number
	cachedInputTokens: number
	reasoningTokens: number
	totalTokens: number
	lastAssistantPreview: string
	updatedAt: string
	wasInterrupted: boolean
	progress?: CodeRuntimeProgress
}

export interface CodeRuntimeProgress {
	currentStep: number
	totalSteps: number
	completedSteps: number
	stepTitle: string
	changedFiles: number
	additions: number
	deletions: number
	files: string[]
	source: "git" | "codex_plan" | "codex_plan+git"
	updatedAt: string
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
	delivery: import("./codeGit").CodeDeliveryJob | null
}

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

export interface CodeSessionImagePreview {
	path: string
	contentType: string
	content: string
	size: number
}
