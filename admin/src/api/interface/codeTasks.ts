import type { AITask } from "./code"

export interface CodeTaskSummary {
	durationMs: number
	totalTokens: number
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
}

export interface CodeTaskListItem extends AITask {
	summary: CodeTaskSummary
}
