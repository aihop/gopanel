import { describe, expect, it } from "vitest"
import { insertTerminalSymbol } from "./mobileTerminalInput"

describe("insertTerminalSymbol", () => {
	it("inserts a slash at the cursor", () => {
		expect(insertTerminalSymbol("model", "/", 0, 0)).toEqual({ value: "/model", cursor: 1 })
	})

	it("replaces the current selection", () => {
		expect(insertTerminalSymbol("ab cd", "/", 2, 4)).toEqual({ value: "ab/d", cursor: 3 })
	})

	it("clamps stale mobile selection positions", () => {
		expect(insertTerminalSymbol("cmd", "/", 99, 120)).toEqual({ value: "cmd/", cursor: 4 })
	})
})
