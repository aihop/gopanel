import http from "@/api"

export interface CodeStructureEntry {
	name: string
	path: string
	isDir: boolean
	extension: string
}

export interface CodeStructureResult {
	path: string
	entries: CodeStructureEntry[]
	truncated: boolean
}

export interface CodeSessionFile {
	path: string
	content: string
	extension: string
	size: number
	version: string
}

export function getCodeSessionStructure(sessionId: number, path = "") {
	return http.get<CodeStructureResult>(`/code/sessions/${sessionId}/structure`, { path }, { timeout: 10000 })
}

export function getCodeSessionFile(sessionId: number, path: string) {
	return http.get<CodeSessionFile>(`/code/sessions/${sessionId}/file`, { path }, { timeout: 10000 })
}

export function saveCodeSessionFile(sessionId: number, path: string, content: string, baseVersion: string) {
	return http.put<{ path: string; size: number; version: string }>(`/code/sessions/${sessionId}/file`, { path, content, baseVersion })
}
