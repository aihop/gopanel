package model

import "time"

const (
	SecurityEventPending  = "pending"
	SecurityEventFiring   = "firing"
	SecurityEventResolved = "resolved"

	SecurityAnalysisPending   = "pending"
	SecurityAnalysisRunning   = "running"
	SecurityAnalysisCompleted = "completed"
	SecurityAnalysisFailed    = "failed"
	SecurityAnalysisSkipped   = "skipped"
)

type SecurityMonitoringConfig struct {
	BaseModel
	Enabled               bool `json:"enabled"`
	WebsiteEnabled        bool `json:"websiteEnabled"`
	SSHEnabled            bool `json:"sshEnabled"`
	PanelEnabled          bool `json:"panelEnabled"`
	AIEnabled             bool `json:"aiEnabled"`
	AIProviderAccountID   uint `json:"aiProviderAccountId"`
	AIIntervalMinutes     int  `json:"aiIntervalMinutes"`
	AIDailyTokenBudget    int  `json:"aiDailyTokenBudget"`
	MaxBatchBytes         int  `json:"maxBatchBytes"`
	MaxBatchLines         int  `json:"maxBatchLines"`
	RequestPerMinute      int  `json:"requestPerMinute"`
	NotFoundPerMinute     int  `json:"notFoundPerMinute"`
	ServerErrorPerMinute  int  `json:"serverErrorPerMinute"`
	LoginFailurePerMinute int  `json:"loginFailurePerMinute"`
	SSHFailurePerMinute   int  `json:"sshFailurePerMinute"`
	DebounceTimes         int  `json:"debounceTimes"`
	ResolveAfterMinutes   int  `json:"resolveAfterMinutes"`
}

func (SecurityMonitoringConfig) TableName() string { return "security_monitoring_configs" }

type SecurityLogCursor struct {
	BaseModel
	SourceType    string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_security_cursor_source" json:"sourceType"`
	SourceID      uint      `gorm:"not null;uniqueIndex:idx_security_cursor_source" json:"sourceId"`
	Path          string    `gorm:"type:text" json:"path"`
	FileIdentity  string    `gorm:"type:varchar(128)" json:"fileIdentity"`
	Offset        int64     `json:"offset"`
	LastEventAt   time.Time `json:"lastEventAt"`
	LastScannedAt time.Time `json:"lastScannedAt"`
	Processed     int64     `json:"processed"`
	Malformed     int64     `json:"malformed"`
	Dropped       int64     `json:"dropped"`
}

func (SecurityLogCursor) TableName() string { return "security_log_cursors" }

type SecurityEvent struct {
	BaseModel
	SourceType  string     `gorm:"type:varchar(32);not null;index" json:"sourceType"`
	SourceID    uint       `gorm:"not null;index" json:"sourceId"`
	SourceName  string     `gorm:"type:varchar(255)" json:"sourceName"`
	EventType   string     `gorm:"type:varchar(64);not null;index" json:"eventType"`
	Level       string     `gorm:"type:varchar(16);not null;index" json:"level"`
	Status      string     `gorm:"type:varchar(16);not null;index" json:"status"`
	Fingerprint string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"fingerprint"`
	Summary     string     `gorm:"type:varchar(512)" json:"summary"`
	Evidence    string     `gorm:"type:text" json:"evidence"`
	Value       float64    `json:"value"`
	HitCount    int        `json:"hitCount"`
	FirstSeenAt time.Time  `json:"firstSeenAt"`
	LastSeenAt  time.Time  `json:"lastSeenAt"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`

	AnalysisStatus   string     `gorm:"type:varchar(16);not null;index" json:"analysisStatus"`
	AIConclusion     string     `gorm:"type:text" json:"aiConclusion"`
	AIEvidence       string     `gorm:"type:text" json:"aiEvidence"`
	SuggestedActions string     `gorm:"type:text" json:"suggestedActions"`
	Confidence       int        `json:"confidence"`
	AIModel          string     `gorm:"type:varchar(255)" json:"aiModel"`
	AITokens         int64      `json:"aiTokens"`
	AnalyzedAt       *time.Time `json:"analyzedAt,omitempty"`
	AnalysisError    string     `gorm:"type:text" json:"analysisError"`

	LastNotifiedAt      *time.Time `json:"lastNotifiedAt,omitempty"`
	LastAINotifiedAt    *time.Time `json:"lastAiNotifiedAt,omitempty"`
	LastNotifyAttemptAt *time.Time `json:"lastNotifyAttemptAt,omitempty"`
	NotifyStatus        string     `gorm:"type:varchar(16)" json:"notifyStatus"`
	NotifyError         string     `gorm:"type:text" json:"notifyError"`
}

func (SecurityEvent) TableName() string { return "security_events" }

type SecurityAnalysisRun struct {
	BaseModel
	EventID      uint       `gorm:"not null;index" json:"eventId"`
	ProviderID   uint       `gorm:"index" json:"providerId"`
	Model        string     `gorm:"type:varchar(255)" json:"model"`
	Status       string     `gorm:"type:varchar(16);not null;index" json:"status"`
	PromptDigest string     `gorm:"type:varchar(64)" json:"promptDigest"`
	Output       string     `gorm:"type:text" json:"output"`
	ErrorMessage string     `gorm:"type:text" json:"errorMessage"`
	InputTokens  int64      `json:"inputTokens"`
	OutputTokens int64      `json:"outputTokens"`
	TotalTokens  int64      `json:"totalTokens"`
	StartedAt    time.Time  `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

func (SecurityAnalysisRun) TableName() string { return "security_analysis_runs" }

type SecurityKnownLoginSource struct {
	BaseModel
	SourceType  string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_security_known_login" json:"sourceType"`
	Username    string    `gorm:"type:varchar(128);not null;uniqueIndex:idx_security_known_login" json:"username"`
	IP          string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_security_known_login" json:"ip"`
	FirstSeenAt time.Time `json:"firstSeenAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

func (SecurityKnownLoginSource) TableName() string { return "security_known_login_sources" }
