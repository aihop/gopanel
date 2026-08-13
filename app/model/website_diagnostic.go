package model

type WebsiteDiagnosticSetting struct {
	BaseModel
	WebsiteID              uint   `gorm:"column:website_id;not null;uniqueIndex" json:"websiteId"`
	CodeProjectID          uint   `gorm:"column:code_project_id;not null;default:0;index" json:"codeProjectId"`
	Enabled                bool   `gorm:"column:enabled;not null;default:false" json:"enabled"`
	CaddyMonitoring        bool   `gorm:"column:caddy_monitoring;not null" json:"caddyMonitoring"`
	ActiveProbes           bool   `gorm:"column:active_probes;not null;default:false" json:"activeProbes"`
	BackendHook            bool   `gorm:"column:backend_hook;not null;default:false" json:"backendHook"`
	BrowserHook            bool   `gorm:"column:browser_hook;not null;default:false" json:"browserHook"`
	AutoAnalysis           bool   `gorm:"column:auto_analysis;not null;default:false" json:"autoAnalysis"`
	MonitorHTTP4xx         bool   `gorm:"column:monitor_http_4xx;not null" json:"monitorHttp4xx"`
	MonitorHTTP5xx         bool   `gorm:"column:monitor_http_5xx;not null" json:"monitorHttp5xx"`
	MonitorUpstreamErrors  bool   `gorm:"column:monitor_upstream_errors;not null" json:"monitorUpstreamErrors"`
	MonitorSlowRequests    bool   `gorm:"column:monitor_slow_requests;not null" json:"monitorSlowRequests"`
	MonitorBusinessErrors  bool   `gorm:"column:monitor_business_errors;not null" json:"monitorBusinessErrors"`
	MonitorBrowserErrors   bool   `gorm:"column:monitor_browser_errors;not null" json:"monitorBrowserErrors"`
	MonitorResourceErrors  bool   `gorm:"column:monitor_resource_errors;not null" json:"monitorResourceErrors"`
	SlowRequestThresholdMS int    `gorm:"column:slow_request_threshold_ms;not null;default:1500" json:"slowRequestThresholdMs"`
	TriggerCount           int    `gorm:"column:trigger_count;not null;default:5" json:"triggerCount"`
	TriggerWindowMinutes   int    `gorm:"column:trigger_window_minutes;not null;default:10" json:"triggerWindowMinutes"`
	RetentionDays          int    `gorm:"column:retention_days;not null;default:7" json:"retentionDays"`
	DefaultExecutorID      string `gorm:"column:default_executor_id;type:varchar(64);not null;default:'codex'" json:"defaultExecutorId"`
	ApprovalPolicy         string `gorm:"column:approval_policy;type:varchar(32);not null;default:'safe_auto'" json:"approvalPolicy"`
}

func (WebsiteDiagnosticSetting) TableName() string {
	return "website_diagnostic_settings"
}
