import type { AITask } from "./code"

export interface CodeTaskSummary {
	durationMs: number
	totalTokens: number
	tokenUsageStatus: "pending" | "recorded" | "recovered" | "partial" | "unavailable"
	tokenRecoveredRuns: number
	tokenUnavailableRuns: number
	tokenPendingRuns: number
	executor?: string
	model?: string
	gitStatus?: "working" | "committed" | "merged" | "pushed" | "push_failed" | "conflict"
	gitError?: string
	branch?: string
	additions: number
	deletions: number
	changedFiles: number
	hasDiff: boolean
	deliveryStatus?: "queued" | "running" | "completed" | "conflict" | "failed"
	deliveryStage?: string
	deliveryProgress: number
	deliveryQueuePosition: number
	deliveryAttempt: number
	deliveryError?: string
	/** 会话当前阶段，比任务 status 细一档：卡在哪一步。 */
	stage?: string
	/** 执行器最后说的那句话（后端已截断到 160 字）。 */
	lastAgentMessage?: string
	lastActivityAt?: string
}

export interface CodeTaskListItem extends AITask {
	summary: CodeTaskSummary
}
