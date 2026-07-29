export interface AIGroup {
  id: number
  createdAt: string
  name: string
  description: string
  creatorId: number
  memberCount?: number
  taskCount?: number
}

export interface AITask {
  id: number
  createdAt: string
  projectId: number
  title: string
  agentName: string
  workDir: string
  status: string
}

export interface AIMessage {
  id: number
  createdAt: string
  taskId: number
  role: string
  content: string
}
