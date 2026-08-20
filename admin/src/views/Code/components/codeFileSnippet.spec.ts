import { describe, expect, it } from "vitest"
import { parseAttachTarget } from "./codeConversationAttachments"
import { clampLineRange, formatFileLineRef, selectionLineRange } from "./codeFileSnippet"

describe("file snippet", () => {
	it("只生成路径和行号，不带代码正文", () => {
		expect(clampLineRange(3, 1, 4)).toEqual({ start: 1, end: 3 })
		expect(formatFileLineRef("src/main.go", 2, 3)).toBe("@attach src/main.go:2-3")
		expect(selectionLineRange({ startLineNumber: 12, endLineNumber: 40 })).toEqual({ start: 12, end: 40 })
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
