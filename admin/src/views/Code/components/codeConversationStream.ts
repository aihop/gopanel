import GlobalStore from "@/store/modules/global"
import { useAuthStore } from "@/store/auth"
import { enc } from "crypto-js"

export interface ConversationStreamPayload {
	type?: string
	runId?: number
	content?: string
	status?: string
	message?: string
	/** 活动种类：command / file / tool / search / thinking，文案由前端 i18n 决定。 */
	activityKind?: string
	/** 活动细节，比如正在执行的命令或正在改的文件。 */
	activity?: string
}

export interface ConversationStreamCallbacks {
	onSnapshot?: (payload: ConversationStreamPayload) => void
	onDelta?: (payload: ConversationStreamPayload) => void
	onDone?: (payload: ConversationStreamPayload) => void
}

export async function streamCodeConversation(
	sessionId: number,
	callbacks: ConversationStreamCallbacks,
	signal?: AbortSignal,
) {
	const apiUrl = (window as typeof window & { __VITE_API_URL__?: string }).__VITE_API_URL__ || import.meta.env.VITE_API_URL || "/api"
	const globalStore = GlobalStore()
	const headers: Record<string, string> = {
		Accept: "text/event-stream",
		"Accept-Language": globalStore.language === "tw" ? "zh-Hant" : globalStore.language,
		"x-auth": useAuthStore().auth || "",
	}
	if (globalStore.entrance) headers.EntranceCode = enc.Base64.stringify(enc.Utf8.parse(globalStore.entrance))
	const response = await fetch(`${apiUrl}/code/sessions/${sessionId}/conversation/stream`, {
		credentials: "include",
		signal,
		headers,
	})
	const contentType = response.headers.get("content-type") || ""
	if (!response.ok || !contentType.includes("text/event-stream") || !response.body) {
		const result = await response.json().catch(() => null)
		throw new Error(result?.msg || result?.message || `HTTP ${response.status}`)
	}
	await readConversationStream(response.body, callbacks)
}

async function readConversationStream(stream: ReadableStream<Uint8Array>, callbacks: ConversationStreamCallbacks) {
	const reader = stream.getReader()
	const decoder = new TextDecoder()
	let buffer = ""
	while (true) {
		const { value, done } = await reader.read()
		buffer += decoder.decode(value, { stream: !done }).replaceAll("\r\n", "\n")
		let boundary = buffer.indexOf("\n\n")
		while (boundary >= 0) {
			handleConversationStreamFrame(buffer.slice(0, boundary), callbacks)
			buffer = buffer.slice(boundary + 2)
			boundary = buffer.indexOf("\n\n")
		}
		if (done) break
	}
}

function handleConversationStreamFrame(frame: string, callbacks: ConversationStreamCallbacks) {
	let event = "message"
	const data: string[] = []
	for (const line of frame.split("\n")) {
		if (line.startsWith(":")) continue
		if (line.startsWith("event:")) event = line.slice(6).trim()
		if (line.startsWith("data:")) data.push(line.slice(5).trimStart())
	}
	if (!data.length) return
	const payload = JSON.parse(data.join("\n")) as ConversationStreamPayload
	// activity 只带活动状态、不带正文，复用快照回调即可——
	// applyStreamPayload 在 content 缺省时会保留已有正文。
	if (event === "snapshot" || event === "activity") callbacks.onSnapshot?.(payload)
	if (event === "delta") callbacks.onDelta?.(payload)
	if (event === "done") callbacks.onDone?.(payload)
}
