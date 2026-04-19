import http from "@/api"
import { AIGroup, AITask, AIMessage } from "../interface/ai_agent";

// === Group APIs ===

export function getAIGroups(params: { page: number; pageSize: number }) {
  return http.get<{ items: AIGroup[]; total: number }>("/ai/groups", params)
}

export function createAIGroup(data: { name: string; description: string }) {
  return http.post<AIGroup>("/ai/groups", data)
}

// === Task APIs ===

// 获取任务列表
export function getAITasks(params: { page: number; pageSize: number; projectId?: number }) {
  return http.get<{ items: AITask[]; total: number }>("/ai/tasks", params)
}

// 获取某个任务的消息记录
export function getAITaskMessages(taskId: number) {
  return http.get<AIMessage[]>(`/ai/tasks/${taskId}/messages`)
}

// 重命名任务
export function updateAITask(taskId: number, title: string) {
  return http.put(`/ai/tasks/${taskId}`, { title })
}

// 删除任务
export function deleteAITask(taskId: number) {
  return http.delete(`/ai/tasks/${taskId}`)
}
