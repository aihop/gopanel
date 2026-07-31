import type { AITask } from "./code"

export interface CodeTaskSummary {
	durationMs: number
	executor?: string
	model?: string
	gitStatus?: "working" | "committed" | "merged" | "pushed" | "push_failed" | "conflict"
	gitError?: string
	branch?: string
	additions: number
	deletions: number
	changedFiles: number
	hasDiff: boolean
}

export interface CodeTaskListItem extends AITask {
	summary: CodeTaskSummary
}
