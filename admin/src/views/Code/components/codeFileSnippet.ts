import { fileExtension } from "./codeConversationAttachments"

export function clampLineRange(start: number, end: number, total: number) {
	const last = Math.max(1, total)
	const rawStart = Number.isFinite(start) ? start : 1
	const rawEnd = Number.isFinite(end) ? end : rawStart
	const from = Math.min(rawStart, rawEnd)
	const to = Math.max(rawStart, rawEnd)
	return {
		start: Math.min(Math.max(1, from), last),
		end: Math.min(Math.max(1, to), last),
	}
}

export function splitFileLines(content: string) {
	return content.split("\n")
}

export function extractFileLines(content: string, start: number, end: number) {
	const lines = splitFileLines(content)
	const range = clampLineRange(start, end, lines.length)
	return lines.slice(range.start - 1, range.end)
}

export function formatFileSnippet(path: string, content: string, start: number, end: number) {
	const lines = splitFileLines(content)
	const range = clampLineRange(start, end, lines.length)
	const numbered = extractFileLines(content, range.start, range.end)
		.map((line, index) => `${range.start + index}| ${line}`)
		.join("\n")
	const language = fileExtension(path) || "txt"
	return `@attach ${path}:${range.start}-${range.end}\n\n\`\`\`${language}\n${numbered}\n\`\`\``
}
