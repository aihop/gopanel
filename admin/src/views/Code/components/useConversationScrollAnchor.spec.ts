import { describe, expect, it } from "vitest"
import {
	conversationListPaddingBottom,
	isConversationAtBottom,
	isProgrammaticScrollEcho,
} from "./useConversationScrollAnchor"

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

describe("isProgrammaticScrollEcho", () => {
	// 贴底会设 scrollTop，浏览器随后异步派发 scroll 事件。等它跑起来时异步 Markdown
	// 往往又撑高了内容，回调就拿到「新 scrollHeight + 旧 scrollTop」，
	// 算出很大的距底距离，把跟随误判成「用户翻上去了」，此后再也不贴底。
	it("位置未被移动过，判定为自己的回声", () => {
		expect(isProgrammaticScrollEcho(10608, 10608)).toBe(true)
		// 子像素误差要一起吃掉。
		expect(isProgrammaticScrollEcho(10609, 10608)).toBe(true)
	})

	it("用户真的拖动过，不算回声", () => {
		expect(isProgrammaticScrollEcho(4000, 10608)).toBe(false)
		expect(isProgrammaticScrollEcho(0, 10608)).toBe(false)
	})

	// 还没有程序化贴底过时，任何滚动都是用户的。
	it("没有贴底记录时一律按用户操作处理", () => {
		expect(isProgrammaticScrollEcho(0, null)).toBe(false)
		expect(isProgrammaticScrollEcho(10608, null)).toBe(false)
	})
})

// 实测现场：内容已长到 31757px，滚动却停在 10608px。
// 这一组把当时那串状态按顺序走一遍，确保不再复现。
describe("内容持续长高时的跟随判定", () => {
	it("贴底后内容再长高，不该被判成用户离开底部", () => {
		const clientHeight = 663
		// 贴底那一刻：内容 11271，落点 10608，确实在底部。
		expect(isConversationAtBottom(11271, 10608, clientHeight)).toBe(true)

		// scroll 事件延迟到内容长到 31757 之后才执行。若直接拿这组数判定，
		// 距底 20486px，会误判成用户翻上去了——必须先被回声判据拦下。
		expect(isConversationAtBottom(31757, 10608, clientHeight)).toBe(false)
		expect(isProgrammaticScrollEcho(10608, 10608)).toBe(true)
	})
})
