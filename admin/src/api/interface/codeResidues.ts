// 会话残留的分类。与后端 code_worktree_residue.go 的常量一一对应：
// 分类只用于展示预期，能不能删由服务端在清理时重新判定。
export type CodeResidueState = "safe" | "dirty" | "unmerged" | "active" | "orphan"

export interface CodeWorktreeResidue {
	sessionId: number
	sessionTitle?: string
	projectId?: number
	sessionStatus?: string
	state: CodeResidueState
	reason?: string
	directories: string[]
	branches?: string[]
	diskBytes: number
}

export interface CodeWorktreeResidueSummary {
	residues: CodeWorktreeResidue[]
	reclaimableIds: number[]
	reclaimBytes: number
}

export interface CodeResidueCleanupOutcome {
	sessionId: number
	cleaned: boolean
	reason?: string
}
