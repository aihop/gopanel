package model

import "time"

type AICodeDeliveryJob struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time  `gorm:"index:idx_ai_code_delivery_jobs_queue,priority:3" json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	SessionID         uint       `gorm:"column:session_id;not null;uniqueIndex" json:"sessionId"`
	TaskID            uint       `gorm:"column:task_id;index" json:"taskId,omitempty"`
	ProjectID         uint       `gorm:"column:project_id;not null;index" json:"projectId"`
	UserID            uint       `gorm:"column:user_id;not null;index" json:"userId"`
	Status            string     `gorm:"column:status;type:varchar(32);not null;index:idx_ai_code_delivery_jobs_queue,priority:1" json:"status"`
	Stage             string     `gorm:"column:stage;type:varchar(32);not null" json:"stage"`
	Progress          int        `gorm:"column:progress;not null;default:0" json:"progress"`
	Attempt           int        `gorm:"column:attempt;not null;default:0" json:"attempt"`
	RepositoryKeys    string     `gorm:"column:repository_keys;type:text;not null" json:"-"`
	TargetBranch      string     `gorm:"column:target_branch;type:varchar(255)" json:"targetBranch,omitempty"`
	ResultCommit      string     `gorm:"column:result_commit;type:varchar(64)" json:"resultCommit,omitempty"`
	ResultType        string     `gorm:"column:result_type;type:varchar(32)" json:"resultType,omitempty"`
	FailureCode       string     `gorm:"column:failure_code;type:varchar(32);index" json:"failureCode,omitempty"`
	RepositoryResults string     `gorm:"column:repository_results;type:text" json:"-"`
	ErrorMessage      string     `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	ConflictFiles     string     `gorm:"column:conflict_files;type:text" json:"-"`
	RequestIP         string     `gorm:"column:request_ip;type:varchar(64)" json:"-"`
	LeaseOwner        string     `gorm:"column:lease_owner;type:varchar(128);index" json:"-"`
	LeaseExpiresAt    *time.Time `gorm:"column:lease_expires_at;index:idx_ai_code_delivery_jobs_queue,priority:2" json:"-"`
	StartedAt         *time.Time `gorm:"column:started_at" json:"startedAt,omitempty"`
	CompletedAt       *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
}

func (AICodeDeliveryJob) TableName() string { return "ai_code_delivery_jobs" }

// AICodeDeliveryAttempt 逐次留档交付结果。
//
// AICodeDeliveryJob 的 session_id 是唯一索引，重新交付会直接覆盖上一次的记录，
// 失败因此在面板里完全不可见——一个实测三次里失败两次的流程，作业表里却只剩
// 一行 completed。这张表只追加不更新，用来回答「这个会话交付过几次、都卡在哪」。
type AICodeDeliveryAttempt struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `gorm:"index:idx_ai_code_delivery_attempts_session,priority:2" json:"createdAt"`
	SessionID    uint      `gorm:"column:session_id;not null;index:idx_ai_code_delivery_attempts_session,priority:1" json:"sessionId"`
	JobID        uint      `gorm:"column:job_id;not null;index" json:"jobId"`
	TaskID       uint      `gorm:"column:task_id;index" json:"taskId,omitempty"`
	ProjectID    uint      `gorm:"column:project_id;not null;index" json:"projectId"`
	UserID       uint      `gorm:"column:user_id;not null;index" json:"userId"`
	Attempt      int       `gorm:"column:attempt;not null;default:0" json:"attempt"`
	Status       string    `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	Stage        string    `gorm:"column:stage;type:varchar(32);not null" json:"stage"`
	FailureCode  string    `gorm:"column:failure_code;type:varchar(32);index" json:"failureCode,omitempty"`
	ResultCommit string    `gorm:"column:result_commit;type:varchar(64)" json:"resultCommit,omitempty"`
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	DurationMS   int64     `gorm:"column:duration_ms;not null;default:0" json:"durationMs"`
}

func (AICodeDeliveryAttempt) TableName() string { return "ai_code_delivery_attempts" }

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
