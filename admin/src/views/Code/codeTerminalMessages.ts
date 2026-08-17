import { codeProjectMessages } from "@/i18n/locales/codeProject"

const terminalMessages = {
	zh: {
		terminalSessionInactive: "该会话当前未运行；你可以查看完整对话，或显式恢复会话",
		terminalWorkspaceBusy: "当前工作区已有其他任务在执行；原任务不会停止，如需并行请启用 Git Worktree 隔离",
		terminalStartFailed: "启动原生会话失败，请稍后重试",
		terminalWebSocketFailed: "终端连接失败，请检查 GoPanel 服务状态",
		terminalDisconnected: "终端连接已断开",
		resumeTerminalSession: "恢复会话",
	},
	en: {
		terminalSessionInactive: "This session is not running. View the full conversation or resume it explicitly.",
		terminalWorkspaceBusy:
			"Another task is running in this workspace. It will not be stopped; enable Git Worktree isolation to run in parallel.",
		terminalStartFailed: "Failed to start the native session. Try again later.",
		terminalWebSocketFailed: "Terminal connection failed. Check the GoPanel service.",
		terminalDisconnected: "Terminal connection closed.",
		resumeTerminalSession: "Resume session",
	},
} as const

export const codeTerminalMessages = {
	zh: { code: { ...codeProjectMessages.zh.code, ...terminalMessages.zh } },
	en: { code: { ...codeProjectMessages.en.code, ...terminalMessages.en } },
} as const
