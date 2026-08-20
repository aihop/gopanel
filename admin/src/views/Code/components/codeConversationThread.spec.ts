import { describe, expect, it } from "vitest"
import {
	conversationMessageText,
	conversationRunForMessage,
	conversationRunRunning,
	conversationSessionClosed,
	conversationSessionInitializing,
	isLongConversationMessage,
} from "./codeConversationThread"

describe("conversation message preview", () => {
	it("长消息才截断", () => {
		expect(isLongConversationMessage("short")).toBe(false)
		expect(conversationMessageText("short", false)).toBe("short")
		const long = `${"line\n".repeat(10)}end`
		expect(isLongConversationMessage(long)).toBe(true)
		expect(conversationMessageText(long, false).endsWith("…")).toBe(true)
		expect(conversationMessageText(long, true)).toBe(long)
	})
})

describe("conversation session gates", () => {
	it("交付和失败后不再发指令", () => {
		expect(conversationSessionClosed("delivered")).toBe(true)
		expect(conversationSessionClosed("delivering")).toBe(true)
		expect(conversationSessionClosed("failed")).toBe(true)
		expect(conversationSessionClosed("active")).toBe(false)
	})

	it("初始化中不能发送", () => {
		expect(conversationSessionInitializing("initializing")).toBe(true)
		expect(conversationSessionInitializing("active")).toBe(false)
	})

	it("识别仍在跑的回合", () => {
		expect(conversationRunRunning([{ status: "running" } as never])).toBe(true)
		expect(conversationRunRunning([{ status: "queued" } as never])).toBe(true)
		expect(conversationRunRunning([{ status: "completed" } as never])).toBe(false)
	})

	it("按 runId 挂上结构化执行结果", () => {
		expect(conversationRunForMessage([{ id: 8, status: "completed" } as never], 8)?.status).toBe("completed")
		expect(conversationRunForMessage([{ id: 8, status: "completed" } as never], 3)).toBeUndefined()
	})
})
