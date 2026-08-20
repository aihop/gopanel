import { describe, expect, it } from "vitest"
import { parseAttachTarget } from "./codeConversationAttachments"
import { clampLineRange, extractFileLines, formatFileSnippet } from "./codeFileSnippet"

describe("file snippet", () => {
	it("截取行范围并带上路径定位", () => {
		const content = "a\nb\nc\nd"
		expect(clampLineRange(3, 1, 4)).toEqual({ start: 1, end: 3 })
		expect(extractFileLines(content, 2, 3)).toEqual(["b", "c"])
		expect(formatFileSnippet("src/main.go", content, 2, 3)).toBe(
			"@attach src/main.go:2-3\n\n```go\n2| b\n3| c\n```",
		)
	})

	it("解析带行号的附件路径", () => {
		expect(parseAttachTarget("src/main.go:12-40")).toEqual({
			path: "src/main.go",
			name: "main.go:12-40",
			startLine: 12,
			endLine: 40,
		})
		expect(parseAttachTarget("src/main.go").path).toBe("src/main.go")
	})
})
