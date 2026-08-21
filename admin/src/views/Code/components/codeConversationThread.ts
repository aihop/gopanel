import type { AIMessage, CodeExecutionRun } from "@/api/interface/code"

const messagePreviewMaxCharacters = 800
const messagePreviewMaxLines = 8

export function isLongConversationMessage(content: string) {
	return content.length > messagePreviewMaxCharacters || content.split("\n").length > messagePreviewMaxLines
}

export function conversationMessagePreview(content: string) {
	if (!isLongConversationMessage(content)) return content
	const lines = content.split("\n").slice(0, messagePreviewMaxLines).join("\n")
	return `${lines.slice(0, messagePreviewMaxCharacters).trimEnd()}\n…`
}


/**
 * 消息在气泡里显示成什么样。
 *
 * 执行中的消息不再压成尾部预览：正在跑的那条就是用户盯着看的东西，
 * 收起来只会让人不停去点展开，而且「…」开头会让人以为前面的内容丢了。
 * 视口停在最新一行由滚动跟随保证，不需要靠截断内容来实现。
 */
export function conversationMessageText(content: string, expanded: boolean) {
	if (expanded) return content
	if (!isLongConversationMessage(content)) return content
	return conversationMessagePreview(content)
}

const closedConversationStatuses = new Set(["delivered", "delivering", "failed"])

export function conversationSessionClosed(status: string | undefined) {
	return closedConversationStatuses.has(status?.trim() || "")
}

export function conversationSessionInitializing(status: string | undefined) {
	return status?.trim() === "initializing"
}

export function conversationRunRunning(runs: CodeExecutionRun[]) {
	return runs.some(run => run.status === "running" || run.status === "queued")
}

export function visibleConversationMessages(messages: AIMessage[], hideExecutorMessages: boolean) {
	return hideExecutorMessages ? messages.filter(item => item.role === "user") : messages
}

const injectedConversationMarkers = ["[GoPanel Git 交付约束]", "[GoPanel 长期记忆]"]

export function stripInjectedConversationPrompt(content: string) {
	let cut = -1
	for (const marker of injectedConversationMarkers) {
		const index = content.indexOf(marker)
		if (index >= 0 && (cut < 0 || index < cut)) cut = index
	}
	return (cut >= 0 ? content.slice(0, cut) : content).trim()
}

export function visibleConversationThread(messages: AIMessage[]) {
	const visible: AIMessage[] = []
	for (const message of messages) {
		const role = message.role?.trim()
		if (role === "system" || role === "developer") continue
		const content = role === "user" ? stripInjectedConversationPrompt(message.content || "") : message.content || ""
		if (!content) continue
		const item = role === "user" ? { ...message, content } : message
		const last = visible[visible.length - 1]
		if (last && last.role === item.role && (last.content || "") === content) continue
		visible.push(item)
	}
	return visible
}

export function conversationRunForMessage(runs: CodeExecutionRun[], runId: number | undefined) {
	if (!runId) return undefined
	return runs.find(run => run.id === runId)
}

export function isUserConversationMessage(role: string) {
	return role?.trim().toLowerCase() === "user"
}
