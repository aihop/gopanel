export type CodeGitDiffKind = "working" | "staged"

export interface CodeGitFile {
	path: string
	oldPath?: string
	workspacePath: string
	indexStatus: string
	worktreeStatus: string
	staged: boolean
	changed: boolean
	untracked: boolean
}

export interface CodeGitRepository {
	id: string
	name: string
	branch: string
	files: CodeGitFile[]
	stagedCount: number
	changedCount: number
	untrackedCount: number
	additions: number
	deletions: number
	stagedAdditions: number
	stagedDeletions: number
	truncated: boolean
	isolated: boolean
	deliveryStatus?: string
}

export interface CodeGitStatus {
	available: boolean
	reason?: string
	repositories: CodeGitRepository[]
	files: number
	staged: number
	changed: number
	untracked: number
	additions: number
	deletions: number
	stagedAdditions: number
	stagedDeletions: number
}

export interface CodeGitDiff {
	repositoryId: string
	path: string
	kind: CodeGitDiffKind
	content: string
	truncated: boolean
}

export interface CodeGitDeliveryResult {
	status: "committed" | "merged" | "conflict" | "failed"
	commit?: string
	branch?: string
	repositoryId?: string
	repositoryName?: string
	conflictFiles?: string[]
	repositories?: CodeRepositoryDeliveryResult[]
}

export interface CodeRepositoryDeliveryResult {
	repositoryId: string
	repositoryName: string
	status: string
	branch: string
	commit?: string
	conflictFiles?: string[]
}
