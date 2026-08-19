import { describe, expect, it } from "vitest"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { CodeTaskSummary } from "@/api/interface/codeTasks"
import {
	codeTaskBucket,
	codeDashboardRecentStatus,
	excludeRecentCodeDashboardTasks,
	groupCodeDashboardTasks,
	mergeCodeDashboardTasks,
	isDeliveringTask,
	recentCodeDashboardTasks,
	sortCodeTasksStably,
} from "./codeDashboardBuckets"

const now = new Date("2026-08-11T15:00:00")

function summary(overrides: Partial<CodeTaskSummary> = {}): CodeTaskSummary {
	return {
		durationMs: 0,
		totalTokens: 0,
		tokenUsageStatus: "recorded",
		tokenRecoveredRuns: 0,
		tokenUnavailableRuns: 0,
		tokenPendingRuns: 0,
		additions: 0,
		deletions: 0,
		changedFiles: 0,
		hasDiff: false,
		unsavedAdditions: 0,
		unsavedDeletions: 0,
		unsavedFiles: 0,
		hasUnsavedChanges: false,
		deliveryProgress: 0,
		deliveryQueuePosition: 0,
		deliveryAttempt: 0,
		...overrides,
	}
}

function task(overrides: Partial<CodeTaskListItem> = {}): CodeTaskListItem {
	return {
		id: 1,
		createdAt: "2026-08-11T10:00:00",
		updatedAt: "2026-08-11T14:00:00",
		sessionId: 1,
		projectId: 1,
		title: "task",
		agentName: "codex",
		workDir: "/tmp",
		status: "running",
		summary: summary(),
		...overrides,
	}
}

describe("codeTaskBucket", () => {
	it("把待审批放进待处理，优先于运行中", () => {
		expect(codeTaskBucket(task({ status: "pending_approval" }), now)).toBe("attention")
	})

	it("交付冲突即使任务已完成也要提醒", () => {
		expect(codeTaskBucket(task({ status: "completed", summary: summary({ deliveryStatus: "conflict" }) }), now)).toBe(
			"attention",
		)
	})

	it("部分交付需要人工检查失败仓库", () => {
		expect(codeTaskBucket(task({ status: "completed", summary: summary({ deliveryStatus: "partial" }) }), now)).toBe(
			"attention",
		)
	})

	it("运行中和排队中都算活跃", () => {
		expect(codeTaskBucket(task({ status: "running" }), now)).toBe("active")
		expect(codeTaskBucket(task({ status: "queued" }), now)).toBe("active")
	})

	it("任务本身已完成但交付还在跑，仍算活跃", () => {
		expect(codeTaskBucket(task({ status: "completed", summary: summary({ deliveryStatus: "running" }) }), now)).toBe(
			"active",
		)
	})

	it("今天完成的进今日完成，昨天完成的不进任何桶", () => {
		expect(codeTaskBucket(task({ status: "completed", updatedAt: "2026-08-11T09:00:00" }), now)).toBe("doneToday")
		expect(codeTaskBucket(task({ status: "completed", updatedAt: "2026-08-10T09:00:00" }), now)).toBeNull()
	})

	it("昨天失败的任务不再打扰，今天失败的要提醒", () => {
		expect(codeTaskBucket(task({ status: "failed", updatedAt: "2026-08-10T09:00:00" }), now)).toBeNull()
		expect(codeTaskBucket(task({ status: "failed", updatedAt: "2026-08-11T09:00:00" }), now)).toBe("attention")
	})

	it("没有 updatedAt 的老数据退回 createdAt", () => {
		const legacy = task({ status: "completed", createdAt: "2026-08-11T08:00:00" })
		delete (legacy as { updatedAt?: string }).updatedAt
		expect(codeTaskBucket(legacy, now)).toBe("doneToday")
	})
})

describe("groupCodeDashboardTasks", () => {
	it("每个任务只落一个桶，交付中单独计数", () => {
		const tasks = [
			task({ id: 1, status: "pending_approval" }),
			task({ id: 2, status: "running" }),
			task({ id: 3, status: "completed", summary: summary({ deliveryStatus: "running" }) }),
			task({ id: 4, status: "completed" }),
			task({ id: 5, status: "completed", updatedAt: "2026-08-01T09:00:00" }),
		]
		const grouped = groupCodeDashboardTasks(tasks, now)
		expect(grouped.attention.map(item => item.id)).toEqual([1])
		expect(grouped.active.map(item => item.id)).toEqual([2, 3])
		expect(grouped.doneToday.map(item => item.id)).toEqual([4])
		expect(grouped.deliveringCount).toBe(1)
	})
})

describe("recentCodeDashboardTasks", () => {
	it("按创建时间倒序保留最近七条，不按状态过滤", () => {
		const tasks = Array.from({ length: 9 }, (_, index) => task({
			id: 20 - index,
			createdAt: `2026-08-${String(index + 1).padStart(2, "0")}T09:00:00`,
			status: "completed",
		}))
		tasks[0].status = "failed"
		expect(recentCodeDashboardTasks(tasks).map(item => item.id)).toEqual([12, 13, 14, 15, 16, 17, 18])
	})

	it("交付中的已完成任务显示为交付中，而不是已完成", () => {
		const delivering = task({ status: "completed", summary: summary({ deliveryStatus: "queued" }) })
		expect(codeDashboardRecentStatus(delivering)).toBe("delivering")
	})

	it("项目手风琴排除已经显示在最近任务里的项目", () => {
		const tasks = Array.from({ length: 9 }, (_, index) => task({ id: index + 1 }))
		const recent = recentCodeDashboardTasks(tasks)
		expect(excludeRecentCodeDashboardTasks(tasks, recent).map(item => item.id)).toEqual([2, 1])
	})
})

describe("mergeCodeDashboardTasks", () => {
	it("合并监控与最近任务并按 id 去重，优先保留带 Git 汇总的监控数据", () => {
		const recent = [task({ id: 3 }), task({ id: 2, summary: summary({ branch: "recent" }) })]
		const monitored = [task({ id: 2, summary: summary({ branch: "monitored" }) }), task({ id: 1 })]
		const merged = mergeCodeDashboardTasks(monitored, recent)
		expect(merged.map(item => item.id)).toEqual([3, 2, 1])
		expect(merged.find(item => item.id === 2)?.summary.branch).toBe("monitored")
	})
})

describe("sortCodeTasksStably", () => {
	it("按 id 倒序，和状态无关", () => {
		const tasks = [
			task({ id: 3, status: "completed" }),
			task({ id: 1, status: "running" }),
			task({ id: 2, status: "pending_approval" }),
		]
		expect(sortCodeTasksStably(tasks).map(item => item.id)).toEqual([3, 2, 1])
	})

	it("状态变化不会改变位置 —— 列表是用来点的，行不能在指针底下跳走", () => {
		const before = [task({ id: 2, status: "running" }), task({ id: 1, status: "queued" })]
		const after = [task({ id: 2, status: "completed" }), task({ id: 1, status: "pending_approval" })]
		expect(sortCodeTasksStably(before).map(item => item.id)).toEqual(sortCodeTasksStably(after).map(item => item.id))
	})

	it("不改动传入的数组", () => {
		const tasks = [task({ id: 1 }), task({ id: 2 })]
		sortCodeTasksStably(tasks)
		expect(tasks.map(item => item.id)).toEqual([1, 2])
	})
})

describe("isDeliveringTask", () => {
	it("认交付队列状态，也认任务自身的 delivering", () => {
		expect(isDeliveringTask(task({ status: "delivering" }))).toBe(true)
		expect(isDeliveringTask(task({ summary: summary({ deliveryStatus: "queued" }) }))).toBe(true)
		expect(isDeliveringTask(task({ summary: summary({ deliveryStatus: "completed" }) }))).toBe(false)
	})
})
