import http from "@/api"
import type { ResPage } from "@/api/interface"
import type { CreateHostTerminalRequest, HostTerminalAuditEvent, HostTerminalCapabilities, HostTerminalSession } from "@/api/interface/hostTerminal"

export function getHostTerminalCapabilities() {
	return http.get<HostTerminalCapabilities>("/host/terminal/capabilities")
}

export function createHostTerminalSession(request: CreateHostTerminalRequest) {
	return http.post<HostTerminalSession>("/host/terminal/sessions", request)
}

export function listHostTerminalSessions(page = 1, limit = 50) {
	return http.get<ResPage<HostTerminalSession>>("/host/terminal/sessions", { page, limit })
}

export function stopHostTerminalSession(sessionId: number) {
	return http.post(`/host/terminal/sessions/${sessionId}/stop`)
}

export function getHostTerminalAuditEvents(sessionId: number) {
	return http.get<HostTerminalAuditEvent[]>(`/host/terminal/sessions/${sessionId}/audit-events`)
}
