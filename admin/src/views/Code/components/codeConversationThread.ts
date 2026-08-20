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

export function conversationMessageText(content: string, expanded: boolean) {
	if (expanded || !isLongConversationMessage(content)) return content
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

export function conversationRunForMessage(runs: CodeExecutionRun[], runId: number | undefined) {
	if (!runId) return undefined
	return runs.find(run => run.id === runId)
}

export function isUserConversationMessage(role: string) {
	return role === "user"
}
