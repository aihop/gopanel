import type { CodeApproval, CodeSession } from "./code"
import type { Dashboard } from "./dashboard"

export interface MobileOverview {
	system: Dashboard.CurrentInfo
	sessions: CodeSession[]
	sessionTotal: number
	pendingApprovals: CodeApproval[]
	serverTime: string
}

export interface MobileAttentionAction {
	type: "approve" | "reject" | "retry_initialization" | "open_session"
	label: string
	method: "POST" | ""
	path: string
	requiresConfirmation: boolean
}

export interface MobileAttentionItem {
	id: string
	type: string
	severity: "warning" | "error"
	title: string
	summary: string
	sessionId: number
	taskId: number
	approvalId: number
	updatedAt: string
	actions: MobileAttentionAction[]
}
