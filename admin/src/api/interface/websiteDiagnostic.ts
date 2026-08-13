export interface WebsiteDiagnosticSummary {
	configured: boolean
	enabled: boolean
	sourceCount: number
	contentCount: number
	codeProjectId: number
	autoAnalysis: boolean
	openCount: number
	reopenedCount: number
	processingCount: number
}

export type WebsiteIssueStatus = "open" | "confirmed" | "ignored" | "code_processing" | "fix_ready" | "verifying" | "resolved" | "reopened"

export interface WebsiteDiagnosticEvent {
	id: number
	source: "backend" | "browser" | "caddy" | "probe"
	kind: string
	severity: "info" | "warning" | "error" | "critical"
	title: string
	message: string
	stack: string
	requestId: string
	sessionId: string
	method: string
	route: string
	httpStatus: number
	businessCode: string
	durationMs: number
	release: string
	occurredAt: string
}

export interface WebsiteDiagnosticTimeline {
	id: number
	type: string
	content: string
	userId: number
	createdAt: string
}

export interface WebsiteIssue {
	id: number
	websiteId: number
	fingerprint: string
	status: WebsiteIssueStatus
	severity: string
	title: string
	kind: string
	route: string
	httpStatus: number
	businessCode: string
	occurrenceCount: number
	sessionCount: number
	firstRelease: string
	latestRelease: string
	firstSeenAt: string
	lastSeenAt: string
	codeSessionId: number
	codeTaskId: number
	codeStatus: string
	verifyRelease: string
}

export interface WebsiteIssueDetail {
	issue: WebsiteIssue
	events: WebsiteDiagnosticEvent[]
	timeline: WebsiteDiagnosticTimeline[]
}

export interface WebsiteProbe {
	id: number
	websiteId: number
	name: string
	enabled: boolean
	method: "GET" | "HEAD"
	path: string
	expectedStatus: number
	expectedCode: string
	requiredFields: string
	timeoutMs: number
	intervalSeconds: number
	failureThreshold: number
	failureCount: number
	lastStatus: string
	lastMessage: string
	lastRunAt?: string
}

export interface WebsiteDiagnosticSetting {
	websiteId: number
	codeProjectId: number
	enabled: boolean
	caddyMonitoring: boolean
	activeProbes: boolean
	backendHook: boolean
	browserHook: boolean
	autoAnalysis: boolean
	monitorHttp4xx: boolean
	monitorHttp5xx: boolean
	monitorUpstreamErrors: boolean
	monitorSlowRequests: boolean
	monitorBusinessErrors: boolean
	monitorBrowserErrors: boolean
	monitorResourceErrors: boolean
	slowRequestThresholdMs: number
	triggerCount: number
	triggerWindowMinutes: number
	retentionDays: number
	defaultExecutorId: string
	approvalPolicy: "manual" | "safe_auto" | "full_auto"
	configured: boolean
	trackingDir: string
	hookSecretConfigured: boolean
	remoteEndpoint: string
}
