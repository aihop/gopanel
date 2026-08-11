import { codeWorkspaceMessages } from "./codeWorkspaceMessages"

export const projectBranchMessages = {
	zh: {
		code: {
			...codeWorkspaceMessages.zh.code,
			repositoryStatus: "仓库状态",
			manageBranches: "分支管理",
			refreshRepositories: "刷新仓库状态",
			repositoryLoadFailed: "仓库状态加载失败",
			currentBranchLabel: "当前",
			otherBranches: "其他分支",
			detachedHead: "游离状态",
			branchManagementTitle: "分支管理",
			branchManagementHint: "在这里查看完整的本地、远端和任务分支。开发任务中的工作分支仍与对应任务关联。",
			taskBranch: "任务分支",
			excludedRepository: "已移除",
			removedRepositoryCleanupHint: "该仓库已从项目中移除；这里仅保留历史任务分支，可按需清理。",
			deleteZombieBranch: "删除遗留分支",
			branchDeleteBlocked_task: "GoPanel 任务分支与任务记录绑定，不能删除"
		}
	},
	en: {
		code: {
			...codeWorkspaceMessages.en.code,
			repositoryStatus: "Repository status",
			manageBranches: "Manage branches",
			refreshRepositories: "Refresh repository status",
			repositoryLoadFailed: "Failed to load repository status",
			currentBranchLabel: "Current",
			otherBranches: "Other branches",
			detachedHead: "Detached HEAD",
			branchManagementTitle: "Branch management",
			branchManagementHint: "View all local, remote, and task branches here. Task work branches remain attached to their development tasks.",
			taskBranch: "Task branch",
			excludedRepository: "Removed",
			removedRepositoryCleanupHint: "This repository was removed from the project. Its historical task branches can be cleaned up here.",
			deleteZombieBranch: "Delete legacy branch",
			branchDeleteBlocked_task: "GoPanel task branches stay attached to their task records"
		}
	}
} as const
