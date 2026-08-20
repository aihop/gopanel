export function clampLineRange(start: number, end: number, total = Math.max(start, end, 1)) {
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

export function formatFileLineRef(path: string, start: number, end: number) {
	const range = clampLineRange(start, end)
	return `@attach ${path}:${range.start}-${range.end}`
}

export function selectionLineRange(selection?: { startLineNumber?: number; endLineNumber?: number } | null) {
	const start = selection?.startLineNumber || 1
	const end = selection?.endLineNumber || start
	return clampLineRange(start, end)
}

export function nextMatchIndex(current: number, total: number, step: number) {
	if (total <= 0) return 0
	return (current + step + total) % total
}

export function structureAncestorDirs(filePath: string) {
	const parts = filePath.replaceAll("\\", "/").split("/").filter(Boolean)
	const dirs: string[] = []
	let current = ""
	for (const part of parts.slice(0, -1)) {
		current = current ? `${current}/${part}` : part
		dirs.push(current)
	}
	return dirs
}
