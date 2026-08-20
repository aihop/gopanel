export const GOPANEL_PATH_MIME = "application/x-gopanel-path"

export type ConversationAttachmentKind = "image" | "file"

export interface ConversationAttachment {
	path: string
	name: string
	kind: ConversationAttachmentKind
	startLine?: number
	endLine?: number
}

export interface ComposerAttachment extends ConversationAttachment {
	id: string
	previewUrl?: string
}

const imageExtensions = new Set(["png", "jpg", "jpeg", "gif", "webp"])

export function fileNameFromPath(path: string) {
	const normalized = path.replaceAll("\\", "/").replace(/\/+$/, "")
	const name = normalized.split("/").pop()
	return name || path
}

export function fileExtension(path: string) {
	const name = fileNameFromPath(path)
	const index = name.lastIndexOf(".")
	return index > 0 ? name.slice(index + 1).toLowerCase() : ""
}

export function parseAttachTarget(raw: string) {
	const value = raw.trim()
	const match = value.match(/^(.*):(\d+)-(\d+)$/)
	if (!match) return { path: value, name: fileNameFromPath(value) }
	const path = match[1]
	const startLine = Number(match[2])
	const endLine = Number(match[3])
	return {
		path,
		name: `${fileNameFromPath(path)}:${startLine}-${endLine}`,
		startLine,
		endLine,
	}
}

export function conversationAttachmentKind(path: string, mime = ""): ConversationAttachmentKind {
	if (mime.startsWith("image/") && mime !== "image/svg+xml") return "image"
	return imageExtensions.has(fileExtension(path)) ? "image" : "file"
}

export function normalizeConversationPath(value: string) {
	return value.trim().replaceAll("\\", "/")
}

export function isWorkspaceRelativePath(path: string) {
	const cleaned = normalizeConversationPath(path)
	if (!cleaned || cleaned === "." || cleaned.includes("://")) return false
	if (cleaned.startsWith("/") || /^[A-Za-z]:\//.test(cleaned)) return false
	if (cleaned === ".." || cleaned.startsWith("../")) return false
	return true
}

export function toWorkspaceRelativePath(filePath: string, workDir = "") {
	const cleaned = normalizeConversationPath(filePath)
	const root = normalizeConversationPath(workDir).replace(/\/+$/, "")
	if (!cleaned) return ""
	if (!root) return cleaned
	if (cleaned === root) return cleaned
	if (cleaned.startsWith(`${root}/`)) return cleaned.slice(root.length + 1)
	return cleaned
}

export function decodeFileUri(uri: string) {
	const trimmed = uri.trim()
	if (!trimmed.toLowerCase().startsWith("file:")) return ""
	try {
		const parsed = new URL(trimmed)
		let path = decodeURIComponent(parsed.pathname)
		if (parsed.hostname && parsed.hostname !== "localhost") {
			path = `/${parsed.hostname}${path}`
		}
		if (/^\/[A-Za-z]:\//.test(path)) path = path.slice(1)
		return path
	} catch {
		return ""
	}
}

export function looksLikeFilePath(value: string) {
	const cleaned = normalizeConversationPath(value)
	if (!cleaned || cleaned.includes("\n") || cleaned.includes(" ")) return false
	if (cleaned.startsWith("/") || /^[A-Za-z]:\//.test(cleaned) || cleaned.startsWith("file:")) return true
	return cleaned.includes("/") && Boolean(fileExtension(cleaned))
}

export function attachmentIdentity(item: { path: string; startLine?: number; endLine?: number }) {
	return item.startLine && item.endLine ? `${item.path}:${item.startLine}-${item.endLine}` : item.path
}

export function parsePastedLineRefs(text: string, workDir = "") {
	const attachments: ComposerAttachment[] = []
	const rest: string[] = []
	for (const line of text.split("\n")) {
		const attach = line.match(/^@attach\s+(.+?)\s*$/)
		const raw = (attach ? attach[1] : line).trim()
		const target = parseAttachTarget(raw)
		const isLineRef = Boolean(target.startLine && target.endLine && (attach || looksLikeFilePath(target.path)))
		if (!isLineRef) {
			rest.push(line)
			continue
		}
		const item = conversationAttachmentFromPath(target.path, workDir)
		if (!item) {
			rest.push(line)
			continue
		}
		attachments.push({
			...item,
			id: `${item.path}:${target.startLine}-${target.endLine}`,
			name: `${item.name}:${target.startLine}-${target.endLine}`,
			startLine: target.startLine,
			endLine: target.endLine,
		})
	}
	return { attachments, rest: rest.join("\n").replace(/^\n+|\n+$/g, "") }
}

export function parseConversationAttachments(content: string) {
	const attachments: ConversationAttachment[] = []
	const textLines: string[] = []
	for (const line of content.split("\n")) {
		const match = line.match(/^@attach\s+(.+?)\s*$/)
		if (!match) {
			textLines.push(line)
			continue
		}
		const raw = normalizeConversationPath(match[1])
		const target = parseAttachTarget(raw)
		if (!target.path || attachments.some(item => item.path === target.path && item.startLine === target.startLine && item.endLine === target.endLine)) continue
		attachments.push({
			path: target.path,
			name: target.name,
			kind: conversationAttachmentKind(target.path),
			startLine: target.startLine,
			endLine: target.endLine,
		})
	}
	return {
		text: textLines.join("\n").replace(/^\n+|\n+$/g, ""),
		attachments,
	}
}

export function serializeInstructionContent(text: string, attachments: ConversationAttachment[]) {
	const body = text.trim()
	const lines: string[] = []
	for (const item of attachments) {
		const path = normalizeConversationPath(item.path)
		if (!path) continue
		const target = item.startLine && item.endLine ? `${path}:${item.startLine}-${item.endLine}` : path
		const line = `@attach ${target}`
		if (!lines.includes(line)) lines.push(line)
	}
	if (!lines.length) return body
	if (!body) return lines.join("\n")
	return `${body}\n\n${lines.join("\n")}`
}

export function conversationAttachmentFromPath(path: string, workDir = "", file?: File): ComposerAttachment | null {
	const resolved = toWorkspaceRelativePath(path, workDir)
	if (!resolved) return null
	const kind = conversationAttachmentKind(resolved, file?.type)
	return {
		id: resolved,
		path: resolved,
		name: fileNameFromPath(resolved),
		kind,
		previewUrl: kind === "image" && file && typeof URL.createObjectURL === "function" ? URL.createObjectURL(file) : undefined,
	}
}

export function attachmentsFromPaths(paths: string[], workDir = "") {
	const attachments: ComposerAttachment[] = []
	for (const path of paths) {
		const item = conversationAttachmentFromPath(path, workDir)
		if (!item || attachments.some(existing => existing.path === item.path)) continue
		attachments.push(item)
	}
	return attachments
}

export function writeStructureDragData(dataTransfer: DataTransfer, path: string) {
	dataTransfer.effectAllowed = "copy"
	dataTransfer.setData(GOPANEL_PATH_MIME, path)
	dataTransfer.setData("text/plain", path)
}

function revokePreviewUrl(url?: string) {
	if (url?.startsWith("blob:") && typeof URL.revokeObjectURL === "function") URL.revokeObjectURL(url)
}

function addDroppedPath(
	attachments: ComposerAttachment[],
	seen: Set<string>,
	rawPath: string,
	workDir: string,
	file?: File,
) {
	const item = conversationAttachmentFromPath(rawPath, workDir, file)
	if (!item) return
	const existing = attachments.find(attachment => attachment.path === item.path)
	if (existing) {
		if (!existing.previewUrl && item.previewUrl) existing.previewUrl = item.previewUrl
		else revokePreviewUrl(item.previewUrl)
		return
	}
	seen.add(item.path)
	attachments.push(item)
}

export function extractDroppedAttachments(dataTransfer: DataTransfer | null, workDir = "") {
	const attachments: ComposerAttachment[] = []
	if (!dataTransfer) return { attachments, missingPath: false }
	const seen = new Set<string>()
	const custom = dataTransfer.getData(GOPANEL_PATH_MIME)
	if (custom) {
		for (const line of custom.split("\n")) {
			if (line.trim()) addDroppedPath(attachments, seen, line, workDir)
		}
	}
	const uriList = dataTransfer.getData("text/uri-list")
	if (uriList) {
		for (const line of uriList.split("\n")) {
			if (!line || line.startsWith("#")) continue
			const decoded = decodeFileUri(line)
			if (decoded) addDroppedPath(attachments, seen, decoded, workDir)
		}
	}
	const files = Array.from(dataTransfer.files || [])
	let missingPath = false
	for (const file of files) {
		const path = (file as File & { path?: string }).path || ""
		if (!path) {
			missingPath = true
			continue
		}
		addDroppedPath(attachments, seen, path, workDir, file)
	}
	if (!attachments.length) {
		const plain = dataTransfer.getData("text/plain").trim()
		if (plain && looksLikeFilePath(plain)) addDroppedPath(attachments, seen, plain, workDir)
	}
	if (attachments.length) missingPath = false
	return { attachments, missingPath }
}
