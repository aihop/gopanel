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

// 进入面板后触发一次 gp-agent 自动更新（后端有防重入+节流）
export async function AgentAutoUpdateAPI() {
	return http.post<any>(`/agent/auto-update`, {})
}

