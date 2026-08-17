export const isDeliveredCodeSession = (status: string | undefined) => status?.trim() === "delivered"

/**
 * 终端实例池上限。每个活实例 = 一个 xterm + 一条 WebSocket，
 * 瓶颈在客户端渲染和内存，不在服务端（后端本来就是多订阅者广播）。
 * 超过上限时 KeepAlive 按 LRU 淘汰最久没看的那个，那次才真正断连。
 * 工作台和开发面板共用同一个上限，免得两处各设一个数对不上。
 */
export const CODE_TERMINAL_POOL_SIZE = 8

export function terminalSizeData(cols: number, rows: number) {
	return JSON.stringify({ cols, rows })
}

export function terminalTakeControlMessage(cols: number, rows: number) {
	return JSON.stringify({ type: "take_control", data: terminalSizeData(cols, rows) })
}

export function shouldAttachOnlyToTerminal(taskId: number | null, forceStart: boolean) {
	return taskId !== null && !forceStart
}

/**
 * 终端实例在 KeepAlive 池里的身份。
 * 相同身份命中缓存 —— 连接不断、滚屏不丢；身份变了才会新建一条终端。
 * 会话优先于任务：后端的 PTY 是按会话建的，同一会话下的多个任务本来就共用一条终端，
 * 按任务分身份会凭空多出几条连接，还会各自重放一遍历史。
 */
export function codeTerminalIdentity(sessionId: number | null, taskId: number | null) {
	if (sessionId !== null) return `session-${sessionId}`
	if (taskId !== null) return `task-${taskId}`
	return ""
}
