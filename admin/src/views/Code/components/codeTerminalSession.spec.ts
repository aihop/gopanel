import { describe, expect, it } from "vitest"
import { isDeliveredCodeSession } from "./codeTerminalSession"

describe("isDeliveredCodeSession", () => {
	it("识别已完成统一交付的终态会话", () => {
		expect(isDeliveredCodeSession("delivered")).toBe(true)
		expect(isDeliveredCodeSession(" delivered ")).toBe(true)
	})

	it("允许活动会话继续连接终端", () => {
		expect(isDeliveredCodeSession("active")).toBe(false)
		expect(isDeliveredCodeSession(undefined)).toBe(false)
	})
})
