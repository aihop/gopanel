import { codeProjectMessages } from "@/i18n/locales/codeProject"

const providerMessages = {
	zh: {
		newAiTask: "新建 AI 任务",
		provider: "模型连接",
		providerDefault: "沿用终端默认",
		providerCustom: "本会话自定义",
		providerDefaultHint: "使用主机现有的 {executor} 模型和认证配置",
		executorNeedsProvider: "主机默认连接尚未配置，请为本会话填写模型连接。",
		executorConnectionUndetected: "未检测到登录状态，仍可沿用 Claude Code 默认连接或自定义连接",
		providerCustomHint: "仅覆盖本次会话；API Key 将加密保存、仅注入子进程且不会回传",
		providerFieldRequired: "请填写 {field}",
		providerField_baseUrl: "Base URL",
		providerField_apiKey: "API Key",
		providerField_model: "模型",
		providerPlaceholder_baseUrl: "例如 https://api.ainode.run/v1",
		providerPlaceholder_apiKey: "请输入本会话使用的 API Key",
		providerPlaceholder_model: "请输入模型 ID",
		executorFullAutoOnly: "该代码助手暂不支持交互式审批，只能使用完全自动模式",
		sessionInitializationFailed: "会话初始化失败",
		sessionInitializationTimedOut: "会话初始化等待超时，请稍后从会话列表查看"
	},
	en: {
		newAiTask: "New AI task",
		provider: "Model connection",
		providerDefault: "Use terminal default",
		providerCustom: "Customize this session",
		providerDefaultHint: "Uses the host's existing {executor} model and authentication configuration.",
		executorNeedsProvider: "The host connection is not configured. Add a model connection for this session.",
		executorConnectionUndetected:
			"Login status was not detected. You can still use Claude Code's default connection or customize one.",
		providerCustomHint:
			"Overrides only this session. The API key is encrypted, injected only into the child process, and never returned.",
		providerFieldRequired: "Enter {field}",
		providerField_baseUrl: "Base URL",
		providerField_apiKey: "API key",
		providerField_model: "Model",
		providerPlaceholder_baseUrl: "Example: https://api.ainode.run/v1",
		providerPlaceholder_apiKey: "Enter the API key for this session",
		providerPlaceholder_model: "Enter the model ID",
		executorFullAutoOnly: "This coding agent does not support interactive approval and requires fully automatic mode.",
		sessionInitializationFailed: "Session initialization failed",
		sessionInitializationTimedOut: "Session initialization is taking longer than expected. Check the session list later."
	}
} as const

export const newCodeSessionMessages = {
	zh: { code: { ...codeProjectMessages.zh.code, ...providerMessages.zh } },
	en: { code: { ...codeProjectMessages.en.code, ...providerMessages.en } }
} as const
