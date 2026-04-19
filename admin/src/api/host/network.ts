import type { Response } from "@/types/api/response"

import { request } from "@/utils/network"

// 此处假设网络数据的请求参数和响应结构与进程类似，你可能需要根据实际情况调整
export async function hostsNetworkListAPI(data: any) {
	return await request<Response<any>>(`/hosts/network/list`, {
		method: "POST",
		data
	})
}
