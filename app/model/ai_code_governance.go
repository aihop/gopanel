package model

import "time"

type AICodeDelivery struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	SessionID       uint       `gorm:"column:session_id;not null;uniqueIndex" json:"sessionId"`
	ProjectID       uint       `gorm:"column:project_id;not null;index" json:"projectId"`
	UserID          uint       `gorm:"column:user_id;not null;index" json:"userId"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	SourceWorkDir   string     `gorm:"column:source_work_dir;type:varchar(1024);not null" json:"sourceWorkDir"`
	WorkDir         string     `gorm:"column:work_dir;type:varchar(1024);not null" json:"workDir"`
	DeliveryWorkDir string     `gorm:"column:delivery_work_dir;type:varchar(1024)" json:"deliveryWorkDir,omitempty"`
	WorktreeBranch  string     `gorm:"column:worktree_branch;type:varchar(255);not null" json:"worktreeBranch"`
	TargetBranch    string     `gorm:"column:target_branch;type:varchar(255)" json:"targetBranch,omitempty"`
	BaseCommit      string     `gorm:"column:base_commit;type:varchar(64)" json:"baseCommit,omitempty"`
	RemoteName      string     `gorm:"column:remote_name;type:varchar(255)" json:"remoteName,omitempty"`
	RemoteBranch    string     `gorm:"column:remote_branch;type:varchar(255)" json:"remoteBranch,omitempty"`
	RemoteCommit    string     `gorm:"column:remote_commit;type:varchar(64)" json:"remoteCommit,omitempty"`
	SourceCommit    string     `gorm:"column:source_commit;type:varchar(64)" json:"sourceCommit,omitempty"`
	WorktreeCommit  string     `gorm:"column:worktree_commit;type:varchar(64)" json:"worktreeCommit"`
	MergeCommit     string     `gorm:"column:merge_commit;type:varchar(64)" json:"mergeCommit"`
	ErrorMessage    string     `gorm:"column:error_message;type:text" json:"errorMessage"`
	MergedAt        *time.Time `gorm:"column:merged_at" json:"mergedAt,omitempty"`
	CompletedAt     *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
	PushStatus      string     `gorm:"column:push_status;type:varchar(32);not null;default:'pending';index" json:"pushStatus"`
	PushedCommit    string     `gorm:"column:pushed_commit;type:varchar(64)" json:"pushedCommit,omitempty"`
	PushError       string     `gorm:"column:push_error;type:text" json:"pushError,omitempty"`
	PushedAt        *time.Time `gorm:"column:pushed_at" json:"pushedAt,omitempty"`
}

func (AICodeDelivery) TableName() string { return "ai_code_deliveries" }

type AICodeAuditEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `gorm:"index:idx_code_audit_session_created,priority:2" json:"createdAt"`
	UserID     uint      `gorm:"column:user_id;not null;index" json:"userId"`
	ProjectID  uint      `gorm:"column:project_id;not null;index" json:"projectId"`
	SessionID  uint      `gorm:"column:session_id;not null;index;index:idx_code_audit_session_created,priority:1" json:"sessionId"`
	Action     string    `gorm:"column:action;type:varchar(64);not null;index" json:"action"`
	Status     string    `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	Resource   string    `gorm:"column:resource;type:varchar(255)" json:"resource"`
	Detail     string    `gorm:"column:detail;type:varchar(500)" json:"detail"`
	IP         string    `gorm:"column:ip;type:varchar(64)" json:"ip"`
	DurationMS int64     `gorm:"column:duration_ms;not null;default:0" json:"durationMs"`
	Meta       string    `gorm:"column:meta;type:text" json:"meta"`
}

func (AICodeAuditEvent) TableName() string { return "ai_code_audit_events" }
