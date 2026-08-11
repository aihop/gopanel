import { describe, expect, it } from "vitest"
import { codeProjectColor } from "./projectColor"

describe("codeProjectColor", () => {
	it("同一项目始终得到同一种颜色", () => {
		expect(codeProjectColor(7)).toBe(codeProjectColor(7))
	})

	it("相邻项目得到不同颜色", () => {
		expect(codeProjectColor(1)).not.toBe(codeProjectColor(2))
	})

	it("无效项目 ID 仍返回安全颜色", () => {
		expect(codeProjectColor(Number.NaN)).toMatch(/^#[0-9a-f]{6}$/)
	})
})
