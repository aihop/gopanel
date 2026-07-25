import http from "@/api"

export async function AgentStatusAPI() {
	return http.get<any>(`/agent/status`)
}

export async function AgentEnsureAPI() {
	return http.post<any>(`/agent/ensure`, {})
}

export async function AgentUpdateCheckAPI() {
	return http.get<any>(`/agent/update-check`)
}

