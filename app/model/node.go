package model

import "time"

// Node 受管节点。
// 当前阶段（观测面）只做只读：主控定时拉取节点摘要用于集中展示，不代理任何写操作。
// ConnectMode 预留 direct / tunnel 两种取值，当前只实现 direct。
type Node struct {
	BaseModel
	Name        string `json:"name" gorm:"type:varchar(128);not null"`
	Addr        string `json:"addr" gorm:"type:varchar(255);not null"`        // 面板地址，含协议与端口，如 https://1.2.3.4:5470
	Entrance    string `json:"entrance" gorm:"type:varchar(255);default:''"` // 节点安全入口，节点未开启则留空
	AccessToken string `json:"-" gorm:"type:varchar(512)"`                   // 节点只读令牌，AES 加密后存储，不出接口
	ConnectMode string `json:"connectMode" gorm:"type:varchar(32);default:'direct'"`
	SkipVerify  bool   `json:"skipVerify" gorm:"default:false"` // 节点使用自签证书时跳过 TLS 校验
	IsProd      bool   `json:"isProd" gorm:"default:false"`     // 生产节点，前端高亮标记
	Sort        int    `json:"sort" gorm:"default:0"`

	// 以下字段由摘要采集任务写入，代表最近一次采集结果
	Status     string      `json:"status" gorm:"type:varchar(32);default:'unknown'"`
	StatusMsg  string      `json:"statusMsg" gorm:"type:varchar(255);default:''"`
	Version    string      `json:"version" gorm:"type:varchar(64);default:''"`
	LastSeenAt time.Time   `json:"lastSeenAt" gorm:"default:NULL"`
	Summary    NodeSummary `json:"summary" gorm:"serializer:json;type:json"`
}

// NodeSummary 节点摘要快照。
// 只保存最近一次结果（随 Node 行一起存），不做时间序列——历史曲线由各节点自己的 monitor.db 负责，
// 主控只回答“现在哪台有问题”。
type NodeSummary struct {
	Hostname   string  `json:"hostname"`
	OS         string  `json:"os"`
	Uptime     uint64  `json:"uptime"`
	CPUPercent float64 `json:"cpuPercent"`
	CPUTotal   int     `json:"cpuTotal"`
	Load1      float64 `json:"load1"`

	MemPercent float64 `json:"memPercent"`
	MemTotal   uint64  `json:"memTotal"`
	MemUsed    uint64  `json:"memUsed"`

	DiskMaxPercent float64 `json:"diskMaxPercent"`
	DiskMaxPath    string  `json:"diskMaxPath"`

	ContainerRunning  int `json:"containerRunning"`
	ContainerTotal    int `json:"containerTotal"`
	ContainerAbnormal int `json:"containerAbnormal"`

	CertTotal         int `json:"certTotal"`         // 参与统计的证书数量，为 0 时下面两个字段无意义
	CertExpiringCount int `json:"certExpiringCount"` // 30 天内到期（含已过期）的证书数量
	CertMinDays       int `json:"certMinDays"`       // 最近一张证书的剩余天数，负数表示已过期

	Version  string    `json:"version"`
	ShotTime time.Time `json:"shotTime"`
}
