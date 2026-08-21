import { describe, expect, it, vi } from "vitest"
import { fetchCodeTaskPages } from "./useCodeTaskPolling"

describe("fetchCodeTaskPages", () => {
	it("补齐超过单页上限的全部任务记录", async () => {
		const tasks = Array.from({ length: 205 }, (_, index) => ({ id: 205 - index }))
		const loadPage = vi.fn(async (page: number) => ({
			items: tasks.slice((page - 1) * 100, page * 100),
			total: tasks.length,
			gitMode: page === 1 ? ("full" as const) : ("live" as const)
		}))

		const result = await fetchCodeTaskPages(loadPage, 100, true)

		expect(loadPage).toHaveBeenCalledTimes(3)
		expect(result.pages.flatMap(page => page.items)).toEqual(tasks)
		expect(result.total).toBe(205)
	})

	it("普通项目任务列表仍只请求一页", async () => {
		const loadPage = vi.fn(async () => ({ items: [{ id: 1 }], total: 50, gitMode: "none" as const }))

		const result = await fetchCodeTaskPages(loadPage, 50, false)

		expect(loadPage).toHaveBeenCalledTimes(1)
		expect(result.pages).toHaveLength(1)
	})
})
