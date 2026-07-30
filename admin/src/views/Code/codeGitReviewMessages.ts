export const codeGitReviewMessages = {
	zh: {
		code: {
			gitReview: "变更评审",
			gitRefresh: "刷新变更",
			gitLoading: "正在读取 Git 变更",
			gitLoadFailed: "Git 变更加载失败",
			gitNoRepository: "当前会话目录不是 Git 仓库",
			gitNoChanges: "当前工作区没有未提交变更",
			gitSelectFile: "选择左侧文件查看差异",
			gitStaged: "已暂存",
			gitChanged: "已修改",
			gitUntracked: "新文件",
			gitStage: "暂存",
			gitUnstage: "取消暂存",
			gitStageSuccess: "Git 暂存状态已更新",
			gitStageFailed: "Git 暂存操作失败",
			gitDiffFailed: "文件差异加载失败",
			gitDiffEmpty: "该文件没有可显示的文本差异",
			gitDiffTruncated: "差异内容超过 1 MB，仅显示前一部分",
			gitFilesTruncated: "变更文件过多，仅显示前 500 项",
			gitBranchDetached: "分离 HEAD",
			gitSummary: "{files} 个文件",
			gitWorkingDiff: "工作区差异",
			gitStagedDiff: "暂存区差异",
			gitOpenFile: "打开文件"
		}
	},
	en: {
		code: {
			gitReview: "Changes",
			gitRefresh: "Refresh changes",
			gitLoading: "Loading Git changes",
			gitLoadFailed: "Failed to load Git changes",
			gitNoRepository: "The session directory is not a Git repository",
			gitNoChanges: "The working tree has no uncommitted changes",
			gitSelectFile: "Select a file to review its diff",
			gitStaged: "Staged",
			gitChanged: "Changed",
			gitUntracked: "Untracked",
			gitStage: "Stage",
			gitUnstage: "Unstage",
			gitStageSuccess: "Git staging updated",
			gitStageFailed: "Failed to update Git staging",
			gitDiffFailed: "Failed to load file diff",
			gitDiffEmpty: "No text diff is available for this file",
			gitDiffTruncated: "The diff exceeds 1 MB; only the first part is shown",
			gitFilesTruncated: "Too many changed files; showing the first 500",
			gitBranchDetached: "Detached HEAD",
			gitSummary: "{files} files",
			gitWorkingDiff: "Working tree diff",
			gitStagedDiff: "Staged diff",
			gitOpenFile: "Open file"
		}
	}
} as const
