import { codeGitReviewMessages } from "./codeGitReviewMessages"

export const codeConflictManualMergeMessages = {
	zh: {
		code: {
			...codeGitReviewMessages.zh.code,
			gitConflictManualMerge: "{repository}：已保留 {branch}，请人工合并到 {target} 并解决冲突。"
		}
	},
	en: {
		code: {
			...codeGitReviewMessages.en.code,
			gitConflictManualMerge:
				"{repository}: branch {branch} was kept. Merge it into {target} manually and resolve the conflicts."
		}
	}
} as const
