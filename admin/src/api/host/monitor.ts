import type { Response } from "@/types/api/response"

import { request } from "@/utils/network"
export async function hostsMonitorListAPI(data: any) {
	return await request<Response<any>>(`/hosts/monitor/list`, {
		method: "POST",
		data
	})
}
