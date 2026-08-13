package model

import "time"

// AIProviderAccount 是面板级的 AI 账号。
//
// 执行器（codex / claude CLI）用的是它们自己的登录态，面板拿不到；
// 而记忆抽取、以及以后任何需要面板自己调模型的功能，都需要一份可用的凭据。
// 集中成账号池而不是每处各配一份：现在建会话要手填 baseURL/key/model，
// 结果是 59 个会话一个都没填——「每次手填」这个设计基本注定没人会用。
type AIProviderAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    uint      `gorm:"column:user_id;not null;index" json:"userId"`
	Name      string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	BaseURL   string    `gorm:"column:base_url;type:varchar(1024);not null" json:"baseUrl"`
	APIKey    string    `gorm:"column:api_key;type:text" json:"-"`
	Model     string    `gorm:"column:model;type:varchar(255);not null" json:"model"`
	Enabled   bool      `gorm:"column:enabled;not null" json:"enabled"`
	// UseForMemoryExtraction 必须单独授权。抽取会把整段会话记录发出去，
	// 用户可能同时配着公司内网、自费和借来的额度——默认拿任意一个去发全部
	// 对话内容，不该由系统替他决定。
	UseForMemoryExtraction bool `gorm:"column:use_for_memory_extraction;not null;default:false" json:"useForMemoryExtraction"`
	// UseForSecurityAnalysis 单独授权把脱敏后的安全证据发送给模型。
	UseForSecurityAnalysis bool `gorm:"column:use_for_security_analysis;not null;default:false" json:"useForSecurityAnalysis"`
	// Priority 越小越优先，用于「自动」挑选。
	Priority int `gorm:"column:priority;not null;default:100" json:"priority"`
	// DefaultReasoningEffort 是用户给这个账号定的推理强度基线。
	// 只在探测确认模型支持时才会真的发出去。
	DefaultReasoningEffort string `gorm:"column:default_reasoning_effort;type:varchar(16);not null;default:''" json:"defaultReasoningEffort"`

	// 以下三个是保存时探测出来的事实，不是用户填的偏好。
	//
	// 注意不能给这些布尔加 gorm 的 default:true：Go 的 false 是零值，
	// GORM 创建时会把零值字段整个省掉，让数据库默认值生效——
	// 探测出「不支持 temperature」反而会被存成「支持」，之后每次抽取都 400。
	//
	// 存事实而不是让用户勾选：用户不该需要知道自己选的模型支不支持
	// temperature——填错的代价是后台调用静默 400，而抽取失败只记日志，
	// 用户只会看到记忆永远不出现。
	SupportsTemperature     bool       `gorm:"column:supports_temperature;not null" json:"supportsTemperature"`
	SupportsJSONSchema      bool       `gorm:"column:supports_json_schema;not null" json:"supportsJsonSchema"`
	SupportsReasoningEffort bool       `gorm:"column:supports_reasoning_effort;not null;default:false" json:"supportsReasoningEffort"`
	ProbedAt                *time.Time `gorm:"column:probed_at" json:"probedAt,omitempty"`
	ProbeError              string     `gorm:"column:probe_error;type:text" json:"probeError,omitempty"`
}

func (AIProviderAccount) TableName() string { return "ai_provider_accounts" }
