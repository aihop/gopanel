import { codeGitReviewMessages } from "./codeGitReviewMessages"

export const codeGitReviewHeaderMessages = {
	zh: {
		code: {
			...codeGitReviewMessages.zh.code,
			gitTaskChanges: "提交",
			gitUnsavedChanges: "任务变更",
			gitCommitHistory: "Git记录"
		}
	},
	en: {
		code: {
			...codeGitReviewMessages.en.code,
			gitTaskChanges: "Commit",
			gitUnsavedChanges: "Task changes",
			gitCommitHistory: "Commit history"
		}
	}
} as const
