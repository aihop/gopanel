import { describe, expect, it, vi } from "vitest"
import type { CodeGitScope, CodeGitStatus } from "@/api/interface/codeGit"
import { useMobileGitReviewStatuses } from "./useMobileGitReviewStatuses"

function deferred<T>() {
	let resolve!: (value: T) => void
	const promise = new Promise<T>(done => {
		resolve = done
	})
	return { promise, resolve }
}

function status(scope: CodeGitScope, files: number): CodeGitStatus {
	return {
		available: true,
		repositories: [],
		files,
		staged: 0,
		changed: 0,
		untracked: 0,
		additions: 0,
		deletions: 0,
		stagedAdditions: 0,
		stagedDeletions: 0,
		scope,
		reviewReady: scope === "result"
	}
}

describe("useMobileGitReviewStatuses", () => {
	it("快速切换时独立加载任务结果与工作区数据", async () => {
		const resultRequest = deferred<CodeGitStatus>()
		const workspaceRequest = deferred<CodeGitStatus>()
		const loadStatus = vi.fn((_: number, scope: CodeGitScope) =>
			scope === "result" ? resultRequest.promise : workspaceRequest.promise
		)
		const state = useMobileGitReviewStatuses(loadStatus, () => "load failed")

		const resultLoading = state.load(1, "result")
		const workspaceLoading = state.load(1, "workspace")
		expect(loadStatus.mock.calls.map(call => call[1])).toEqual(["result", "workspace"])

		workspaceRequest.resolve(status("workspace", 2))
		resultRequest.resolve(status("result", 5))
		await Promise.all([resultLoading, workspaceLoading])

		expect(state.statuses.value.workspace?.files).toBe(2)
		expect(state.statuses.value.result?.files).toBe(5)
	})

	it("重置后忽略上一会话的迟到响应", async () => {
		const oldRequest = deferred<CodeGitStatus>()
		const loadStatus = vi.fn(() => oldRequest.promise)
		const state = useMobileGitReviewStatuses(loadStatus, () => "load failed")

		const pending = state.load(1, "result")
		state.reset()
		oldRequest.resolve(status("result", 9))
		await pending

		expect(state.statuses.value.result).toBeNull()
		expect(state.loading.value.result).toBe(false)
	})

	it("用户操作返回的新状态不会被旧刷新覆盖", async () => {
		const oldRequest = deferred<CodeGitStatus>()
		const state = useMobileGitReviewStatuses(
			() => oldRequest.promise,
			() => "load failed"
		)

		const pending = state.load(1, "workspace", true)
		state.replace("workspace", status("workspace", 3))
		oldRequest.resolve(status("workspace", 8))
		await pending

		expect(state.statuses.value.workspace?.files).toBe(3)
		expect(state.refreshing.value.workspace).toBe(false)
	})
})
