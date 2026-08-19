import axios from "axios"
import type { ResultData } from "@/api/interface"

/**
 * 移动端共用的 http 客户端与响应解包。
 * 单独成文件，让设备/资源类接口与 Code 会话类接口可以分开维护。
 */
export const mobileHttp = axios.create({
	baseURL: import.meta.env.VITE_API_URL as string,
	timeout: 15000,
	withCredentials: true
})

export function mobileNodePath(nodeId: number, path: string) {
	const clean = path.startsWith("/") ? path : `/${path}`
	return nodeId > 0 ? `/mobile/app/node-proxy/${nodeId}/mobile/app${clean}` : `/mobile/app${clean}`
}

mobileHttp.interceptors.request.use(config => {
	config.headers.set("X-Mobile-Request", "1")
	return config
})

export async function mobileRequest<T>(request: Promise<{ data: ResultData<T> }>) {
	const response = await request
	if (response.data.code !== 0) {
		throw new Error(response.data.msg || response.data.message || "Request failed")
	}
	return response.data.data
}
