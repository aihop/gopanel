import { createPinia, setActivePinia } from "pinia"
import { beforeEach, describe, expect, it, vi } from "vitest"

const nodeListAPI = vi.fn()

vi.mock("@/api/modules/node", () => ({
	nodeListAPI,
	nodeRefreshAPI: vi.fn()
}))

const { default: NodeStore } = await import("../node")

describe("NodeStore.fetchList", () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		nodeListAPI.mockReset()
	})

	it("合并同时发起的节点列表请求", async () => {
		let resolveRequest: ((value: unknown) => void) | undefined
		nodeListAPI.mockReturnValue(
			new Promise(resolve => {
				resolveRequest = resolve
			})
		)
		const store = NodeStore()

		const first = store.fetchList()
		const second = store.fetchList()

		expect(nodeListAPI).toHaveBeenCalledTimes(1)
		resolveRequest?.({ data: [{ id: 1, name: "node-1" }] })
		await Promise.all([first, second])

		expect(store.list).toEqual([{ id: 1, name: "node-1" }])
		expect(store.loaded).toBe(true)
		expect(store.loading).toBe(false)
	})
})
