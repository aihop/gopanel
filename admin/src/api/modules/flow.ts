import http from "@/api"
import type { ResPage } from "@/api/interface"
import type { Flow } from "@/api/interface/flow"

export function getFlowPage(params: { page: number; limit: number }) {
	return http.get<ResPage<Flow.Item>>("/flow/list", params)
}

export function createFlow(input: Flow.CreateInput) {
	return http.post<Flow.Item>("/flow", input)
}

export function updateFlow(id: number, input: Flow.UpdateInput) {
	return http.put<Flow.Item>(`/flow/${id}`, input)
}

export function deleteFlow(id: number) {
	return http.delete<void>(`/flow/${id}`)
}

export function getFlowRunPage(params: { flowId?: number; page: number; limit: number }) {
	return http.get<ResPage<Flow.Run>>("/flow/runs", params)
}

export function getFlowRun(id: number) {
	return http.get<Flow.Run>(`/flow/runs/${id}`)
}

export function resumeFlowRun(id: number) {
	return http.post<Flow.Run>(`/flow/runs/${id}/resume`)
}

export function rebuildFlowRun(id: number) {
	return http.post<Flow.Run>(`/flow/runs/${id}/rebuild`)
}

export function createFlowRun(input: Flow.RunCreateInput) {
	return http.post<Flow.Run>("/flow/runs", input)
}

export function getFlowCodeDeliverySources(flowId: number) {
	return http.get<Flow.CodeDeliverySource[]>(`/flow/${flowId}/code-deliveries`)
}

export function getFlowCodeBaselineSource(flowId: number) {
	return http.get<Flow.CodeBaselineSource>(`/flow/${flowId}/code-baseline`)
}
