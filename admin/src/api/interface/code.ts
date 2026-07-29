export interface AIGroup {
	id: number
	createdAt: string
	name: string
	description: string
	workDir: string
	creatorId: number
	memberCount?: number
	taskCount?: number
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
}

export interface CodeSession {
	id: number
	createdAt: string
	projectId: number
	title: string
	agentName: string
	workDir: string
	status: string
	currentStage: string
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
	title: string
	content: string
	riskLevel: string
	status: string
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

export interface CodeSessionState {
	session: CodeSession
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
