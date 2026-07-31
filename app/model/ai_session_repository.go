package model

import "time"

type AIDevSessionRepository struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	SessionID      uint       `gorm:"column:session_id;not null;uniqueIndex:idx_ai_session_repository" json:"sessionId"`
	ProjectID      uint       `gorm:"column:project_id;not null;index" json:"projectId"`
	SourceDir      string     `gorm:"column:source_dir;type:varchar(1024);not null" json:"sourceDir"`
	WorktreeDir    string     `gorm:"column:worktree_dir;type:varchar(1024);not null" json:"worktreeDir"`
	LinkName       string     `gorm:"column:link_name;type:varchar(255);not null;uniqueIndex:idx_ai_session_repository" json:"linkName"`
	Branch         string     `gorm:"column:branch;type:varchar(255);not null" json:"branch"`
	BaseCommit     string     `gorm:"column:base_commit;type:varchar(64);not null" json:"baseCommit"`
	WorktreeCommit string     `gorm:"column:worktree_commit;type:varchar(64)" json:"worktreeCommit,omitempty"`
	MergeCommit    string     `gorm:"column:merge_commit;type:varchar(64)" json:"mergeCommit,omitempty"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	ErrorMessage   string     `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	MergedAt       *time.Time `gorm:"column:merged_at" json:"mergedAt,omitempty"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
}

func (AIDevSessionRepository) TableName() string { return "ai_dev_session_repositories" }
