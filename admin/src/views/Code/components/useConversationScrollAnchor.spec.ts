import { describe, expect, it } from "vitest"
import { conversationListPaddingBottom, isConversationAtBottom } from "./useConversationScrollAnchor"

describe("isConversationAtBottom", () => {
	// 容差不能省：子像素和缩放会让差值停在 1px 上下，严格相等会把
	// 「正在跟看最新消息」误判成「用户翻上去了」，跟随就此断掉。
	it("差几像素仍算贴底", () => {
		expect(isConversationAtBottom(1000, 600, 400)).toBe(true)
		expect(isConversationAtBottom(1000, 590, 400)).toBe(true)
	})

	it("真正翻进历史才算离开底部", () => {
		expect(isConversationAtBottom(1000, 500, 400)).toBe(false)
		expect(isConversationAtBottom(1000, 0, 400)).toBe(false)
	})

	// 内容还没撑满一屏时（异步 Markdown 未加载就是这个状态）不能判成「离开底部」，
	// 否则跟随一开始就是关的，后面高度暴涨也不会贴底——正是这个 bug 的成因。
	it("内容不足一屏时算贴底", () => {
		expect(isConversationAtBottom(300, 0, 400)).toBe(true)
	})
})

describe("conversationListPaddingBottom", () => {
	// 输入框会随输入内容长高，写死的 pb-40（160px）一旦不够，
	// 最后一条消息就被压在输入框下面，即使真滚到底也像没到底。
	it("按输入框实测高度让位", () => {
		expect(conversationListPaddingBottom(160)).toBe("184px")
		expect(conversationListPaddingBottom(320)).toBe("344px")
	})

	// 量不到高度时（尚未挂载、或环境没有 ResizeObserver）回落到原先的固定值，
	// 保证不会比改动前更糟。
	it("量不到高度时回落到原固定值", () => {
		expect(conversationListPaddingBottom(0)).toBe("184px")
	})
})
