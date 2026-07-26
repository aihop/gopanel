import type { ResultData } from "@/api/interface"
import http from "@/api"

/**
 * 节点作用域的 API 客户端。
 *
 * 刻意做成"按节点显式创建"而不是全局改写 axios 拦截器：拦截器是全局的，
 * 一旦按"当前节点"改写 URL，后台组件（比如节点细条每分钟的轮询）就有可能
 * 被误路由到远程节点。显式传 nodeId 让每次调用的目标一目了然。
 *
 * 复用全局 http 实例是为了继承它已有的 x-auth、EntranceCode、错误提示、
 * 登录失效跳转等行为——主控侧的用户鉴权与本机请求完全一致。
 */
export interface NodeApi {
	get: <T>(path: string, params?: object) => Promise<ResultData<T>>
	post: <T>(path: string, params?: object, timeout?: number) => Promise<ResultData<T>>
}

const PROXY_PREFIX = "/node-proxy"

function join(nodeId: number, path: string): string {
	const clean = path.startsWith("/") ? path : `/${path}`
	return `${PROXY_PREFIX}/${nodeId}${clean}`
}

export function nodeApi(nodeId: number): NodeApi {
	return {
		get: <T>(path: string, params?: object) => http.get<T>(join(nodeId, path), params),
		post: <T>(path: string, params?: object, timeout?: number) => http.post<T>(join(nodeId, path), params, timeout)
	}
}

// ---- 远程节点上的具体操作 ----

export interface RemoteContainer {
	containerID: string
	name: string
	imageName: string
	state: string
	createTime: string
	runTime: string
	ports?: string[]
}

/** 远程节点的容器列表。参数形状与本机 /container/list 完全一致 */
export const remoteContainerListAPI = (nodeId: number, params: { page: number; limit: number; name?: string }) => {
	return nodeApi(nodeId).post<{ total: number; items: RemoteContainer[] }>(`/container/list`, {
		page: params.page,
		limit: params.limit,
		name: params.name || "",
		// 这四个字段后端有 validate:"required,oneof=..."，不能省
		state: "all",
		orderBy: "created_at",
		order: "null",
		filters: ""
	})
}

/** 远程容器启停。operation 取值与本机一致：start/stop/restart/kill/pause/unpause/remove */
export const remoteContainerOperateAPI = (nodeId: number, names: string[], operation: string) => {
	return nodeApi(nodeId).post<any>(`/container/operate`, { names, operation })
}

/** 远程节点的网站列表 */
export const remoteWebsiteListAPI = (nodeId: number) => {
	return nodeApi(nodeId).post<any>(`/website/list`, {})
}

/** 远程节点的系统信息 */
export const remoteOsInfoAPI = (nodeId: number) => {
	return nodeApi(nodeId).get<any>(`/dashboard/base/os`)
}
