import { describe, expect, it } from "vitest"
import { isConversationSubmitKey, sanitizeConversationMarkdown } from "./codeConversationMarkdown"

describe("isConversationSubmitKey", () => {
	it("Enter 发送，Shift+Enter 换行", () => {
		expect(isConversationSubmitKey({ key: "Enter", shiftKey: false })).toBe(true)
		expect(isConversationSubmitKey({ key: "Enter", shiftKey: true })).toBe(false)
		expect(isConversationSubmitKey({ key: "Enter", shiftKey: false, isComposing: true })).toBe(false)
		expect(isConversationSubmitKey({ key: "a", shiftKey: false })).toBe(false)
	})
})

describe("sanitizeConversationMarkdown", () => {
	it("去掉脚本和不安全链接，保留标题和代码", () => {
		const html = sanitizeConversationMarkdown(
			`<h3>Fix</h3><script>alert(1)</script><a href="javascript:alert(1)">x</a><a href="https://example.com">ok</a><pre><code>go test</code></pre>`,
		)
		expect(html).toContain("<h3>Fix</h3>")
		expect(html).toContain("go test")
		expect(html).not.toContain("script")
		expect(html).not.toContain("javascript:")
		expect(html).toContain("https://example.com")
	})
})
