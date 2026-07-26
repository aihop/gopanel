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

// 手动更新 gp-agent（用户点按钮才调用）。返回日志名，用 /agent/ensure/logs?log=xxx 看过程
export async function AgentUpdateAPI() {
	return http.post<any>(`/agent/update`, {})
}

