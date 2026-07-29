import http from "@/api"
import type { AIGroup, AITask, AIMessage, CodeExecutor, CodeSession } from "../interface/code"

// === Group APIs ===

export function getAIGroups(params: { page: number; limit: number }) {
	return http.get<{ items: AIGroup[]; total: number }>("/code/groups", params)
}

export function createAIGroup(data: { name: string; description: string }) {
	return http.post<AIGroup>("/code/groups", data)
}

export function getCodeExecutors() {
	return http.get<CodeExecutor[]>("/code/executors")
}

export function createCodeSession(data: { title: string; workDir: string; projectId: number; executorId: string }) {
	return http.post<CodeSession>("/code/sessions", data)
}

// === Task APIs ===

// 获取任务列表
export function getAITasks(params: { page: number; limit: number; projectId?: number }) {
	return http.get<{ items: AITask[]; total: number }>("/code/tasks", params)
}

// 获取某个任务的消息记录
export function getAITaskMessages(taskId: number) {
	return http.get<AIMessage[]>(`/code/tasks/${taskId}/messages`)
}

// 重命名任务
export function updateAITask(taskId: number, title: string) {
	return http.put(`/code/tasks/${taskId}`, { title })
}

// 删除任务
export function deleteAITask(taskId: number) {
	return http.delete(`/code/tasks/${taskId}`)
}
