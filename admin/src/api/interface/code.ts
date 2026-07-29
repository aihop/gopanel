export interface AIGroup {
	id: number
	createdAt: string
	name: string
	description: string
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
