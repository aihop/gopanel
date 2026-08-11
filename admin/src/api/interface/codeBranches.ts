export interface CodeProjectBranch {
	name: string
	ref: string
	scope: "local" | "remote"
	current: boolean
	upstream?: string
	commit: string
	subject: string
	updatedAt: string
	merged: boolean
	managed: boolean
	taskBranch: boolean
	deletable: boolean
	deleteBlockReason?: "remote" | "current" | "delivery" | "worktree" | "session" | "task"
	additions: number
	deletions: number
}

export interface CodeProjectBranchRepository {
	name: string
	path: string
	excluded: boolean
	currentBranch?: string
	detached: boolean
	dirty: boolean
	changedFiles: number
	branches: CodeProjectBranch[]
}

export interface CodeProjectBranches {
	repositories: CodeProjectBranchRepository[]
	totalBranches: number
}
