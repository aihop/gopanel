export const isDeliveredCodeSession = (status: string | undefined) => status?.trim() === "delivered"

/**
 * 终端实例池上限。每个缓存实例保留一个 xterm，只有当前可见实例维持 WebSocket；
 * 隐藏实例释放控制并断开，重新显示时按 sequence 补回输出。
 * 超过上限时 KeepAlive 按 LRU 淘汰最久没看的 xterm。
 * 工作台和开发面板共用同一个上限，免得两处各设一个数对不上。
 */
export const CODE_TERMINAL_POOL_SIZE = 8

/** xterm 的构造配置。与终端的行为逻辑放在一处，组件只负责挂载。 */
export function codeTerminalOptions() {
	return {
		cursorBlink: true,
		fontSize: 14,
		fontFamily: 'Menlo, Monaco, "Courier New", monospace',
		theme: { background: "#1e1e1e", foreground: "#d4d4d4" },
	}
}

export function terminalSizeData(cols: number, rows: number) {
	return JSON.stringify({ cols, rows })
}

export function terminalTakeControlMessage(cols: number, rows: number) {
	return JSON.stringify({ type: "take_control", data: terminalSizeData(cols, rows) })
}

export function terminalReleaseControlMessage() {
	return JSON.stringify({ type: "release_control", data: "" })
}

export function shouldAutoAcquireTerminalControl(
	hasControl: boolean,
	controlReason: unknown,
	leaseExpiresAt: unknown,
	now = Date.now(),
) {
	if (hasControl || controlReason) return false
	const expiresAt = Number(leaseExpiresAt)
	return Number.isFinite(expiresAt) && expiresAt <= now
}

export function shouldAttachOnlyToTerminal(taskId: number | null, forceStart: boolean) {
	return taskId !== null && !forceStart
}

/**
 * 服务端下发的权威尺寸。多端共看同一条终端时以服务端为准，
 * 否则两个窗口会各自 resize、把 PTY 的换行位置搅乱。
 * 拿不到合法值时返回 null——宁可保持当前尺寸，也不要 resize 成 0。
 */
export function authoritativeTerminalSize(cols: unknown, rows: unknown) {
	const authoritativeCols = Number(cols)
	const authoritativeRows = Number(rows)
	if (!Number.isInteger(authoritativeCols) || !Number.isInteger(authoritativeRows)) return null
	if (authoritativeCols <= 0 || authoritativeRows <= 0) return null
	return { cols: authoritativeCols, rows: authoritativeRows }
}

/** 只取本模块用得到的那几个字段，避免为了纯逻辑去依赖 xterm 的类型。 */
export interface TerminalViewport {
	buffer: { active: { baseY: number; viewportY: number } }
	scrollToBottom: () => void
}

/**
 * 视口是不是贴着底部。留 1 行容差：光标所在行会让差值在 0/1 之间来回跳，
 * 严格相等会把「正在跟看最新输出」误判成「用户翻到历史里去了」。
 */
export function isTerminalViewportAtBottom(baseY: number, viewportY: number) {
	return baseY - viewportY <= 1
}

/**
 * 执行一次会重排缓冲区的操作（fit / resize），并保证原本贴底的视口重排完还在底部。
 *
 * xterm 重排时按行重新折行，行数一变视口位置就跟着漂——回到会话经常落在历史中间，
 * 得手动往下滚一大截才能看到当前进度。原本就翻在历史里的不动，那是用户自己选的位置。
 */
export function keepingTerminalBottom(terminal: TerminalViewport, reflow: () => void) {
	const { baseY, viewportY } = terminal.buffer.active
	const followingBottom = isTerminalViewportAtBottom(baseY, viewportY)
	reflow()
	if (followingBottom) terminal.scrollToBottom()
}

export interface TerminalSocketParams {
	host: string
	secure: boolean
	token: string
	cols: number
	rows: number
	sessionId?: number | null
	taskId: number | null
	attachOnly: boolean
	afterSequence: number
	takeControl: boolean
}

/** 拼终端 WebSocket 地址。参数组合较多，抽出来才能逐条验。 */
export function terminalWebSocketUrl(params: TerminalSocketParams) {
	const protocol = params.secure ? "wss:" : "ws:"
	let url = `${protocol}//${params.host}/api/code/terminal?token=${params.token}&cols=${params.cols}&rows=${params.rows}`
	if (params.sessionId) {
		url += `&session_id=${params.sessionId}`
		if (params.attachOnly) url += "&attach_only=1"
		if (params.afterSequence > 0) url += `&after_sequence=${params.afterSequence}`
		if (params.takeControl) url += "&take_control=1"
	} else if (params.taskId) {
		url += `&task_id=${params.taskId}`
		if (params.attachOnly) url += "&attach_only=1"
	}
	return url
}

export type TerminalInputIntent = "resume" | "send" | "ignore"

/**
 * 终端收到输入时该做什么。
 *
 * 会话未运行时，任何输入都视为「我要在这干活」——往终端里打字不会是无意的。
 * 原先这种情况下连接已关，输入被静默丢弃，而看到终端的人第一反应必然是
 * 先敲两下；毫无反应之后才会去上方状态栏找那个「恢复会话」按钮。
 */
export function terminalInputIntent(
	terminalInactive: boolean,
	socketOpen: boolean,
	hasControl: boolean,
): TerminalInputIntent {
	if (terminalInactive) return "resume"
	if (socketOpen && hasControl) return "send"
	return "ignore"
}

export class CodeTerminalInputFallback {
	private composing = false
	private compositionEnding = false
	private terminalData: string[] = []
	private timers = new Set<ReturnType<typeof setTimeout>>()
	private compositionTimer: ReturnType<typeof setTimeout> | null = null

	recordTerminalData(data: string) {
		this.terminalData.push(data)
		this.schedule(() => {
			const index = this.terminalData.indexOf(data)
			if (index >= 0) this.terminalData.splice(index, 1)
		}, 50)
	}

	startComposition() {
		if (this.compositionTimer) {
			clearTimeout(this.compositionTimer)
			this.timers.delete(this.compositionTimer)
			this.compositionTimer = null
		}
		this.composing = true
		this.compositionEnding = false
	}

	endComposition() {
		this.composing = false
		this.compositionEnding = true
		if (this.compositionTimer) {
			clearTimeout(this.compositionTimer)
			this.timers.delete(this.compositionTimer)
		}
		this.compositionTimer = this.schedule(() => {
			this.compositionEnding = false
			this.compositionTimer = null
		}, 0)
	}

	queueInput(event: Pick<InputEvent, "data" | "inputType" | "isComposing">, send: (data: string) => void) {
		if (
			event.inputType !== "insertText" ||
			!event.data ||
			event.isComposing ||
			this.composing ||
			this.compositionEnding
		)
			return
		const data = event.data
		this.schedule(() => {
			const index = this.terminalData.indexOf(data)
			if (index >= 0) {
				this.terminalData.splice(index, 1)
				return
			}
			send(data)
		})
	}

	dispose() {
		for (const timer of this.timers) clearTimeout(timer)
		this.timers.clear()
		this.compositionTimer = null
		this.terminalData = []
	}

	private schedule(callback: () => void, delay = 0) {
		const timer = setTimeout(() => {
			this.timers.delete(timer)
			callback()
		}, delay)
		this.timers.add(timer)
		return timer
	}
}

/**
 * 终端实例在 KeepAlive 池里的身份。
 * 相同身份命中缓存 —— 滚屏不丢，重新显示时增量重连；身份变了才会新建一条终端。
 * 会话优先于任务：后端的 PTY 是按会话建的，同一会话下的多个任务本来就共用一条终端，
 * 按任务分身份会凭空多出几条连接，还会各自重放一遍历史。
 */
export function codeTerminalIdentity(sessionId: number | null, taskId: number | null) {
	if (sessionId !== null) return `session-${sessionId}`
	if (taskId !== null) return `task-${taskId}`
	return ""
}
