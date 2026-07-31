package model

import "time"

type AIDevSessionRepository struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	SessionID       uint       `gorm:"column:session_id;not null;uniqueIndex:idx_ai_session_repository" json:"sessionId"`
	ProjectID       uint       `gorm:"column:project_id;not null;index" json:"projectId"`
	SourceDir       string     `gorm:"column:source_dir;type:varchar(1024);not null" json:"sourceDir"`
	ParentSourceDir string     `gorm:"column:parent_source_dir;type:varchar(1024)" json:"parentSourceDir,omitempty"`
	GitlinkPath     string     `gorm:"column:gitlink_path;type:varchar(1024)" json:"gitlinkPath,omitempty"`
	WorktreeDir     string     `gorm:"column:worktree_dir;type:varchar(1024);not null" json:"worktreeDir"`
	LinkName        string     `gorm:"column:link_name;type:varchar(255);not null;uniqueIndex:idx_ai_session_repository" json:"linkName"`
	Branch          string     `gorm:"column:branch;type:varchar(255);not null" json:"branch"`
	TargetBranch    string     `gorm:"column:target_branch;type:varchar(255);not null;default:''" json:"targetBranch"`
	BaseCommit      string     `gorm:"column:base_commit;type:varchar(64);not null" json:"baseCommit"`
	RemoteName      string     `gorm:"column:remote_name;type:varchar(255)" json:"remoteName,omitempty"`
	RemoteBranch    string     `gorm:"column:remote_branch;type:varchar(255)" json:"remoteBranch,omitempty"`
	RemoteCommit    string     `gorm:"column:remote_commit;type:varchar(64)" json:"remoteCommit,omitempty"`
	SyncStatus      string     `gorm:"column:sync_status;type:varchar(32);not null;default:'local'" json:"syncStatus"`
	Snapshot        bool       `gorm:"column:snapshot;not null;default:false" json:"snapshot"`
	WorktreeCommit  string     `gorm:"column:worktree_commit;type:varchar(64)" json:"worktreeCommit,omitempty"`
	MergeCommit     string     `gorm:"column:merge_commit;type:varchar(64)" json:"mergeCommit,omitempty"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	ErrorMessage    string     `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	MergedAt        *time.Time `gorm:"column:merged_at" json:"mergedAt,omitempty"`
	CompletedAt     *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
	PushStatus      string     `gorm:"column:push_status;type:varchar(32);not null;default:'pending';index" json:"pushStatus"`
	PushedCommit    string     `gorm:"column:pushed_commit;type:varchar(64)" json:"pushedCommit,omitempty"`
	PushError       string     `gorm:"column:push_error;type:text" json:"pushError,omitempty"`
	PushedAt        *time.Time `gorm:"column:pushed_at" json:"pushedAt,omitempty"`
}

func (AIDevSessionRepository) TableName() string { return "ai_dev_session_repositories" }
