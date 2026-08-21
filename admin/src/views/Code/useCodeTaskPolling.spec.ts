import { describe, expect, it, vi } from "vitest"
import { fetchCodeTaskPages, shouldDropDuplicateTaskFetch } from "./useCodeTaskPolling"

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

describe("shouldDropDuplicateTaskFetch", () => {
	const base = {
		silent: true,
		gitMode: "full" as const,
		activeGitMode: "full" as const,
		selectedTaskId: 0,
		activeSelectedTaskId: 0,
	}

	// 这条是「刷新按钮点了没反应」的直接成因：后台轮询几乎总有一个同参数请求在飞，
	// 按参数去重就把用户的点击一并丢了，而且连补跑都不排。
	it("用户主动刷新永远不丢", () => {
		expect(shouldDropDuplicateTaskFetch({ ...base, silent: false })).toBe(false)
	})

	it("后台轮询的同参数重复请求可以丢", () => {
		expect(shouldDropDuplicateTaskFetch(base)).toBe(true)
	})

	// 参数不同意味着结果不同，丢了就拿不到想要的数据。
	it("gitMode 不同不能丢", () => {
		expect(shouldDropDuplicateTaskFetch({ ...base, gitMode: "live" })).toBe(false)
	})

	// live 模式的结果跟着选中任务走，换了任务就是另一个请求。
	it("live 模式下选中任务不同不能丢", () => {
		const live = { ...base, gitMode: "live" as const, activeGitMode: "live" as const }
		expect(shouldDropDuplicateTaskFetch({ ...live, selectedTaskId: 7, activeSelectedTaskId: 7 })).toBe(true)
		expect(shouldDropDuplicateTaskFetch({ ...live, selectedTaskId: 7, activeSelectedTaskId: 9 })).toBe(false)
	})
})
