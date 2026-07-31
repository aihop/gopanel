import type { AIProjectExecutionSummary } from "./code"

export interface CodeProjectLatestRun {
	id: number
	sessionId: number
	taskId: number
	executorId: string
	model: string
	status: string
	durationMs: number
	totalTokens: number
	createdAt: string
	completedAt?: string
}

export interface CodeProjectOverview {
	projectId: number
	taskCount: number
	executionSummary: AIProjectExecutionSummary
	tokenUsage: {
		inputTokens: number
		outputTokens: number
		cachedInputTokens: number
		reasoningTokens: number
		totalTokens: number
		runs: number
	}
	budget: {
		limitTokens: number
		usedTokens: number
		remainingTokens: number
		usagePercent: number
		exceeded: boolean
		unlimited: boolean
	}
	latestRun: CodeProjectLatestRun | null
}
