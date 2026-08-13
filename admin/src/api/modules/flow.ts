import http from "@/api"
import type { ResPage } from "@/api/interface"
import type { Flow } from "@/api/interface/flow"

export function getFlowPage(params: { page: number; limit: number }) {
	return http.get<ResPage<Flow.Item>>("/flow/list", params)
}

export function createFlow(input: Flow.CreateInput) {
	return http.post<Flow.Item>("/flow", input)
}
