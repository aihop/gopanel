export type AIReasoningEffort = "" | "low" | "medium" | "high"

export interface AIProviderAccount {
	id: number
	name: string
	baseUrl: string
	model: string
	enabled: boolean
	/** 抽取会把整段会话记录发出去，必须单独授权，不随 enabled 一起打开。 */
	useForMemoryExtraction: boolean
	priority: number
	defaultReasoningEffort: AIReasoningEffort
	/** 以下三项是保存时探测出来的事实，只读。 */
	supportsTemperature: boolean
	supportsJsonSchema: boolean
	supportsReasoningEffort: boolean
	probedAt?: string
	probeError?: string
	hasApiKey: boolean
}

export interface AIProviderAccountInput {
	name: string
	baseUrl: string
	apiKey: string
	model: string
	enabled: boolean
	useForMemoryExtraction: boolean
	priority: number
	defaultReasoningEffort: AIReasoningEffort
}
