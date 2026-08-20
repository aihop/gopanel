import { describe, expect, it } from "vitest"
import {
	conversationMessageText,
	conversationRunForMessage,
	conversationRunRunning,
	conversationSessionClosed,
	conversationSessionInitializing,
	isLongConversationMessage,
	isUserConversationMessage,
	stripInjectedConversationPrompt,
	visibleConversationThread,
} from "./codeConversationThread"

describe("conversation message preview", () => {
	it("长消息才截断", () => {
		expect(isLongConversationMessage("short")).toBe(false)
		expect(conversationMessageText("short", false)).toBe("short")
		const long = `${"line\n".repeat(10)}end`
		expect(isLongConversationMessage(long)).toBe(true)
		expect(conversationMessageText(long, false).endsWith("…")).toBe(true)
		expect(conversationMessageText(long, true)).toBe(long)
		expect(conversationMessageText(long, false, true).split("\n").length).toBeLessThanOrEqual(5)
		expect(conversationMessageText(long, true, true)).toBe(long)
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

	it("用户消息去掉系统注入的提示词", () => {
		expect(stripInjectedConversationPrompt("修登录\n\n[GoPanel 长期记忆]\nfoo")).toBe("修登录")
		expect(
			visibleConversationThread([
				{ id: 1, role: "user", content: "看图\n\n[GoPanel Git 交付约束]\n不要 push", sessionId: 1, taskId: 1, runId: 0, createdAt: "" },
				{ id: 2, role: "developer", content: "系统提示", sessionId: 1, taskId: 1, runId: 0, createdAt: "" },
				{ id: 3, role: "agent", content: "好的", sessionId: 1, taskId: 1, runId: 1, createdAt: "" },
			]).map(item => item.content),
		).toEqual(["看图", "好的"])
	})

	it("连续相同内容的消息只显示一条", () => {
		expect(
			visibleConversationThread([
				{ id: 1, role: "user", content: "修登录", sessionId: 1, taskId: 1, runId: 0, createdAt: "" },
				{ id: 2, role: "user", content: "修登录", sessionId: 1, taskId: 1, runId: 0, createdAt: "" },
				{ id: 3, role: "agent", content: "好了", sessionId: 1, taskId: 1, runId: 1, createdAt: "" },
				{ id: 4, role: "agent", content: "好了", sessionId: 1, taskId: 1, runId: 1, createdAt: "" },
			]).map(item => item.id),
		).toEqual([1, 3])
	})

	it("用户消息靠右，执行结果靠左", () => {
		expect(isUserConversationMessage("user")).toBe(true)
		expect(isUserConversationMessage("User")).toBe(true)
		expect(isUserConversationMessage("agent")).toBe(false)
		expect(isUserConversationMessage("assistant")).toBe(false)
	})

	it("按 runId 挂上结构化执行结果", () => {
		expect(conversationRunForMessage([{ id: 8, status: "completed" } as never], 8)?.status).toBe("completed")
		expect(conversationRunForMessage([{ id: 8, status: "completed" } as never], 3)).toBeUndefined()
	})
})
