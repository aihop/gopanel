import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
	applyMobileTerminalCtrlModifier,
	insertTerminalSymbol,
	mobileTerminalSocketUrl,
	MobileTerminalInputFallback,
} from "./mobileTerminalInput"

describe("applyMobileTerminalCtrlModifier", () => {
	it("把 Ctrl 组合键换成控制字符", () => {
		expect(applyMobileTerminalCtrlModifier(false, "c")).toBe("c")
		expect(applyMobileTerminalCtrlModifier(true, "c")).toBe("\u0003")
		expect(applyMobileTerminalCtrlModifier(true, "?")).toBe("\x7f")
	})
})

describe("mobileTerminalSocketUrl", () => {
	it("拼出 AI 只读接管地址", () => {
		expect(
			mobileTerminalSocketUrl({
				mode: "ai",
				sessionId: 9,
				cols: 80,
				rows: 24,
				host: "panel.local",
				protocol: "https:",
				afterSequence: 4,
			}),
		).toBe("wss://panel.local/api/mobile/app/terminal?session_id=9&cols=80&rows=24&read_only=1&take_control=1&after_sequence=4")
	})
})

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

describe("MobileTerminalInputFallback", () => {
	beforeEach(() => vi.useFakeTimers())
	afterEach(() => vi.useRealTimers())

	it("forwards symbols emitted only by the mobile input event", () => {
		const fallback = new MobileTerminalInputFallback()
		const send = vi.fn()
		fallback.queueInput({ data: "@", inputType: "insertText", isComposing: false }, send)
		vi.runAllTimers()
		expect(send).toHaveBeenCalledWith("@")
		fallback.dispose()
	})

	it("does not duplicate symbols already emitted by xterm keyboard handling", () => {
		const fallback = new MobileTerminalInputFallback()
		const send = vi.fn()
		fallback.recordTerminalData("#")
		fallback.queueInput({ data: "#", inputType: "insertText", isComposing: false }, send)
		vi.runAllTimers()
		expect(send).not.toHaveBeenCalled()
		fallback.dispose()
	})

	it("waits for xterm keyCode 229 fallback before forwarding input", () => {
		const fallback = new MobileTerminalInputFallback()
		const send = vi.fn()
		setTimeout(() => fallback.recordTerminalData("?"), 0)
		fallback.queueInput({ data: "?", inputType: "insertText", isComposing: false }, send)
		vi.runAllTimers()
		expect(send).not.toHaveBeenCalled()
		fallback.dispose()
	})

	it("leaves composition input to xterm", () => {
		const fallback = new MobileTerminalInputFallback()
		const send = vi.fn()
		fallback.startComposition()
		fallback.queueInput({ data: "中", inputType: "insertText", isComposing: true }, send)
		fallback.endComposition()
		fallback.queueInput({ data: "中", inputType: "insertText", isComposing: false }, send)
		vi.runAllTimers()
		expect(send).not.toHaveBeenCalled()
		fallback.dispose()
	})
})
