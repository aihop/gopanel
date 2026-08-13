export type SecurityRiskLevel = "info" | "low" | "medium" | "high" | "critical"
export type SecurityEventStatus = "pending" | "firing" | "resolved"

export interface SecurityEvidence {
  source: string
  description: string
  count: number
  samples?: string[]
}

export interface SecurityRecommendedAction {
  action: string
  risk: string
  requiresApproval: boolean
}

export interface SecurityEvent {
  id: number
  sourceType: string
  sourceId: number
  sourceName: string
  eventType: string
  level: SecurityRiskLevel
  status: SecurityEventStatus
  summary: string
  evidence: string
  hitCount: number
  firstSeenAt: string
  lastSeenAt: string
  resolvedAt?: string
  analysisStatus: string
  aiConclusion: string
	aiEvidence: string
  suggestedActions: string
  confidence: number
  aiModel: string
  analyzedAt?: string
  analysisError: string
  notifyStatus: string
  notifyError: string
}

export interface SecurityMonitoringConfig {
  id?: number
  enabled: boolean
  websiteEnabled: boolean
  sshEnabled: boolean
  panelEnabled: boolean
  aiEnabled: boolean
	aiProviderAccountId: number
  aiIntervalMinutes: number
  aiDailyTokenBudget: number
  maxBatchBytes: number
  maxBatchLines: number
  requestPerMinute: number
  notFoundPerMinute: number
  serverErrorPerMinute: number
  loginFailurePerMinute: number
  sshFailurePerMinute: number
  debounceTimes: number
  resolveAfterMinutes: number
}
