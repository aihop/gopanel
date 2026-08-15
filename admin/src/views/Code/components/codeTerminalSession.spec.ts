import { describe, expect, it } from "vitest"
import {
	codeTerminalIdentity,
	isDeliveredCodeSession,
	terminalSizeData,
	terminalTakeControlMessage,
} from "./codeTerminalSession"

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

describe("codeTerminalIdentity", () => {
	it("会话优先：同一会话下换任务命中同一条终端，不重连", () => {
		expect(codeTerminalIdentity(7, 11)).toBe("session-7")
		expect(codeTerminalIdentity(7, 12)).toBe("session-7")
	})

	it("没有会话时按任务分身份", () => {
		expect(codeTerminalIdentity(null, 11)).toBe("task-11")
	})

	it("换会话就是另一条终端", () => {
		expect(codeTerminalIdentity(7, 11)).not.toBe(codeTerminalIdentity(8, 11))
	})

	it("回到任务主页时没有身份，池里的实例被缓存而不是销毁", () => {
		expect(codeTerminalIdentity(null, null)).toBe("")
	})
})

describe("terminal control protocol", () => {
	it("接管时携带当前终端尺寸", () => {
		expect(JSON.parse(terminalTakeControlMessage(120, 36))).toEqual({
			type: "take_control",
			data: terminalSizeData(120, 36),
		})
		expect(JSON.parse(terminalSizeData(120, 36))).toEqual({ cols: 120, rows: 36 })
	})
})
