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
