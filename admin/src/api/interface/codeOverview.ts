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
	tokenUsageStatus: "pending" | "recorded" | "recovered" | "unavailable"
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
		recordedRuns: number
		recoveredRuns: number
		unavailableRuns: number
		pendingRuns: number
		complete: boolean
	}
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
	latestRun: CodeProjectLatestRun | null
}

export type CodeProjectRepositorySyncStatus =
	| "synced"
	| "local"
	| "behind"
	| "ahead"
	| "diverged"
	| "dirty"
	| "offline"
	| "blocked"

export interface CodeProjectRepositorySync {
	name: string
	path: string
	branch: string
	remote?: string
	remoteBranch?: string
	localCommit?: string
	remoteCommit?: string
	ahead: number
	behind: number
	status: CodeProjectRepositorySyncStatus
	reason?: string
}

export interface CodeProjectSyncStatus {
	projectId: number
	status: CodeProjectRepositorySyncStatus
	canSync: boolean
	updated: boolean
	repositories: CodeProjectRepositorySync[]
}
