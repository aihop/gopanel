import type { AIProviderAccount } from "./aiAccounts"

export interface SystemDiagnosticMessage {
	id: number
	createdAt: string
	role: "user" | "agent"
	content: string
}

export interface SystemDiagnosticSession {
	id: number
	title: string
	providerModel?: string
}

export interface SystemDiagnosticControlPlane {
	healthy: boolean
	autoRepairable: boolean
	nextAction: string
}

export interface SystemDiagnosticSnapshot {
	checkedAt: string
	controlPlane: SystemDiagnosticControlPlane
	counts?: Record<string, number>
	recentFailedJobs?: unknown[]
	recentFailedAIRuns?: unknown[]
	recentFailedOperations?: unknown[]
}

export interface SystemDiagnosticState {
	session: SystemDiagnosticSession
	messages: SystemDiagnosticMessage[]
	accounts: AIProviderAccount[]
	snapshot: SystemDiagnosticSnapshot
}

export interface SystemDiagnosticChatResult {
	session: SystemDiagnosticSession
	userMessage: SystemDiagnosticMessage
	assistantMessage: SystemDiagnosticMessage
}
