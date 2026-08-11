export type CodeGitScope = "workspace" | "result"
export type CodeGitDiffKind = "working" | "staged" | "result"

export interface CodeGitFile {
	path: string
	oldPath?: string
	workspacePath: string
	resultStatus?: string
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
	baseCommit?: string
	resultCommit?: string
	reviewState?: "live" | "saved" | "delivered"
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
	scope: CodeGitScope
	reviewReady: boolean
	reviewRevision?: string
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

export interface CodeGitHistoryCommit {
	commit: string
	shortCommit: string
	author: string
	authoredAt: string
	subject: string
	merged: boolean
}

export interface CodeGitHistoryRepository {
	id: string
	name: string
	branch: string
	targetBranch: string
	baseCommit: string
	resultCommit: string
	commits: CodeGitHistoryCommit[]
}

export interface CodeGitHistory {
	available: boolean
	repositories: CodeGitHistoryRepository[]
	commits: number
}

export interface CodeGitHistoryDiff {
	repositoryId: string
	commit: string
	content: string
	truncated: boolean
}

export interface CodeGitHistorySelection extends CodeGitHistoryDiff {
	title: string
	subtitle: string
}

export interface CodeGitDeliveryResult {
	status: "committed" | "merged" | "partial" | "conflict" | "failed"
	resultType?: "local" | "remote_verified" | "mixed" | "delivered"
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
	resultType?: "local" | "remote_verified" | "mixed" | "delivered"
	failureCode?:
		| "source_dirty"
		/** 会话里还有运行中的 AI 执行或终端，交付拿不到工作区独占权。 */
		| "workspace_busy"
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

export interface CodeDeliveryConflictRepository {
	id: string
	name: string
	branch: string
	targetBranch: string
	files: string[]
	unresolvedFiles: string[]
	resolved: number
	total: number
}

export interface CodeDeliveryConflicts {
	repositories: CodeDeliveryConflictRepository[]
}

export type CodeDeliveryConflictResolution = "content" | "main" | "task" | "delete"

export interface CodeDeliveryConflictFile {
	repositoryId: string
	path: string
	baseContent?: string
	mainContent?: string
	taskContent?: string
	resultContent?: string
	baseExists: boolean
	mainExists: boolean
	taskExists: boolean
	resultExists: boolean
	binary: boolean
	resolved: boolean
	version: string
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
	repositoryPath?: string
	status: "pending" | "pushed" | "failed" | "local"
	remote?: string
	branch?: string
	commit?: string
	errorMessage?: string
	/** 交付提交是否已快进到本地主仓。未同步不阻断推送。 */
	localSynced: boolean
	/** 本地主仓未能自动快进的原因。 */
	localSyncError?: string
	/** 可直接执行的手动同步命令。 */
	localSyncCommand?: string
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
	repositoryPath?: string
	status: string
	branch: string
	additions: number
	deletions: number
	changedFiles: number
	targetBranch: string
	remote?: string
	remoteBranch?: string
	commit?: string
	mergeReady?: boolean
	pushStatus: "pending" | "pushed" | "failed" | "local"
	pushedCommit?: string
	sourceAppliedAt?: string
	/** 交付提交是否已经快进到本地主仓。未同步不影响交付结果，也不阻断推送。 */
	localSynced: boolean
	/** 本地主仓未能自动快进的原因。 */
	localSyncError?: string
	/** 可直接执行的手动同步命令。 */
	localSyncCommand?: string
	errorMessage?: string
	conflictFiles?: string[]
}
