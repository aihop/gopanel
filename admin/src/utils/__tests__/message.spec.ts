import type { MessageApi, MessageOptions } from "naive-ui"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MsgRequestError, setGlobalMessageApi } from "../message"

function createMessageApi() {
	const error = vi.fn((content: string, options?: MessageOptions) => ({
		content,
		options,
		type: "error",
		destroy: vi.fn()
	}))
	const api = {
		create: vi.fn(),
		info: vi.fn(),
		success: vi.fn(),
		warning: vi.fn(),
		error,
		loading: vi.fn(),
		destroyAll: vi.fn()
	} as unknown as MessageApi

	return { api, error }
}

describe("request error messages", () => {
	beforeEach(() => {
		vi.useFakeTimers()
	})

	afterEach(() => {
		vi.runOnlyPendingTimers()
		vi.useRealTimers()
	})

	it("uses the global request error when a page also reports an error", () => {
		const { api, error } = createMessageApi()
		setGlobalMessageApi(api)

		MsgRequestError("后端返回的错误")
		api.error("页面兜底错误")
		vi.runAllTimers()

		expect(error).toHaveBeenCalledTimes(1)
		expect(error).toHaveBeenCalledWith(
			"后端返回的错误",
			expect.objectContaining({ duration: 3000, showIcon: true, closable: true })
		)
	})

	it("shows the global request error when the page stays silent", () => {
		const { api, error } = createMessageApi()
		setGlobalMessageApi(api)

		MsgRequestError("请求失败")
		expect(error).not.toHaveBeenCalled()
		vi.runAllTimers()

		expect(error).toHaveBeenCalledTimes(1)
		expect(error).toHaveBeenCalledWith("请求失败", expect.objectContaining({ duration: 3000 }))
	})

	it("does not suppress ordinary page errors", () => {
		const { api, error } = createMessageApi()
		setGlobalMessageApi(api)

		api.error("本地校验失败")

		expect(error).toHaveBeenCalledTimes(1)
		expect(error).toHaveBeenCalledWith("本地校验失败", undefined)
	})
})
