package model

import "time"

// AIGroup 记录 Code 项目。保留原表名以兼容已有数据。
type AIGroup struct {
	ID               uint                       `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time                  `json:"createdAt"`
	UpdatedAt        time.Time                  `json:"updatedAt"`
	Name             string                     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description      string                     `gorm:"column:description;type:text" json:"description"`
	WorkDir          string                     `gorm:"column:work_dir;type:varchar(1024);not null;default:''" json:"workDir"`
	SourceDirs       []string                   `gorm:"column:source_dirs;serializer:json;type:text" json:"sourceDirs"`
	CreatorID        uint                       `gorm:"column:creator_id;type:integer;not null;index" json:"creatorId"`
	TaskCount        int64                      `gorm:"-" json:"taskCount"`
	ExecutionSummary *AIProjectExecutionSummary `gorm:"-" json:"executionSummary"`
}

type AIProjectExecutionSummary struct {
	Status               string     `json:"status"`
	ActiveTaskCount      int64      `json:"activeTaskCount"`
	PendingApprovalCount int64      `json:"pendingApprovalCount"`
	CurrentSessionID     uint       `json:"currentSessionId"`
	CurrentTaskID        uint       `json:"currentTaskId"`
	CurrentTaskTitle     string     `json:"currentTaskTitle"`
	CurrentStage         string     `json:"currentStage"`
	UpdatedAt            *time.Time `json:"updatedAt,omitempty"`
}

func (AIGroup) TableName() string {
	return "ai_groups"
}

// AITask 记录一次 AI 终端的会话/任务，允许用户后续根据 ID 恢复任务
type AITask struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	UserID          uint      `gorm:"column:user_id;type:integer;not null;index" json:"userId"`
	SessionID       uint      `gorm:"column:session_id;type:integer;index" json:"sessionId"`
	ProjectID       uint      `gorm:"column:project_id;type:integer;index;comment:所属项目/团队组ID，用于未来的团队共享记忆库" json:"projectId"`
	Title           string    `gorm:"column:title;type:varchar(255);not null" json:"title"`
	AgentName       string    `gorm:"column:agent_name;type:varchar(64)" json:"agentName"`
	NativeSessionID string    `gorm:"column:native_session_id;type:varchar(255)" json:"nativeSessionId"`
	WorkDir         string    `gorm:"column:work_dir;type:varchar(255);not null" json:"workDir"`
	Status          string    `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`
}

func (AITask) TableName() string {
	return "ai_tasks"
}

// AIMessage 记录 AI 任务中的具体对话内容
type AIMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	SessionID uint      `gorm:"column:session_id;type:integer;index" json:"sessionId"`
	TaskID    uint      `gorm:"column:task_id;type:integer;not null;index" json:"taskId"`
	RunID     uint      `gorm:"column:run_id;type:integer;index" json:"runId"`
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
	SourceWorkDir     string     `gorm:"column:source_work_dir;type:varchar(255)" json:"sourceWorkDir,omitempty"`
	WorktreeBranch    string     `gorm:"column:worktree_branch;type:varchar(255)" json:"worktreeBranch,omitempty"`
	Status            string     `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`
	CurrentStage      string     `gorm:"column:current_stage;type:varchar(64);default:'idle'" json:"currentStage"`
	LastTaskID        uint       `gorm:"column:last_task_id;type:integer;index" json:"lastTaskId"`
	NativeSessionID   string     `gorm:"column:native_session_id;type:varchar(255)" json:"nativeSessionId"`
	ProviderBaseURL   string     `gorm:"column:codex_base_url;type:varchar(1024)" json:"providerBaseUrl,omitempty"`
	ProviderModel     string     `gorm:"column:provider_model;type:varchar(255)" json:"providerModel,omitempty"`
	ProviderAPIKey    string     `gorm:"column:codex_api_key;type:text" json:"-"`
	ApprovalPolicy    string     `gorm:"column:approval_policy;type:varchar(32);not null;default:'safe_auto'" json:"approvalPolicy"`
	LastInstructionAt *time.Time `gorm:"column:last_instruction_at" json:"lastInstructionAt,omitempty"`
}

func (AIDevSession) TableName() string {
	return "ai_dev_sessions"
}

// AIExecutionRun 保存每轮执行的原始层数据，消息表仅承载适合界面展示的对话内容。
type AIExecutionRun struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	SessionID       uint       `gorm:"column:session_id;type:integer;not null;index" json:"sessionId"`
	TaskID          uint       `gorm:"column:task_id;type:integer;index" json:"taskId"`
	InstructionID   uint       `gorm:"column:instruction_id;type:integer;index" json:"instructionId"`
	ExecutorID      string     `gorm:"column:executor_id;type:varchar(64);not null;index" json:"executorId"`
	NativeSessionID string     `gorm:"column:native_session_id;type:varchar(255)" json:"nativeSessionId"`
	Prompt          string     `gorm:"column:prompt;type:text;not null" json:"prompt"`
	Output          string     `gorm:"column:output;type:text" json:"output"`
	RawOutput       string     `gorm:"column:raw_output;type:text" json:"rawOutput,omitempty"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	ExitCode        int        `gorm:"column:exit_code;default:0" json:"exitCode"`
	DurationMS      int64      `gorm:"column:duration_ms;default:0" json:"durationMs"`
	ErrorMessage    string     `gorm:"column:error_message;type:text" json:"errorMessage"`
	StartedAt       time.Time  `gorm:"column:started_at;not null" json:"startedAt"`
	CompletedAt     *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
}

func (AIExecutionRun) TableName() string {
	return "ai_execution_runs"
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

// AITimelineEvent 记录开发会话中的结构化过程事件，供手机端按时间线展示。
type AITimelineEvent struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	SessionID     uint      `gorm:"column:session_id;type:integer;not null;index" json:"sessionId"`
	TaskID        uint      `gorm:"column:task_id;type:integer;index" json:"taskId"`
	InstructionID uint      `gorm:"column:instruction_id;type:integer;index" json:"instructionId"`
	EventType     string    `gorm:"column:event_type;type:varchar(64);not null;index" json:"eventType"`
	Stage         string    `gorm:"column:stage;type:varchar(64)" json:"stage"`
	Title         string    `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Content       string    `gorm:"column:content;type:text" json:"content"`
	Status        string    `gorm:"column:status;type:varchar(32);default:'info'" json:"status"`
	Meta          string    `gorm:"column:meta;type:text" json:"meta"`
}

func (AITimelineEvent) TableName() string {
	return "ai_timeline_events"
}

// AIApproval 记录 AI 开发过程中的人工审批节点。
// 第一阶段仅覆盖“危险开发指令”的允许/拒绝，后续再补更多动作类型与审计细节。
type AIApproval struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	SessionID      uint       `gorm:"column:session_id;type:integer;not null;index" json:"sessionId"`
	TaskID         uint       `gorm:"column:task_id;type:integer;index" json:"taskId"`
	InstructionID  uint       `gorm:"column:instruction_id;type:integer;not null;index" json:"instructionId"`
	RequestUserID  uint       `gorm:"column:request_user_id;type:integer;not null;index" json:"requestUserId"`
	ApproveUserID  uint       `gorm:"column:approve_user_id;type:integer;index" json:"approveUserId"`
	Title          string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Content        string     `gorm:"column:content;type:text;not null" json:"content"`
	RiskLevel      string     `gorm:"column:risk_level;type:varchar(32);default:'medium'" json:"riskLevel"`
	Status         string     `gorm:"column:status;type:varchar(32);default:'pending'" json:"status"`
	Decision       string     `gorm:"column:decision;type:varchar(32)" json:"decision"`
	DecisionReason string     `gorm:"column:decision_reason;type:text" json:"decisionReason"`
	DecisionAt     *time.Time `gorm:"column:decision_at" json:"decisionAt,omitempty"`
}

func (AIApproval) TableName() string {
	return "ai_approvals"
}
