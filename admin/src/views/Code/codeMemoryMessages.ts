import { codeWorkspaceMessages } from "./codeWorkspaceMessages"

export const codeMemoryMessages = {
	zh: {
		code: {
			...codeWorkspaceMessages.zh.code,
			memoryTitle: "记忆",
			memoryEntry: "记忆",
			memoryHint: "这些内容会加进每次执行的提示词",
			memoryLoadFailed: "记忆加载失败",
			memoryRefresh: "刷新记忆",
			// 空状态必须解释机制：只写「暂无数据」会让刚上线的用户以为功能坏了。
			memoryEmptyTitle: "还没有记忆",
			memoryEmptyHint: "每次执行结束会自动沉淀，几轮对话后开始出现",
			memoryAddFirst: "手动添加一条",
			memoryAdd: "添加记忆",
			memoryPlaceholder: "例如：这个项目一律用 pnpm，不要用 npm",
			memoryAllProjects: "所有项目",
			memoryAdded: "已添加",
			memoryAddFailed: "添加失败",
			memoryRemoved: "已移除",
			memoryRemoveFailed: "移除失败",
			memoryRemoveLabel: "移除这条记忆",
		},
	},
	en: {
		code: {
			...codeWorkspaceMessages.en.code,
			memoryTitle: "Memory",
			memoryEntry: "Memory",
			memoryHint: "These are added to the prompt of every run",
			memoryLoadFailed: "Failed to load memory",
			memoryRefresh: "Refresh memory",
			memoryEmptyTitle: "No memory yet",
			memoryEmptyHint: "Entries are distilled after each run and show up after a few exchanges",
			memoryAddFirst: "Add one manually",
			memoryAdd: "Add memory",
			memoryPlaceholder: "e.g. This project always uses pnpm, never npm",
			memoryAllProjects: "All projects",
			memoryAdded: "Added",
			memoryAddFailed: "Failed to add",
			memoryRemoved: "Removed",
			memoryRemoveFailed: "Failed to remove",
			memoryRemoveLabel: "Remove this memory",
		},
	},
}
