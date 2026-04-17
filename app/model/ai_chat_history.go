package model

import "time"

// AIGroup 记录团队项目组，实现 GoPanel 的团队级 AI 协作与共享记忆
type AIGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Name        string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	CreatorID   uint      `gorm:"column:creator_id;type:integer;not null;index" json:"creatorId"`
	// 以下字段可通过统计任务数和组成员表获取，为简单起见，目前通过 SQL 连表查询返回
}

func (AIGroup) TableName() string {
	return "ai_groups"
}

// AITask 记录一次 AI 终端的会话/任务，允许用户后续根据 ID 恢复任务
type AITask struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    uint      `gorm:"column:user_id;type:integer;not null;index" json:"userId"`
	ProjectID uint      `gorm:"column:project_id;type:integer;index;comment:所属项目/团队组ID，用于未来的团队共享记忆库" json:"projectId"`
	Title     string    `gorm:"column:title;type:varchar(255);not null" json:"title"`
	AgentName string    `gorm:"column:agent_name;type:varchar(64)" json:"agentName"`
	WorkDir   string    `gorm:"column:work_dir;type:varchar(255);not null" json:"workDir"`
	Status    string    `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`
}

func (AITask) TableName() string {
	return "ai_tasks"
}

// AIMessage 记录 AI 任务中的具体对话内容
type AIMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	TaskID    uint      `gorm:"column:task_id;type:integer;not null;index" json:"taskId"`
	Role      string    `gorm:"column:role;type:varchar(32);not null" json:"role"` // user / agent
	Content   string    `gorm:"column:content;type:text;not null" json:"content"`
}

func (AIMessage) TableName() string {
	return "ai_messages"
}
