export interface WebsiteDiagnosticSummary {
	configured: boolean
	enabled: boolean
	sourceCount: number
	contentCount: number
	codeProjectId: number
	autoAnalysis: boolean
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
}
