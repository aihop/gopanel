import { codeWorkspaceMessages } from "./codeWorkspaceMessages"

export const taskRepositoryMessages = {
	zh: {
		code: {
			...codeWorkspaceMessages.zh.code,
			taskDeliveryAllRepositories: "统一交付上述所有仓库，不逐仓拆分",
			removedTaskRepository: "已从项目移除",
			taskUnsaved: "未提交",
			taskUnsavedChanges: "未提交变更",
			taskGitDetails: "查看任务 Git 详情",
			taskCumulativeOutput: "任务累计产出",
			deleteZombieBranch: "删除遗留分支",
			zombieBranchDeleted: "遗留分支 {branch} 已删除",
			zombieBranchStateFailed: "无法检查历史分支状态",
			branchDeleteBlocked_remote: "远端分支需在远端仓库中管理",
			branchDeleteBlocked_current: "当前所在分支不能删除",
			branchDeleteBlocked_delivery: "项目交付目标分支不能删除",
			branchDeleteBlocked_worktree: "分支仍被 Git Worktree 使用",
			branchDeleteBlocked_session: "分支仍被 Code 会话引用",
			branchDeleteBlocked_task: "分支仍与当前项目任务绑定"
		}
	},
	en: {
		code: {
			...codeWorkspaceMessages.en.code,
			taskDeliveryAllRepositories: "Deliver all repositories together, not separately",
			removedTaskRepository: "Removed from project",
			taskUnsaved: "Uncommitted",
			taskUnsavedChanges: "Uncommitted changes",
			taskGitDetails: "View task Git details",
			taskCumulativeOutput: "Cumulative task output",
			deleteZombieBranch: "Delete legacy branch",
			zombieBranchDeleted: "Legacy branch {branch} was deleted",
			zombieBranchStateFailed: "Unable to inspect the historical branch",
			branchDeleteBlocked_remote: "Remote branches must be managed on the remote",
			branchDeleteBlocked_current: "The current branch cannot be deleted",
			branchDeleteBlocked_delivery: "The delivery branch cannot be deleted",
			branchDeleteBlocked_worktree: "The branch is still used by a Git Worktree",
			branchDeleteBlocked_session: "The branch is still used by a Code session",
			branchDeleteBlocked_task: "The branch is still attached to an active project task"
		}
	}
} as const
