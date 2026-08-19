import { codeProjectMessages } from "@/i18n/locales/codeProject"

const terminalMessages = {
	zh: {
		terminalSessionInactive: "该会话已停止运行。历史对话在上方可以查看；在此输入任意内容即可恢复会话。",
		terminalResumingByInput: "正在恢复会话…",
		terminalJumpToBottom: "回到最新输出",
		terminalWorkspaceBusy: "当前工作区已有其他任务在执行；原任务不会停止，如需并行请启用 Git Worktree 隔离",
		terminalStartFailed: "启动原生会话失败，请稍后重试",
		terminalWebSocketFailed: "终端连接失败，请检查 GoPanel 服务状态",
		terminalDisconnected: "终端连接已断开",
		resumeTerminalSession: "恢复会话",
		tokenUsageDetails: "Token 使用详情",
		inputTokens: "输入 Token",
		outputTokens: "输出 Token",
		cachedInputTokens: "缓存输入",
		reasoningTokens: "推理 Token",
		cacheHitRate: "缓存命中率",
		runtimeStep: "第 {current} / {total} 步",
		runtimeChangedFiles: "{count} 个文件已更改",
	},
	en: {
		terminalSessionInactive:
			"This session has stopped. The conversation above is still available; type anything here to resume it.",
		terminalResumingByInput: "Resuming session…",
		terminalJumpToBottom: "Jump to latest output",
		terminalWorkspaceBusy:
			"Another task is running in this workspace. It will not be stopped; enable Git Worktree isolation to run in parallel.",
		terminalStartFailed: "Failed to start the native session. Try again later.",
		terminalWebSocketFailed: "Terminal connection failed. Check the GoPanel service.",
		terminalDisconnected: "Terminal connection closed.",
		resumeTerminalSession: "Resume session",
		tokenUsageDetails: "Token usage details",
		inputTokens: "Input tokens",
		outputTokens: "Output tokens",
		cachedInputTokens: "Cached input",
		reasoningTokens: "Reasoning tokens",
		cacheHitRate: "Cache hit rate",
		runtimeStep: "Step {current} / {total}",
		runtimeChangedFiles: "{count} files changed",
	},
} as const

export const codeTerminalMessages = {
	zh: { code: { ...codeProjectMessages.zh.code, ...terminalMessages.zh } },
	en: { code: { ...codeProjectMessages.en.code, ...terminalMessages.en } },
} as const
