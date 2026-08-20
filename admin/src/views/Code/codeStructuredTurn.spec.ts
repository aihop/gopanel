import { describe, expect, it } from "vitest"
import {
	CODE_STRUCTURED_TURN_CAPABILITY,
	defaultCodeWorkspaceView,
	defaultDashboardWorkbenchMode,
	executorSupportsStructuredTurn,
	findExecutorById,
} from "./codeStructuredTurn"

describe("executorSupportsStructuredTurn", () => {
	it("只认执行器返回的 structured_turn 能力，不写死名单", () => {
		expect(executorSupportsStructuredTurn({ capabilities: [CODE_STRUCTURED_TURN_CAPABILITY, "resume"] })).toBe(true)
		expect(executorSupportsStructuredTurn({ capabilities: ["code", "resume"] })).toBe(false)
		expect(executorSupportsStructuredTurn({ capabilities: [] })).toBe(false)
		expect(executorSupportsStructuredTurn(undefined)).toBe(false)
	})
})

describe("findExecutorById", () => {
	const executors = [
		{ id: "grok", capabilities: [CODE_STRUCTURED_TURN_CAPABILITY] },
		{ id: "aider", capabilities: ["resume"] },
	]

	it("按 id 查找执行器", () => {
		expect(findExecutorById(executors, "grok")?.id).toBe("grok")
		expect(findExecutorById(executors, " missing ")).toBeUndefined()
		expect(findExecutorById(executors, "")).toBeUndefined()
	})
})

describe("defaultCodeWorkspaceView", () => {
	it("结构化回合默认对话且不挂载原生终端", () => {
		expect(defaultCodeWorkspaceView({ isProjectTerminal: false, structuredTurn: true })).toEqual({
			mode: "conversation",
			terminalMounted: false,
		})
	})

	it("项目终端和没有结构化回合的执行器仍走原生终端", () => {
		expect(defaultCodeWorkspaceView({ isProjectTerminal: true, structuredTurn: true })).toEqual({
			mode: "terminal",
			terminalMounted: true,
		})
		expect(defaultCodeWorkspaceView({ isProjectTerminal: false, structuredTurn: false })).toEqual({
			mode: "terminal",
			terminalMounted: true,
		})
	})
})

describe("defaultDashboardWorkbenchMode", () => {
	it("普通终端会话仍打开 CLI", () => {
		expect(
			defaultDashboardWorkbenchMode({ executorId: "terminal", structuredTurn: false, executorsLoaded: true }),
		).toBe("terminal")
	})

	it("执行器列表未返回时先走对话，避免 Grok TUI 抢先启动", () => {
		expect(
			defaultDashboardWorkbenchMode({ executorId: "grok", structuredTurn: false, executorsLoaded: false }),
		).toBe("conversation")
	})

	it("有 structured_turn 时默认对话", () => {
		expect(
			defaultDashboardWorkbenchMode({ executorId: "codex", structuredTurn: true, executorsLoaded: true }),
		).toBe("conversation")
	})
})
