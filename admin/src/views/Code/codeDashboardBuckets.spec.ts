import { describe, expect, it } from "vitest"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import type { CodeTaskSummary } from "@/api/interface/codeTasks"
import {
	codeTaskBucket,
	filterCodeDashboardTasksByProject,
	groupCodeDashboardTasks,
	isDeliveringTask,
	matchesCodeDashboardFilter,
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

describe("filterCodeDashboardTasksByProject", () => {
	it("未选择项目时保留全部任务，选择后只保留对应项目", () => {
		const tasks = [task({ id: 1, projectId: 1 }), task({ id: 2, projectId: 2 })]
		expect(filterCodeDashboardTasksByProject(tasks, null)).toBe(tasks)
		expect(filterCodeDashboardTasksByProject(tasks, 2).map(item => item.id)).toEqual([2])
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

describe("matchesCodeDashboardFilter", () => {
	it("状态筛选和分桶口径一致", () => {
		expect(matchesCodeDashboardFilter(task({ status: "running" }), "active", now)).toBe(true)
		expect(matchesCodeDashboardFilter(task({ status: "running" }), "attention", now)).toBe(false)
		expect(matchesCodeDashboardFilter(task({ status: "pending_approval" }), "attention", now)).toBe(true)
	})

	it("交付中横跨其它桶，单独按谓词判断", () => {
		const delivering = task({ status: "completed", summary: summary({ deliveryStatus: "running" }) })
		expect(matchesCodeDashboardFilter(delivering, "delivering", now)).toBe(true)
		expect(matchesCodeDashboardFilter(delivering, "active", now)).toBe(true)
	})
})

describe("isDeliveringTask", () => {
	it("认交付队列状态，也认任务自身的 delivering", () => {
		expect(isDeliveringTask(task({ status: "delivering" }))).toBe(true)
		expect(isDeliveringTask(task({ summary: summary({ deliveryStatus: "queued" }) }))).toBe(true)
		expect(isDeliveringTask(task({ summary: summary({ deliveryStatus: "completed" }) }))).toBe(false)
	})
})
