import { describe, expect, it } from "vitest"
import { terminalBufferText, type TerminalBufferLineReader } from "./mobileTerminalClipboard"

function createBuffer(lines: Array<{ text: string; wrapped?: boolean }>) {
	return {
		length: lines.length,
		getLine(index: number): TerminalBufferLineReader | undefined {
			const line = lines[index]
			if (!line) return undefined
			return {
				isWrapped: Boolean(line.wrapped),
				translateToString: trimRight => (trimRight ? line.text.trimEnd() : line.text),
			}
		},
	}
}

describe("terminalBufferText", () => {
	it("trims terminal padding and trailing empty rows", () => {
		const buffer = createBuffer([{ text: "$ pwd   " }, { text: "/workspace   " }, { text: "   " }, { text: "" }])
		expect(terminalBufferText(buffer)).toBe("$ pwd\n/workspace")
	})

	it("preserves empty lines inside output", () => {
		const buffer = createBuffer([{ text: "first" }, { text: "" }, { text: "third" }])
		expect(terminalBufferText(buffer)).toBe("first\n\nthird")
	})

	it("preserves empty lines before output", () => {
		const buffer = createBuffer([{ text: "" }, { text: "second" }])
		expect(terminalBufferText(buffer)).toBe("\nsecond")
	})

	it("joins wrapped terminal rows without inserting a newline", () => {
		const buffer = createBuffer([{ text: "long command" }, { text: " continued", wrapped: true }])
		expect(terminalBufferText(buffer)).toBe("long command continued")
	})

	it("returns an empty string for an empty buffer", () => {
		expect(terminalBufferText(createBuffer([]))).toBe("")
	})
})
