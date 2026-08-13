import type { AIProviderAccount, AIProviderProtocol } from "@/api/interface/aiAccounts"

const executorProtocols: Record<string, AIProviderProtocol> = {
	codex: "openai_responses",
	claude: "anthropic_messages",
	opencode: "openai_chat_completions",
	aider: "openai_chat_completions"
}

export function providerProtocolSupportsExecutor(protocol: AIProviderProtocol, executorId: string) {
	return executorProtocols[executorId] === protocol
}

export function filterProviderAccountsForExecutor(accounts: AIProviderAccount[], executorId: string) {
	return accounts.filter(
		account =>
			account.enabled &&
			account.hasApiKey &&
			providerProtocolSupportsExecutor(account.protocol || "openai_chat_completions", executorId)
	)
}
