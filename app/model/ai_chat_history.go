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

// AIDevSession 记录面向手机/Web 的长期开发会话。
// 它是 AITask 之上的控制平面对象，用来承载会话、指令和状态摘要。
type AIDevSession struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	UserID            uint       `gorm:"column:user_id;type:integer;not null;index" json:"userId"`
	ProjectID         uint       `gorm:"column:project_id;type:integer;index" json:"projectId"`
	Title             string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	AgentName         string     `gorm:"column:agent_name;type:varchar(64)" json:"agentName"`
	WorkDir           string     `gorm:"column:work_dir;type:varchar(255);not null" json:"workDir"`
	Status            string     `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`
	CurrentStage      string     `gorm:"column:current_stage;type:varchar(64);default:'idle'" json:"currentStage"`
	LastTaskID        uint       `gorm:"column:last_task_id;type:integer;index" json:"lastTaskId"`
	LastInstructionAt *time.Time `gorm:"column:last_instruction_at" json:"lastInstructionAt,omitempty"`
}

func (AIDevSession) TableName() string {
	return "ai_dev_sessions"
}

// AIInstruction 记录会话中的一条开发指令。
// 第一阶段先完成“落库 + 绑定任务 + 手机可查”，后续再补审批与预览链路。
type AIInstruction struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	SessionID       uint      `gorm:"column:session_id;type:integer;not null;index" json:"sessionId"`
	UserID          uint      `gorm:"column:user_id;type:integer;not null;index" json:"userId"`
	ProjectID       uint      `gorm:"column:project_id;type:integer;index" json:"projectId"`
	TaskID          uint      `gorm:"column:task_id;type:integer;index" json:"taskId"`
	Content         string    `gorm:"column:content;type:text;not null" json:"content"`
	Status          string    `gorm:"column:status;type:varchar(32);default:'queued'" json:"status"`
	AllowCode       bool      `gorm:"column:allow_code;default:true" json:"allowCode"`
	AutoPreview     bool      `gorm:"column:auto_preview;default:false" json:"autoPreview"`
	RequireApproval bool      `gorm:"column:require_approval;default:true" json:"requireApproval"`
	AnalysisOnly    bool      `gorm:"column:analysis_only;default:false" json:"analysisOnly"`
}

func (AIInstruction) TableName() string {
	return "ai_instructions"
}

// AIPreview 记录开发过程里提取或登记出来的预览对象。
// 第一阶段先覆盖 Web 预览 URL 的回传与展示，后续再补探测、截图和更多来源类型。
type AIPreview struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	SessionID     uint       `gorm:"column:session_id;type:integer;not null;index" json:"sessionId"`
	TaskID        uint       `gorm:"column:task_id;type:integer;index" json:"taskId"`
	InstructionID uint       `gorm:"column:instruction_id;type:integer;index" json:"instructionId"`
	PreviewType   string     `gorm:"column:preview_type;type:varchar(32);default:'web'" json:"previewType"`
	Source        string     `gorm:"column:source;type:varchar(32);default:'agent_output'" json:"source"`
	Title         string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	URL           string     `gorm:"column:url;type:text;not null" json:"url"`
	Status        string     `gorm:"column:status;type:varchar(32);default:'ready'" json:"status"`
	LastCheckedAt *time.Time `gorm:"column:last_checked_at" json:"lastCheckedAt,omitempty"`
}

func (AIPreview) TableName() string {
	return "ai_previews"
}
