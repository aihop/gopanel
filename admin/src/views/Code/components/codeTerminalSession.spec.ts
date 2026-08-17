import { describe, expect, it } from "vitest"
import {
	codeTerminalIdentity,
	isDeliveredCodeSession,
	shouldAttachOnlyToTerminal,
	terminalInputIntent,
	terminalWebSocketUrl,
	terminalSizeData,
	terminalTakeControlMessage
} from "./codeTerminalSession"

describe("isDeliveredCodeSession", () => {
	it("识别已完成统一交付的终态会话", () => {
		expect(isDeliveredCodeSession("delivered")).toBe(true)
		expect(isDeliveredCodeSession(" delivered ")).toBe(true)
	})

	it("允许活动会话继续连接终端", () => {
		expect(isDeliveredCodeSession("active")).toBe(false)
		expect(isDeliveredCodeSession(undefined)).toBe(false)
	})
})

describe("codeTerminalIdentity", () => {
	it("会话优先：同一会话下换任务命中同一条终端，不重连", () => {
		expect(codeTerminalIdentity(7, 11)).toBe("session-7")
		expect(codeTerminalIdentity(7, 12)).toBe("session-7")
	})

	it("没有会话时按任务分身份", () => {
		expect(codeTerminalIdentity(null, 11)).toBe("task-11")
	})

	it("换会话就是另一条终端", () => {
		expect(codeTerminalIdentity(7, 11)).not.toBe(codeTerminalIdentity(8, 11))
	})

	it("回到任务主页时没有身份，池里的实例被缓存而不是销毁", () => {
		expect(codeTerminalIdentity(null, null)).toBe("")
	})
})

describe("terminal control protocol", () => {
	it("接管时携带当前终端尺寸", () => {
		expect(JSON.parse(terminalTakeControlMessage(120, 36))).toEqual({
			type: "take_control",
			data: terminalSizeData(120, 36)
		})
		expect(JSON.parse(terminalSizeData(120, 36))).toEqual({ cols: 120, rows: 36 })
	})
})

describe("terminal startup intent", () => {
	it("查看已有任务时只附加，不自动恢复执行器", () => {
		expect(shouldAttachOnlyToTerminal(12, false)).toBe(true)
	})

	it("新会话和显式恢复会启动执行器", () => {
		expect(shouldAttachOnlyToTerminal(null, false)).toBe(false)
		expect(shouldAttachOnlyToTerminal(12, true)).toBe(false)
	})
})

describe("terminalInputIntent", () => {
	it("会话未运行时，任何输入都当作恢复意图", () => {
		// 往终端里打字不会是无意的。原先这种情况下连接已关、输入被静默丢弃，
		// 用户敲两下没反应，只能自己去上方状态栏找按钮。
		expect(terminalInputIntent(true, false, false)).toBe("resume")
		// 即便连接还开着、也还有控制权，未运行状态仍优先恢复：
		// 这时发 cmd 没有进程会收到。
		expect(terminalInputIntent(true, true, true)).toBe("resume")
	})

	it("正常运行且持有控制权时才发送输入", () => {
		expect(terminalInputIntent(false, true, true)).toBe("send")
	})

	it("连接未就绪或没有控制权时丢弃输入", () => {
		// 只读旁观者的按键不该串到别人的终端里。
		expect(terminalInputIntent(false, true, false)).toBe("ignore")
		expect(terminalInputIntent(false, false, true)).toBe("ignore")
	})
})

describe("terminalWebSocketUrl", () => {
	const base = {
		host: "127.0.0.1:9999",
		secure: false,
		token: "tok",
		cols: 80,
		rows: 24,
		taskId: null as number | null,
		attachOnly: false,
		afterSequence: 0,
		takeControl: false,
	}

	it("会话模式带齐各项参数", () => {
		const url = terminalWebSocketUrl({
			...base,
			sessionId: 120,
			taskId: 7,
			attachOnly: true,
			afterSequence: 42,
			takeControl: true,
		})
		expect(url).toBe(
			"ws://127.0.0.1:9999/api/code/terminal?token=tok&cols=80&rows=24" +
				"&session_id=120&attach_only=1&after_sequence=42&take_control=1",
		)
	})

	it("没有会话时退回任务模式，且不带会话专有参数", () => {
		const url = terminalWebSocketUrl({ ...base, sessionId: null, taskId: 7, afterSequence: 42, takeControl: true })
		expect(url).toContain("&task_id=7")
		// after_sequence / take_control 是会话模式独有的，任务模式带上去后端不认。
		expect(url).not.toContain("after_sequence")
		expect(url).not.toContain("take_control")
	})

	// 桌面端不是 https，协议必须退回 ws——写死 wss 会连不上本地网关。
	it("按页面协议选择 ws 还是 wss", () => {
		expect(terminalWebSocketUrl({ ...base, sessionId: 1 })).toMatch(/^ws:\/\//)
		expect(terminalWebSocketUrl({ ...base, secure: true, sessionId: 1 })).toMatch(/^wss:\/\//)
	})

	it("默认不带 attach_only，避免误判成只读附加", () => {
		expect(terminalWebSocketUrl({ ...base, sessionId: 1 })).not.toContain("attach_only")
	})
})
