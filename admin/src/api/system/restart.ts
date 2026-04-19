import type { Response } from "@/types/api/response"

import { request } from "@/utils/network"
export async function systemRestartPanelAPI(data: any) {
	return await request<Response<any>>(`/system/restart/panel`, {
		method: "POST",
		data
	})
}

export async function systemRestartSystemAPI(data: any) {
	return await request<Response<any>>(`/system/restart/system`, {
		method: "POST",
		data
	})
}
