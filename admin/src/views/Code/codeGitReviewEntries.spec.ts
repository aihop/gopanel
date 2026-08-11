import { describe, expect, it } from "vitest"
import type { CodeGitFile, CodeGitRepository, CodeGitStatus } from "@/api/interface/codeGit"
import { codeGitReviewEntries, codeGitReviewTotals } from "./codeGitReviewEntries"

function file(overrides: Partial<CodeGitFile> = {}): CodeGitFile {
	return {
		path: "src/example.ts",
		workspacePath: "/workspace/src/example.ts",
		indexStatus: "M",
		worktreeStatus: "M",
		staged: false,
		changed: false,
		untracked: false,
		...overrides
	}
}

function repository(files: CodeGitFile[]): CodeGitRepository {
	return {
		id: "repository",
		name: "repository",
		branch: "task/example",
		files,
		stagedCount: 0,
		changedCount: 0,
		untrackedCount: 0,
		additions: 0,
		deletions: 0,
		stagedAdditions: 0,
		stagedDeletions: 0,
		truncated: false,
		isolated: true
	}
}

function status(scope: CodeGitStatus["scope"], files: CodeGitFile[]): CodeGitStatus {
	return {
		available: true,
		repositories: [repository(files)],
		files: files.length,
		staged: 0,
		changed: 0,
		untracked: 0,
		additions: 3,
		deletions: 2,
		stagedAdditions: 5,
		stagedDeletions: 4,
		scope,
		reviewReady: true
	}
}

describe("codeGitReviewEntries", () => {
	it("任务变更按 result scope 每个结果文件只生成一个条目", () => {
		const entries = codeGitReviewEntries(
			status("result", [file({ resultStatus: "M", staged: true, changed: true })])
		)

		expect(entries.map(entry => entry.kind)).toEqual(["result"])
		expect(entries[0].key).toBe("repository:result:src/example.ts")
	})

	it("提交按 workspace scope 分开生成暂存和工作区条目", () => {
		const entries = codeGitReviewEntries(status("workspace", [file({ staged: true, changed: true })]))

		expect(entries.map(entry => entry.kind)).toEqual(["staged", "working"])
	})

	it("PC 和移动端共享同一套增删统计", () => {
		expect(codeGitReviewTotals(status("workspace", []))).toEqual({ additions: 8, deletions: 6 })
	})
})
