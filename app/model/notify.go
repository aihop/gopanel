package model

import "time"

// TLS 连接方式。465 端口用的是隐式 TLS（连上就握手），587/25 是 STARTTLS（先明文再升级），
// 两者不能混用——填错的表现是连接卡住直到超时，这是配 SMTP 最常见的坑。
const (
	SMTPTLSNone     = "none"     // 不加密，仅限内网自建中继
	SMTPTLSStartTLS = "starttls" // 25 / 587
	SMTPTLSSSL      = "ssl"      // 465，隐式 TLS
)

// NotifyConfig 通知配置，全局单行（ID 固定为 1）。
// 单独建表而不是塞进 setting 表：setting.value 是 varchar(256)，
// 收件人列表稍多就会溢出。
//
// 注意：这里的字段一律不加 gorm `default:` 标签。GORM 在插入时会跳过
// 「零值 + 有 default 标签」的字段，让数据库去填默认值——后果是用户把
// 「磁盘告警」关掉（false）、把静默期设成 0（只发一次），保存后又变回
// 开启和 6 小时，而且毫无提示。默认值统一在 repo.GetConfig 里给。
type NotifyConfig struct {
	BaseModel
	Enabled bool `json:"enabled"`

	SMTPHost string `gorm:"type:varchar(255)" json:"smtpHost"`
	SMTPPort int    `json:"smtpPort"`
	SMTPUser string `gorm:"type:varchar(255)" json:"smtpUser"`
	// SMTPPassword AES 加密后存储，接口一律不回明文（与节点令牌同样处理）
	SMTPPassword string `gorm:"type:varchar(512)" json:"-"`
	SMTPFrom     string `gorm:"type:varchar(255)" json:"smtpFrom"`
	SMTPTLSMode  string `gorm:"type:varchar(16)" json:"smtpTlsMode"`
	// Receivers 收件人，逗号分隔
	Receivers string `gorm:"type:text" json:"receivers"`

	// DebounceTimes 连续命中多少次才真正触发。采集每分钟一轮，
	// 节点网络闪断会瞬间变 offline，不去抖就会误报。
	DebounceTimes int `json:"debounceTimes"`
	// SilenceHours 同一事件持续未恢复时，隔多久再提醒一次。
	// 没有这个的话磁盘满会每分钟发一封，用户第二天就把通知关了。
	SilenceHours int `json:"silenceHours"`
	// NotifyResolved 恢复时是否也发一封。只报警不报恢复，用户会一直提心吊胆。
	NotifyResolved bool `json:"notifyResolved"`

	// 事件类型开关
	EnableDisk      bool `json:"enableDisk"`
	EnableContainer bool `json:"enableContainer"`
	EnableOffline   bool `json:"enableOffline"`
	EnableCert      bool `json:"enableCert"`
}

func (NotifyConfig) TableName() string {
	return "notify_configs"
}

// 告警事件状态
const (
	AlertStatusPending  = "pending"  // 命中了但还没达到去抖次数
	AlertStatusFiring   = "firing"   // 已触发并已通知
	AlertStatusResolved = "resolved" // 已恢复
)

// 告警来源
const (
	AlertSourceLocal = "local"
	AlertSourceNode  = "node"
)

// AlertEvent 告警事件。
//
// 必须落库而不是放内存：状态机（去抖计数、静默期、是否已通知）一旦随进程丢失，
// 面板重启后会把所有仍在持续的告警重新发一遍。
//
// 唯一性由 (SourceType, NodeID, Type) 决定：同一个节点的同一类告警只保留一条活动记录，
// 恢复后再次触发则复用该行重新计数。
type AlertEvent struct {
	BaseModel
	SourceType string  `gorm:"type:varchar(16);not null;index" json:"sourceType"`
	NodeID     uint    `gorm:"index" json:"nodeId"` // 本机为 0
	SourceName string  `gorm:"type:varchar(128)" json:"sourceName"`
	Type       string  `gorm:"type:varchar(32);not null;index" json:"type"` // disk / container / offline / cert / unauthorized
	Level      string  `gorm:"type:varchar(16)" json:"level"`               // warn / danger
	Status     string  `gorm:"type:varchar(16);not null;index" json:"status"`
	Value      float64 `json:"value"`
	Detail     string  `gorm:"type:varchar(512)" json:"detail"`

	HitCount       int       `gorm:"default:0" json:"hitCount"` // 连续命中次数，用于去抖
	FirstSeenAt    time.Time `json:"firstSeenAt"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
	LastNotifiedAt time.Time `gorm:"default:NULL" json:"lastNotifiedAt"`
	ResolvedAt     time.Time `gorm:"default:NULL" json:"resolvedAt"`
}

func (AlertEvent) TableName() string {
	return "alert_events"
}
