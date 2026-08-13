package model

import "time"

type WebsiteDiagnosticSetting struct {
	BaseModel
	WebsiteID              uint   `gorm:"column:website_id;not null;uniqueIndex" json:"websiteId"`
	CodeProjectID          uint   `gorm:"column:code_project_id;not null;default:0;index" json:"codeProjectId"`
	ConfiguredByUserID     uint   `gorm:"column:configured_by_user_id;not null;default:0;index" json:"-"`
	HookSecretEncrypted    string `gorm:"column:hook_secret_encrypted;type:text" json:"-"`
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

type WebsiteDiagnosticEvent struct {
	BaseModel
	WebsiteID    uint      `gorm:"column:website_id;not null;index;uniqueIndex:idx_website_event" json:"websiteId"`
	EventID      string    `gorm:"column:event_id;type:varchar(128);not null;uniqueIndex:idx_website_event" json:"eventId"`
	IssueID      uint      `gorm:"column:issue_id;not null;default:0;index" json:"issueId"`
	Source       string    `gorm:"column:source;type:varchar(32);not null;index" json:"source"`
	Kind         string    `gorm:"column:kind;type:varchar(64);not null;index" json:"kind"`
	Severity     string    `gorm:"column:severity;type:varchar(16);not null;index" json:"severity"`
	Fingerprint  string    `gorm:"column:fingerprint;type:varchar(64);not null;index" json:"fingerprint"`
	Title        string    `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Message      string    `gorm:"column:message;type:text" json:"message"`
	Stack        string    `gorm:"column:stack;type:text" json:"stack"`
	RequestID    string    `gorm:"column:request_id;type:varchar(128);index" json:"requestId"`
	SessionID    string    `gorm:"column:session_key;type:varchar(128);index" json:"sessionId"`
	Method       string    `gorm:"column:method;type:varchar(16)" json:"method"`
	Route        string    `gorm:"column:route;type:varchar(512);index" json:"route"`
	HTTPStatus   int       `gorm:"column:http_status;index" json:"httpStatus"`
	BusinessCode string    `gorm:"column:business_code;type:varchar(128);index" json:"businessCode"`
	DurationMS   int64     `gorm:"column:duration_ms" json:"durationMs"`
	Release      string    `gorm:"column:release;type:varchar(128);index" json:"release"`
	Metadata     string    `gorm:"column:metadata;type:text" json:"metadata,omitempty"`
	OccurredAt   time.Time `gorm:"column:occurred_at;not null;index" json:"occurredAt"`
}

func (WebsiteDiagnosticEvent) TableName() string { return "website_diagnostic_events" }

type WebsiteIssue struct {
	BaseModel
	WebsiteID       uint       `gorm:"column:website_id;not null;index;uniqueIndex:idx_website_issue" json:"websiteId"`
	Fingerprint     string     `gorm:"column:fingerprint;type:varchar(64);not null;uniqueIndex:idx_website_issue" json:"fingerprint"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	Severity        string     `gorm:"column:severity;type:varchar(16);not null;index" json:"severity"`
	Title           string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Kind            string     `gorm:"column:kind;type:varchar(64);not null" json:"kind"`
	Route           string     `gorm:"column:route;type:varchar(512);index" json:"route"`
	HTTPStatus      int        `gorm:"column:http_status" json:"httpStatus"`
	BusinessCode    string     `gorm:"column:business_code;type:varchar(128)" json:"businessCode"`
	OccurrenceCount int64      `gorm:"column:occurrence_count;not null;default:0" json:"occurrenceCount"`
	SessionCount    int64      `gorm:"column:session_count;not null;default:0" json:"sessionCount"`
	FirstRelease    string     `gorm:"column:first_release;type:varchar(128)" json:"firstRelease"`
	LatestRelease   string     `gorm:"column:latest_release;type:varchar(128)" json:"latestRelease"`
	FirstSeenAt     time.Time  `gorm:"column:first_seen_at;not null;index" json:"firstSeenAt"`
	LastSeenAt      time.Time  `gorm:"column:last_seen_at;not null;index" json:"lastSeenAt"`
	ConfirmedAt     *time.Time `gorm:"column:confirmed_at" json:"confirmedAt,omitempty"`
	IgnoredAt       *time.Time `gorm:"column:ignored_at" json:"ignoredAt,omitempty"`
	CodeSessionID   uint       `gorm:"column:code_session_id;not null;default:0;index" json:"codeSessionId"`
	CodeTaskID      uint       `gorm:"column:code_task_id;not null;default:0;index" json:"codeTaskId"`
	CodeStatus      string     `gorm:"column:code_status;type:varchar(32)" json:"codeStatus"`
	VerifyRelease   string     `gorm:"column:verify_release;type:varchar(128)" json:"verifyRelease"`
	VerifyStartedAt *time.Time `gorm:"column:verify_started_at" json:"verifyStartedAt,omitempty"`
	ResolvedAt      *time.Time `gorm:"column:resolved_at" json:"resolvedAt,omitempty"`
}

func (WebsiteIssue) TableName() string { return "website_issues" }

type WebsiteDiagnosticTimeline struct {
	BaseModel
	WebsiteID uint   `gorm:"column:website_id;not null;index" json:"websiteId"`
	IssueID   uint   `gorm:"column:issue_id;not null;index" json:"issueId"`
	Type      string `gorm:"column:type;type:varchar(32);not null;index" json:"type"`
	Content   string `gorm:"column:content;type:text" json:"content"`
	UserID    uint   `gorm:"column:user_id;not null;default:0" json:"userId"`
}

func (WebsiteDiagnosticTimeline) TableName() string { return "website_diagnostic_timeline" }

type WebsiteProbe struct {
	BaseModel
	WebsiteID        uint       `gorm:"column:website_id;not null;index" json:"websiteId"`
	Name             string     `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Enabled          bool       `gorm:"column:enabled;not null;default:true;index" json:"enabled"`
	Method           string     `gorm:"column:method;type:varchar(16);not null;default:'GET'" json:"method"`
	Path             string     `gorm:"column:path;type:varchar(512);not null" json:"path"`
	ExpectedStatus   int        `gorm:"column:expected_status;not null;default:200" json:"expectedStatus"`
	ExpectedCode     string     `gorm:"column:expected_code;type:varchar(128)" json:"expectedCode"`
	RequiredFields   string     `gorm:"column:required_fields;type:text" json:"requiredFields"`
	TimeoutMS        int        `gorm:"column:timeout_ms;not null;default:5000" json:"timeoutMs"`
	IntervalSeconds  int        `gorm:"column:interval_seconds;not null;default:60" json:"intervalSeconds"`
	FailureThreshold int        `gorm:"column:failure_threshold;not null;default:3" json:"failureThreshold"`
	FailureCount     int        `gorm:"column:failure_count;not null;default:0" json:"failureCount"`
	LastStatus       string     `gorm:"column:last_status;type:varchar(32)" json:"lastStatus"`
	LastMessage      string     `gorm:"column:last_message;type:text" json:"lastMessage"`
	LastRunAt        *time.Time `gorm:"column:last_run_at;index" json:"lastRunAt,omitempty"`
}

func (WebsiteProbe) TableName() string { return "website_diagnostic_probes" }

type WebsiteDiagnosticNonce struct {
	BaseModel
	WebsiteID uint      `gorm:"column:website_id;not null;uniqueIndex:idx_website_nonce" json:"-"`
	Nonce     string    `gorm:"column:nonce;type:varchar(128);not null;uniqueIndex:idx_website_nonce" json:"-"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index" json:"-"`
}

func (WebsiteDiagnosticNonce) TableName() string { return "website_diagnostic_nonces" }
