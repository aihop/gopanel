package model

import "time"

// AICodeMemoryEntry 是一条长期记忆。
//
// 会话之间原本完全隔离：AI 上一次踩过的坑、摸清的项目约定、用户纠正过的
// 偏好，下一个会话全部从零开始。这张表把这些沉淀下来，在下一次执行时回灌。
//
// 分三个维度：
//   - Scope   user 跨项目生效（ProjectID 为 0），project 只在本项目生效
//   - Kind    preference / decision / fact / bug_lesson，四类穷尽且互斥
//   - Tier    core / working / archive，由 Kind 自动推导，不由模型决定
type AICodeMemoryEntry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_ai_code_memory_scope,priority:4" json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    uint      `gorm:"column:user_id;not null;index;index:idx_ai_code_memory_scope,priority:1" json:"userId"`
	// scope=user 的记忆跨项目生效，这里存 0。
	ProjectID uint   `gorm:"column:project_id;not null;index:idx_ai_code_memory_scope,priority:2" json:"projectId"`
	Scope     string `gorm:"column:scope;type:varchar(16);not null;index:idx_ai_code_memory_scope,priority:3" json:"scope"`
	Kind      string `gorm:"column:kind;type:varchar(32);not null;index" json:"kind"`
	Tier      string `gorm:"column:tier;type:varchar(16);not null;index" json:"tier"`
	// ModuleKey 是记忆归属的模块名（frontend / git / delivery …），
	// 用来在注入时按模块聚合，避免一次灌进去几十条零散事实。
	ModuleKey string `gorm:"column:module_key;type:varchar(64);not null" json:"moduleKey"`
	Content   string `gorm:"column:content;type:text;not null" json:"content"`
	Rationale string `gorm:"column:rationale;type:text" json:"rationale,omitempty"`
	Status    string `gorm:"column:status;type:varchar(16);not null;index" json:"status"`
	// 记忆是从哪次会话抽出来的，用于回溯「这条是怎么来的」。
	SourceSessionID uint `gorm:"column:source_session_id;index" json:"sourceSessionId,omitempty"`
	// 被合并/取代时指向接手的那条，保留链路而不是直接删。
	SupersededBy uint       `gorm:"column:superseded_by" json:"supersededBy,omitempty"`
	ArchivedAt   *time.Time `gorm:"column:archived_at" json:"archivedAt,omitempty"`
}

func (AICodeMemoryEntry) TableName() string { return "ai_code_memory_entries" }

// AICodeMemorySummary 是用户级的长期画像，每个用户一条。
//
// 与逐条记忆分开：条目回答「有哪些具体事实」，摘要回答「这个人怎么干活」。
// 后者每次抽取由模型整体重写，不做增量拼接——拼接会越滚越长。
type AICodeMemorySummary struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    uint      `gorm:"column:user_id;not null;uniqueIndex" json:"userId"`
	Content   string    `gorm:"column:content;type:text" json:"content"`
}

func (AICodeMemorySummary) TableName() string { return "ai_code_memory_summaries" }
