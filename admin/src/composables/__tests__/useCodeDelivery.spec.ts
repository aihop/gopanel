import { describe, expect, it } from "vitest"
import { ref } from "vue"
import type { CodeDeliveryJob } from "@/api/interface/codeGit"
import { codeDeliveryPhaseIcon, codeDeliveryPhaseType, useCodeDelivery } from "../useCodeDelivery"

function job(overrides: Partial<CodeDeliveryJob>): CodeDeliveryJob {
	return {
		id: 1,
		sessionId: 1,
		status: "completed",
		stage: "completed",
		progress: 100,
		attempt: 1,
		queuePosition: 0,
		hasPendingChanges: false,
		hasPendingCommits: false,
		hasUncommittedChanges: false,
		conflictFiles: [],
		createdAt: "",
		updatedAt: "",
		...overrides
	} as CodeDeliveryJob
}

// 桌面端：待交付提交只在交付完成后才算数。
function desktop(current: CodeDeliveryJob | null) {
	return useCodeDelivery({ job: ref(current), available: true, pendingRequiresCompleted: true })
}

// 移动端：不要求 completed，并额外接受会话状态与本地改动信号。
function mobile(current: CodeDeliveryJob | null, extra: { delivering?: boolean; delivered?: boolean; local?: boolean } = {}) {
	return useCodeDelivery({
		job: ref(current),
		available: true,
		extraDelivering: ref(Boolean(extra.delivering)),
		extraDelivered: ref(Boolean(extra.delivered)),
		hasLocalChanges: ref(Boolean(extra.local))
	})
}

describe("useCodeDelivery phase (desktop)", () => {
	it("treats a missing job as ready to deliver", () => {
		expect(desktop(null).phase.value).toBe("idle")
	})

	it("reports queue, quality gate and progress separately", () => {
		expect(desktop(job({ status: "queued" })).phase.value).toBe("queued")
		expect(desktop(job({ status: "running", stage: "quality_check" })).phase.value).toBe("quality_check")
		expect(desktop(job({ status: "running", stage: "merging" })).phase.value).toBe("running")
	})

	it("asks to save before delivering again when changes are uncommitted", () => {
		const state = desktop(job({ status: "completed", hasUncommittedChanges: true, hasPendingCommits: true }))
		expect(state.phase.value).toBe("needs_save")
		expect(state.canDeliver.value).toBe(false)
	})

	it("offers a follow-up delivery when only new commits are pending", () => {
		const state = desktop(job({ status: "completed", hasPendingCommits: true }))
		expect(state.phase.value).toBe("deliverable")
		expect(state.canDeliver.value).toBe(true)
	})

	it("settles on delivered when nothing is pending", () => {
		const state = desktop(job({ status: "completed" }))
		expect(state.phase.value).toBe("delivered")
		expect(state.canDeliver.value).toBe(false)
	})

	it("offers a retry after a failed delivery", () => {
		for (const status of ["failed", "conflict", "partial"] as const) {
			expect(desktop(job({ status })).phase.value).toBe("retry")
		}
	})

	// 桌面端沿用「completed 才判定待交付」，未完成时的 pending 标记不应把按钮切成保存态。
	it("ignores pending flags while the job has not completed", () => {
		const state = desktop(job({ status: "failed", hasUncommittedChanges: true, hasPendingCommits: true }))
		expect(state.phase.value).toBe("retry")
		expect(state.canDeliver.value).toBe(true)
	})

	it("keeps running state ahead of pending changes", () => {
		expect(desktop(job({ status: "running", hasUncommittedChanges: true })).phase.value).toBe("running")
	})
})

describe("useCodeDelivery phase (mobile)", () => {
	it("uses the session signal when no job exists yet", () => {
		expect(mobile(null, { delivering: true }).phase.value).toBe("running")
		expect(mobile(null, { delivered: true }).phase.value).toBe("delivered")
	})

	it("switches to save when the workspace has unsaved changes", () => {
		expect(mobile(null, { local: true }).phase.value).toBe("needs_save")
		expect(mobile(job({ status: "completed" }), { local: true }).phase.value).toBe("needs_save")
	})

	// 移动端不要求 completed，未完成时的待交付提交也应可交付。
	it("honours pending commits without requiring completion", () => {
		expect(mobile(job({ status: "idle" as never, hasPendingCommits: true })).phase.value).toBe("deliverable")
	})

	it("keeps delivering ahead of local changes", () => {
		expect(mobile(job({ status: "running" }), { local: true }).phase.value).toBe("running")
	})
})

describe("phase presentation", () => {
	it("maps phases to a stable button type", () => {
		expect(codeDeliveryPhaseType("delivered")).toBe("success")
		expect(codeDeliveryPhaseType("running")).toBe("info")
		expect(codeDeliveryPhaseType("queued")).toBe("info")
		expect(codeDeliveryPhaseType("idle")).toBe("primary")
		expect(codeDeliveryPhaseType("needs_save")).toBe("primary")
	})

	it("maps phases to a stable icon", () => {
		expect(codeDeliveryPhaseIcon("needs_save")).toBe("mdi:content-save-outline")
		expect(codeDeliveryPhaseIcon("delivered")).toBe("mdi:cloud-check-outline")
		expect(codeDeliveryPhaseIcon("running")).toBe("mdi:cloud-sync-outline")
		expect(codeDeliveryPhaseIcon("idle")).toBe("mdi:source-merge")
	})
})

describe("useCodeDelivery pendingLocalSync", () => {
	it("按 facts.local 判断，而不是 resultType", () => {
		// 推成了远端，resultType 因此是 remote_verified —— 但本地主仓只同步了 8/9。
		// 早期版本按 resultType 判定，这种情况会被漏掉，界面照样说「已交付主仓」。
		const { pendingLocalSync } = desktop(
			job({
				resultType: "remote_verified",
				facts: [
					{ key: "merge", status: "completed", count: 9, total: 9 },
					{ key: "local", status: "partial", count: 8, total: 9 },
					{ key: "remote", status: "completed", count: 9, total: 9 },
				],
			}),
		)
		expect(pendingLocalSync.value).toBe(true)
	})

	it("本地主仓全部同步时不提示待同步", () => {
		const { pendingLocalSync } = desktop(
			job({
				resultType: "mixed",
				facts: [{ key: "local", status: "completed", count: 9, total: 9 }],
			}),
		)
		expect(pendingLocalSync.value).toBe(false)
	})

	it("一个都没同步时也要提示", () => {
		const { pendingLocalSync } = desktop(
			job({ resultType: "delivered", facts: [{ key: "local", status: "pending", count: 0, total: 1 }] }),
		)
		expect(pendingLocalSync.value).toBe(true)
	})

	it("旧任务没有 facts 时不误报", () => {
		expect(desktop(job({ resultType: "local" })).pendingLocalSync.value).toBe(false)
	})

	it("交付还在跑的时候不提示", () => {
		const { pendingLocalSync } = desktop(
			job({ status: "running", facts: [{ key: "local", status: "pending", count: 0, total: 1 }] }),
		)
		expect(pendingLocalSync.value).toBe(false)
	})
})
