import http from "@/api"

export async function AgentStatusAPI() {
	return http.get<any>(`/agent/status`)
}

export async function AgentEnsureAPI() {
	return http.post<any>(`/agent/ensure`, {})
}

