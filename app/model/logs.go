package model

import (
	"time"
)

type OperationLog struct {
	BaseModel
	Source    string `json:"source" gorm:"type:varchar(64)"`
	IP        string `json:"ip" gorm:"type:varchar(64)"`
	Path      string `json:"path" gorm:"type:varchar(255)"`
	Method    string `json:"method" gorm:"type:varchar(64)"`
	UserAgent string `json:"userAgent" gorm:"type:varchar(255)"`

	Latency time.Duration `json:"latency"`
	Status  string        `json:"status" gorm:"type:varchar(64)"`
	Message string        `json:"message" gorm:"type:text"`

	DetailZH string `json:"detailZH" gorm:"type:varchar(255)"`
	DetailEN string `json:"detailEN" gorm:"type:varchar(255)"`
}

type LoginLog struct {
	BaseModel
	IP      string `json:"ip" gorm:"type:varchar(64)"`
	Address string `json:"address" gorm:"type:varchar(255)"`
	Agent   string `json:"agent" gorm:"type:varchar(255)"`
	Status  string `json:"status" gorm:"type:varchar(64)"`
	Message string `json:"message" gorm:"type:text"`
}

type HostTerminalSession struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	UserID       uint       `gorm:"column:user_id;not null;index" json:"userId"`
	Status       string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	Shell        string     `gorm:"column:shell;type:varchar(64);not null" json:"shell"`
	WorkDir      string     `gorm:"column:work_dir;type:varchar(1024);not null" json:"workDir"`
	PID          int        `gorm:"column:pid;not null;default:0" json:"pid"`
	ExitCode     int        `gorm:"column:exit_code;not null;default:0" json:"exitCode"`
	ClientIP     string     `gorm:"column:client_ip;type:varchar(64)" json:"clientIp"`
	OutputBytes  int64      `gorm:"column:output_bytes;not null;default:0" json:"outputBytes"`
	ErrorMessage string     `gorm:"column:error_message;type:varchar(500)" json:"errorMessage,omitempty"`
	StartedAt    time.Time  `gorm:"column:started_at;not null" json:"startedAt"`
	EndedAt      *time.Time `gorm:"column:ended_at" json:"endedAt,omitempty"`
}

func (HostTerminalSession) TableName() string { return "host_terminal_sessions" }

type HostTerminalAuditEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_host_terminal_audit_session_created,priority:2" json:"createdAt"`
	SessionID uint      `gorm:"column:session_id;not null;index;index:idx_host_terminal_audit_session_created,priority:1" json:"sessionId"`
	UserID    uint      `gorm:"column:user_id;not null;index" json:"userId"`
	Action    string    `gorm:"column:action;type:varchar(64);not null;index" json:"action"`
	Status    string    `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	IP        string    `gorm:"column:ip;type:varchar(64)" json:"ip"`
	Detail    string    `gorm:"column:detail;type:varchar(500)" json:"detail"`
}

func (HostTerminalAuditEvent) TableName() string { return "host_terminal_audit_events" }
