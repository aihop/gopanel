import type { Response } from "@/types/api/response"

import { request } from "@/utils/network"
export async function hostsMonitorSearchAPI(data: any) {
	return await request<Response<any>>(`/hosts/monitor/search`, {
		method: "POST",
		data
	})
}
