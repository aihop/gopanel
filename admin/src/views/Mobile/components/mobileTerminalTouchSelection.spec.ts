import { describe, expect, it } from "vitest"
import { terminalCellFromPoint, terminalSelectionRange } from "./mobileTerminalTouchSelection"

describe("terminalCellFromPoint", () => {
	it("maps a screen point into the visible terminal buffer", () => {
		const cell = terminalCellFromPoint(60, 70, { left: 10, top: 20, width: 100, height: 100 }, 10, 5, 12)
		expect(cell).toEqual({ column: 5, row: 14 })
	})

	it("clamps touches outside the terminal screen", () => {
		expect(terminalCellFromPoint(-10, 200, { left: 0, top: 0, width: 100, height: 100 }, 10, 5, 3)).toEqual({
			column: 0,
			row: 7,
		})
	})
})

describe("terminalSelectionRange", () => {
	it("creates a forward multi-line selection", () => {
		expect(terminalSelectionRange({ column: 8, row: 2 }, { column: 3, row: 4 }, 10)).toEqual({
			column: 8,
			row: 2,
			length: 16,
		})
	})

	it("normalizes a backwards drag", () => {
		expect(terminalSelectionRange({ column: 3, row: 4 }, { column: 8, row: 2 }, 10)).toEqual({
			column: 8,
			row: 2,
			length: 16,
		})
	})
})
