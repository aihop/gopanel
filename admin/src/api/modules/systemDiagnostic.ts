import http from "@/api"
import GlobalStore from "@/store/modules/global"
import { useAuthStore } from "@/store/auth"
import { enc } from "crypto-js"
import type {
	SystemDiagnosticChatResult,
	SystemDiagnosticState,
	SystemDiagnosticStreamCallbacks,
	SystemDiagnosticStreamStart
} from "../interface/systemDiagnostic"

export function getSystemDiagnosticState() {
	return http.get<SystemDiagnosticState>("/code/system-diagnostics/state", undefined, { timeout: 15000 })
}

export async function streamSystemDiagnostic(content: string, accountId: number, callbacks: SystemDiagnosticStreamCallbacks, signal?: AbortSignal) {
	const apiUrl = (window as typeof window & { __VITE_API_URL__?: string }).__VITE_API_URL__ || import.meta.env.VITE_API_URL || "/api"
	const globalStore = GlobalStore()
	const headers: Record<string, string> = {
		"Content-Type": "application/json",
		Accept: "text/event-stream",
		"Accept-Language": globalStore.language === "tw" ? "zh-Hant" : globalStore.language,
		"x-auth": useAuthStore().auth || ""
	}
	if (globalStore.entrance) headers.EntranceCode = enc.Base64.stringify(enc.Utf8.parse(globalStore.entrance))
	const response = await fetch(`${apiUrl}/code/system-diagnostics/chat`, {
		method: "POST",
		credentials: "include",
		signal,
		headers,
		body: JSON.stringify({ content, accountId })
	})
	const contentType = response.headers.get("content-type") || ""
	if (!response.ok || !contentType.includes("text/event-stream")) {
		const result = await response.json().catch(() => null)
		throw new Error(result?.msg || `HTTP ${response.status}`)
	}
	if (!response.body) throw new Error("Stream body is unavailable")
	await readSystemDiagnosticStream(response.body, callbacks)
}

async function readSystemDiagnosticStream(stream: ReadableStream<Uint8Array>, callbacks: SystemDiagnosticStreamCallbacks) {
	const reader = stream.getReader()
	const decoder = new TextDecoder()
	let buffer = ""
	let completed = false
	while (true) {
		const { value, done } = await reader.read()
		buffer += decoder.decode(value, { stream: !done }).replaceAll("\r\n", "\n")
		let boundary = buffer.indexOf("\n\n")
		while (boundary >= 0) {
			const frame = buffer.slice(0, boundary)
			buffer = buffer.slice(boundary + 2)
			if (handleSystemDiagnosticFrame(frame, callbacks) === "done") completed = true
			boundary = buffer.indexOf("\n\n")
		}
		if (done) break
	}
	if (!completed) throw new Error("Diagnostic stream ended before completion")
}

function handleSystemDiagnosticFrame(frame: string, callbacks: SystemDiagnosticStreamCallbacks) {
	let event = "message"
	const data: string[] = []
	for (const line of frame.split("\n")) {
		if (line.startsWith("event:")) event = line.slice(6).trim()
		if (line.startsWith("data:")) data.push(line.slice(5).trimStart())
	}
	if (!data.length) return event
	const payload = JSON.parse(data.join("\n"))
	if (event === "start") callbacks.onStart?.(payload as SystemDiagnosticStreamStart)
	if (event === "delta") callbacks.onDelta?.(String(payload.content || ""))
	if (event === "done") callbacks.onDone?.(payload as SystemDiagnosticChatResult)
	if (event === "error") throw new Error(payload.message || "Diagnostic stream failed")
	return event
}
