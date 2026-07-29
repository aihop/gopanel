import { codeProjectMessages } from "@/i18n/locales/codeProject"

const codexProviderMessages = {
	zh: {
		codexProvider: "模型连接",
		codexProviderDefault: "沿用系统默认",
		codexProviderCustom: "本会话自定义",
		codexProviderDefaultHint: "使用主机现有的 Codex 模型、服务地址和认证配置。",
		codexProviderCustomHint: "仅覆盖本次会话，不会修改主机的 Codex 全局配置；API Key 将加密保存且不会回传。",
		codexBaseUrl: "Base URL",
		codexBaseUrlPlaceholder: "例如 https://api.openai.com/v1",
		codexBaseUrlRequired: "请输入 Codex Base URL",
		codexApiKey: "API Key",
		codexApiKeyPlaceholder: "请输入本会话使用的 API Key",
		codexApiKeyRequired: "请输入 Codex API Key",
		codexWireApi: "接口协议",
		codexWireApiResponses: "Responses API",
		codexWireApiChatUnsupported: "Chat Completions（当前 Codex 不支持）"
	},
	en: {
		codexProvider: "Model connection",
		codexProviderDefault: "Use system default",
		codexProviderCustom: "Customize this session",
		codexProviderDefaultHint: "Uses the host's existing Codex model, endpoint, and authentication configuration.",
		codexProviderCustomHint:
			"Overrides only this session without changing the host Codex configuration. The API key is encrypted and never returned.",
		codexBaseUrl: "Base URL",
		codexBaseUrlPlaceholder: "Example: https://api.openai.com/v1",
		codexBaseUrlRequired: "Enter the Codex Base URL",
		codexApiKey: "API key",
		codexApiKeyPlaceholder: "Enter the API key for this session",
		codexApiKeyRequired: "Enter the Codex API key",
		codexWireApi: "API protocol",
		codexWireApiResponses: "Responses API",
		codexWireApiChatUnsupported: "Chat Completions (unsupported by current Codex)"
	}
} as const

export const newCodeSessionMessages = {
	zh: { code: { ...codeProjectMessages.zh.code, ...codexProviderMessages.zh } },
	en: { code: { ...codeProjectMessages.en.code, ...codexProviderMessages.en } }
} as const
