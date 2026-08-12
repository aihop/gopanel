export interface CodeMemoryEntry {
	id: number
	scope: "user" | "project"
	kind: string
	tier: string
	moduleKey: string
	content: string
	projectId: number
}

export interface CodeMemoryList {
	// 后端已按注入顺序返回，界面直接渲染即可——自己再排一次，
	// 「看到的就是 AI 看到的」这条就不成立了。
	entries: CodeMemoryEntry[]
	summary: string
	total: number
}

export interface CodeMemorySetting {
	enabled: boolean
	/** 0 表示「自动」：按 启用 + 已授权抽取 + 优先级 挑一个账号。 */
	accountId: number
	accountName?: string
	// 距上次抽取新增多少条消息才再抽一次；0 表示每次执行都抽。
	growthThreshold: number
	/** 开了开关但没有可用账号时为 false，reason 说明缺什么。 */
	ready: boolean
	readyReason?: string
}
