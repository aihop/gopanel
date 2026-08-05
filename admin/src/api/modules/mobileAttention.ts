import axios from "axios"
import type { ResultData } from "@/api/interface"
import type { MobileAttentionAction, MobileAttentionItem } from "../interface/mobileControlPlane"

const mobileAttentionHttp = axios.create({
	baseURL: import.meta.env.VITE_API_URL as string,
	timeout: 15000,
	withCredentials: true
})

mobileAttentionHttp.interceptors.request.use(config => {
	config.headers.set("X-Mobile-Request", "1")
	return config
})

export async function getMobileAttention() {
	const response =
		await mobileAttentionHttp.get<ResultData<{ items: MobileAttentionItem[]; total: number }>>(
			"/mobile/app/attention"
		)
	if (response.data.code !== 0) throw new Error(response.data.msg || response.data.message || "Request failed")
	return { ...response.data.data, items: response.data.data.items || [] }
}

export async function executeMobileAttentionAction(action: MobileAttentionAction) {
	if (action.method !== "POST" || !action.path.startsWith("/api/mobile/app/")) {
		throw new Error("Unsupported mobile attention action")
	}
	const response = await mobileAttentionHttp.post<ResultData<void>>(action.path.replace(/^\/api/, ""), {})
	if (response.data.code !== 0) throw new Error(response.data.msg || response.data.message || "Request failed")
}
