import http from "@/api"
import type { SystemDiagnosticChatResult, SystemDiagnosticState } from "../interface/systemDiagnostic"

export function getSystemDiagnosticState() {
	return http.get<SystemDiagnosticState>("/code/system-diagnostics/state", undefined, { timeout: 15000 })
}

export function chatSystemDiagnostic(content: string, accountId: number) {
	return http.post<SystemDiagnosticChatResult>("/code/system-diagnostics/chat", { content, accountId }, 120000)
}
