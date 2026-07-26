import http from "@/api"

export interface NodeSummary {
	hostname: string
	os: string
	uptime: number
	cpuPercent: number
	cpuTotal: number
	load1: number
	memPercent: number
	memTotal: number
	memUsed: number
	diskMaxPercent: number
	diskMaxPath: string
	containerRunning: number
	containerTotal: number
	containerAbnormal: number
	certTotal: number
	certExpiringCount: number
	certMinDays: number
	version: string
	shotTime: string
}

export interface NodeWarning {
	type: "offline" | "disk" | "cert" | "container" | "unauthorized"
	level: "warn" | "danger"
	value: number
}

export interface NodeItem {
	id: number
	name: string
	addr: string
	entrance: string
	connectMode: string
	skipVerify: boolean
	isProd: boolean
	sort: number
	status: "online" | "offline" | "unauthorized" | "unknown"
	statusMsg: string
	version: string
	lastSeenAt: string
	summary: NodeSummary
	warnings: NodeWarning[]
	hasToken: boolean
	/** 已保存令牌的明文长度（不含内容）。与 tokenLenExpected 不一致说明存错了 */
	tokenLen: number
	/** 节点签发令牌的标准长度，由后端给出，不写死在前端 */
	tokenLenExpected: number
	/** 是否已配置控制令牌。false 表示该节点只能观测，不能代理操作 */
	hasControlToken: boolean
	controlTokenLen: number
}

export interface NodeSaveParams {
	id?: number
	name: string
	addr: string
	accessToken: string
	/** 控制令牌。留空=不修改（编辑态）；"-" = 清除，即关闭该节点的操作能力 */
	controlToken?: string
	entrance?: string
	skipVerify?: boolean
	isProd?: boolean
	sort?: number
}

export const nodeListAPI = () => {
	return http.get<NodeItem[]>(`/node/list`)
}

export const nodeCreateAPI = (params: NodeSaveParams) => {
	return http.post<any>(`/node/create`, params)
}

export const nodeUpdateAPI = (params: NodeSaveParams) => {
	return http.post<any>(`/node/update`, params)
}

export const nodeDeleteAPI = (params: { id: number }) => {
	return http.post<any>(`/node/del`, params)
}

/** 采集单个节点，返回该节点最新状态 */
export const nodeProbeAPI = (id: number) => {
	return http.post<NodeItem>(`/node/probe/${id}`, {})
}

/** 保存前测试连接，不落库 */
export const nodeProbeDraftAPI = (params: NodeSaveParams) => {
	return http.post<{ addr: string; hostname: string; version: string }>(`/node/probe`, params)
}

/** 手动触发一轮全量采集，返回刷新后的列表 */
export const nodeRefreshAPI = () => {
	return http.post<NodeItem[]>(`/node/refresh`, {})
}

/** 本机是否已开启只读接入 */
export const nodeLocalTokenStatusAPI = () => {
	return http.get<{ enabled: boolean }>(`/node/local/token`)
}

/** 签发本机只读令牌，明文仅本次返回 */
export const nodeLocalTokenIssueAPI = () => {
	return http.post<{ accessToken: string }>(`/node/local/token/issue`, {})
}

export const nodeLocalTokenRevokeAPI = () => {
	return http.post<any>(`/node/local/token/revoke`, {})
}

/** 本机是否已开启控制接入（可被主控代理执行写操作） */
export const nodeLocalControlStatusAPI = () => {
	return http.get<{ enabled: boolean }>(`/node/local/control`)
}

/** 签发本机控制令牌。该令牌等价于本机管理员，明文仅本次返回 */
export const nodeLocalControlIssueAPI = () => {
	return http.post<{ accessToken: string }>(`/node/local/control/issue`, {})
}

export const nodeLocalControlRevokeAPI = () => {
	return http.post<any>(`/node/local/control/revoke`, {})
}
