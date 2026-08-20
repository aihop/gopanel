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

export interface CodeStructureSearchHit {
	name: string
	path: string
	isDir: boolean
	extension: string
	kind: "name" | "content"
	line?: number
	preview?: string
}

export interface CodeStructureSearchResult {
	query: string
	hits: CodeStructureSearchHit[]
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

export function searchCodeSessionStructure(sessionId: number, query: string) {
	return http.get<CodeStructureSearchResult>(`/code/sessions/${sessionId}/structure/search`, { q: query }, { timeout: 15000 })
}

export function getCodeSessionFile(sessionId: number, path: string) {
	return http.get<CodeSessionFile>(`/code/sessions/${sessionId}/file`, { path }, { timeout: 10000 })
}

export function saveCodeSessionFile(sessionId: number, path: string, content: string, baseVersion: string) {
	return http.put<{ path: string; size: number; version: string }>(`/code/sessions/${sessionId}/file`, { path, content, baseVersion })
}
