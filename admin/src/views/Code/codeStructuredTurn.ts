export const CODE_STRUCTURED_TURN_CAPABILITY = "structured_turn"

export type CodeWorkspaceMode = "conversation" | "editor" | "changes" | "terminal"

export function executorSupportsStructuredTurn(executor: { capabilities?: string[] } | null | undefined) {
	return Boolean(executor?.capabilities?.includes(CODE_STRUCTURED_TURN_CAPABILITY))
}

export function findExecutorById<T extends { id: string }>(executors: T[], executorId: string | null | undefined) {
	const id = executorId?.trim() || ""
	if (!id) return undefined
	return executors.find(executor => executor.id === id)
}

/** 有 JSON 回合的执行器默认进对话，避免一打开就把各家 TUI 塞进 xterm。 */
export function defaultCodeWorkspaceView(options: { isProjectTerminal: boolean; structuredTurn: boolean }): {
	mode: CodeWorkspaceMode
	terminalMounted: boolean
} {
	if (options.isProjectTerminal || !options.structuredTurn) {
		return { mode: "terminal", terminalMounted: true }
	}
	return { mode: "conversation", terminalMounted: false }
}

/** 首页工作台默认模式：名单没回来前也不要先挂 PTY。 */
export function defaultDashboardWorkbenchMode(options: {
	executorId: string
	structuredTurn: boolean
	executorsLoaded: boolean
}): CodeWorkspaceMode {
	if (options.executorId.trim() === "terminal") return "terminal"
	if (!options.executorsLoaded || options.structuredTurn) return "conversation"
	return "terminal"
}
