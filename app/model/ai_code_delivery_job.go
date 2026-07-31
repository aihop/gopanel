package model

import "time"

type AICodeDeliveryJob struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time  `gorm:"index:idx_ai_code_delivery_jobs_queue,priority:3" json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	SessionID      uint       `gorm:"column:session_id;not null;uniqueIndex" json:"sessionId"`
	ProjectID      uint       `gorm:"column:project_id;not null;index" json:"projectId"`
	UserID         uint       `gorm:"column:user_id;not null;index" json:"userId"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;index:idx_ai_code_delivery_jobs_queue,priority:1" json:"status"`
	Stage          string     `gorm:"column:stage;type:varchar(32);not null" json:"stage"`
	Progress       int        `gorm:"column:progress;not null;default:0" json:"progress"`
	Attempt        int        `gorm:"column:attempt;not null;default:0" json:"attempt"`
	RepositoryKeys string     `gorm:"column:repository_keys;type:text;not null" json:"-"`
	TargetBranch   string     `gorm:"column:target_branch;type:varchar(255)" json:"targetBranch,omitempty"`
	ResultCommit   string     `gorm:"column:result_commit;type:varchar(64)" json:"resultCommit,omitempty"`
	ErrorMessage   string     `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	ConflictFiles  string     `gorm:"column:conflict_files;type:text" json:"-"`
	RequestIP      string     `gorm:"column:request_ip;type:varchar(64)" json:"-"`
	LeaseOwner     string     `gorm:"column:lease_owner;type:varchar(128);index" json:"-"`
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;index:idx_ai_code_delivery_jobs_queue,priority:2" json:"-"`
	StartedAt      *time.Time `gorm:"column:started_at" json:"startedAt,omitempty"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
}

func (AICodeDeliveryJob) TableName() string { return "ai_code_delivery_jobs" }

type AICodeDeliveryLease struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	RepositoryKey  string     `gorm:"column:repository_key;type:varchar(64);not null;uniqueIndex" json:"repositoryKey"`
	JobID          uint       `gorm:"column:job_id;not null;index" json:"jobId"`
	LeaseOwner     string     `gorm:"column:lease_owner;type:varchar(128);not null;index" json:"-"`
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;index" json:"-"`
}

func (AICodeDeliveryLease) TableName() string { return "ai_code_delivery_leases" }
