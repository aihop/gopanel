import { describe, expect, it } from "vitest"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { isMobileTaskActive, sortMobileRecentTasks } from "./mobileRecentTaskOrder"

function task(overrides: Partial<CodeTaskListItem>): CodeTaskListItem {
	return {
		id: 1,
		createdAt: "2026-08-18T10:00:00Z",
		sessionId: 1,
		projectId: 1,
		title: "task",
		agentName: "codex",
		workDir: "/workspace",
		status: "completed",
		summary: {} as CodeTaskListItem["summary"],
		...overrides
	}
}

describe("sortMobileRecentTasks", () => {
	it("places active tasks first and keeps each group recent-first", () => {
		const tasks = [
			task({ id: 4, status: "completed", updatedAt: "2026-08-18T14:00:00Z" }),
			task({ id: 2, status: "running", updatedAt: "2026-08-18T12:00:00Z" }),
			task({ id: 3, status: "queued", updatedAt: "2026-08-18T13:00:00Z" }),
			task({ id: 5, status: "pending_approval", updatedAt: "2026-08-18T15:00:00Z" }),
			task({ id: 1, status: "failed", updatedAt: "2026-08-18T11:00:00Z" })
		]
		expect(sortMobileRecentTasks(tasks).map(item => item.id)).toEqual([2, 5, 3, 4, 1])
	})

	it("recognizes active session stages and delivery queues", () => {
		expect(
			isMobileTaskActive(
				task({ status: "active", summary: { stage: "executing" } as CodeTaskListItem["summary"] })
			)
		).toBe(true)
		expect(
			isMobileTaskActive(
				task({
					status: "completed",
					summary: { deliveryStatus: "running" } as CodeTaskListItem["summary"]
				})
			)
		).toBe(true)
	})

	it("does not mutate the API response", () => {
		const tasks = [task({ id: 1 }), task({ id: 2, status: "running" })]
		sortMobileRecentTasks(tasks)
		expect(tasks.map(item => item.id)).toEqual([1, 2])
	})
})
