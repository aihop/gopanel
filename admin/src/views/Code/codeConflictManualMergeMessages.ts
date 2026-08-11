import { codeGitReviewMessages } from "./codeGitReviewMessages"
import { codeConflictResolverMessages } from "./codeConflictResolverMessages"

export const codeConflictManualMergeMessages = {
	zh: {
		code: {
			...codeGitReviewMessages.zh.code,
			...codeConflictResolverMessages.zh.code,
			gitConflictManualMerge: "{repository}：{branch} 合并到 {target} 时发生冲突，冲突现场已保留。"
		}
	},
	en: {
		code: {
			...codeGitReviewMessages.en.code,
			...codeConflictResolverMessages.en.code,
			gitConflictManualMerge:
				"{repository}: merging {branch} into {target} caused conflicts. The conflict workspace was preserved."
		}
	}
} as const
