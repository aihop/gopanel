import http from "@/api"

export interface ControlPlaneComponentStatus {
	name: string
	state: "healthy" | "missing" | "service_stopped" | "socket_missing" | "permission_denied" | "config_mismatch"
	healthy: boolean
	installed: boolean
	reachable: boolean
	socketPath?: string
	version?: string
	error?: string
	commands?: string[]
}

export interface ControlPlaneStatus {
	healthy: boolean
	autoRepairable: boolean
	requiresSudo: boolean
	nextAction: "none" | "repair_agent" | "repair_gpc"
	checkedAt: number
	gpc: ControlPlaneComponentStatus
	agent: ControlPlaneComponentStatus
}

export async function AgentStatusAPI() {
	return http.get<any>(`/agent/status`)
}

export async function ControlPlaneStatusAPI() {
	return http.get<ControlPlaneStatus>(`/agent/control-plane/status`)
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
