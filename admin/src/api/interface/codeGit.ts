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
	savedCommits?: number
	headCommit?: string
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

export type CodeSessionGitSyncRepositoryStatus =
	| "synced"
	| "local"
	| "behind"
	| "offline"
	| "local_ahead"
	| "integrated"
	| "dirty"
	| "diverged"
	| "remote_behind"
	| "blocked"

export interface CodeSessionGitSyncRepository {
	id: string
	name: string
	branch: string
	remote?: string
	remoteBranch?: string
	localCommit?: string
	remoteCommit?: string
	ahead: number
	behind: number
	status: CodeSessionGitSyncRepositoryStatus
	reason?: string
	canSync: boolean
	updated: boolean
}

export interface CodeSessionGitSyncStatus {
	sessionId: number
	status: CodeSessionGitSyncRepositoryStatus
	canSync: boolean
	repositories: CodeSessionGitSyncRepository[]
}

export interface CodeGitDiff {
	repositoryId: string
	path: string
	kind: CodeGitDiffKind
	content: string
	truncated: boolean
}

export interface CodeGitDeliveryResult {
	status: "committed" | "merged" | "partial" | "conflict" | "failed"
	resultType?: "local" | "remote_verified" | "mixed"
	errorMessage?: string
	commit?: string
	branch?: string
	repositoryId?: string
	repositoryName?: string
	conflictFiles?: string[]
	repositories?: CodeRepositoryDeliveryResult[]
}

export type CodeDeliveryJobStatus = "queued" | "running" | "completed" | "partial" | "conflict" | "failed"

export interface CodeDeliveryJob {
	id: number
	sessionId: number
	taskId?: number
	status: CodeDeliveryJobStatus
	stage: string
	progress: number
	attempt: number
	queuePosition: number
	targetBranch?: string
	resultCommit?: string
	resultType?: "local" | "remote_verified" | "mixed"
	failureCode?:
		| "source_dirty"
		| "conflict"
		| "quality_failed"
		| "remote_advanced"
		| "authentication_failed"
		| "repository_unavailable"
		| "network_failed"
		| "push_failed"
		| "partial"
		| "delivery_failed"
	hasPendingChanges: boolean
	hasPendingCommits: boolean
	hasUncommittedChanges: boolean
	repositories?: CodeRepositoryDeliveryResult[]
	facts?: CodeDeliveryFact[]
	errorMessage?: string
	conflictFiles: string[]
	createdAt: string
	updatedAt: string
	startedAt?: string
	completedAt?: string
}

export interface CodeDeliveryFact {
	key: "snapshot" | "merge" | "local" | "remote"
	status: "pending" | "partial" | "completed" | "skipped"
	count?: number
	total?: number
}

export interface CodeDeliveryPushRepository {
	repositoryId: string
	repositoryName: string
	status: "pending" | "pushed" | "failed"
	remote?: string
	branch?: string
	commit?: string
	snapshotReady: boolean
	mergeReady: boolean
	errorMessage?: string
	ready: boolean
}

export interface CodeDeliveryPushResult {
	available: boolean
	status: "unavailable" | "pending" | "pushed" | "failed"
	repositories: CodeDeliveryPushRepository[]
}

export interface CodeRepositoryDeliveryResult {
	repositoryId: string
	repositoryName: string
	status: string
	branch: string
	targetBranch: string
	remote?: string
	remoteBranch?: string
	commit?: string
	pushStatus: "pending" | "pushed" | "failed" | "local"
	pushedCommit?: string
	sourceAppliedAt?: string
	errorMessage?: string
	conflictFiles?: string[]
}
