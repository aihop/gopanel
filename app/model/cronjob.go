package model

import (
	"time"
)

// Cronjob 计划任务
type Cronjob struct {
	BaseModel

	Name   string `gorm:"type:varchar(64);not null" json:"name"`
	Type   string `gorm:"type:varchar(32);not null" json:"type"` // shell | db_backup | log_clean | ssl_renew
	Spec   string `gorm:"type:varchar(64);not null" json:"spec"` // 5 段 cron 表达式
	Status string `gorm:"type:varchar(16);not null" json:"status"`
	// robfig/cron 返回的 EntryID，未调度（如禁用状态）时为 0
	EntryID int `gorm:"type:int;not null;default:0" json:"entryID"`

	// shell 类型
	Script string `gorm:"type:longtext" json:"script"`

	// db_backup 类型
	ServerID     uint   `gorm:"type:int unsigned;not null;default:0" json:"serverID"`
	DBType       string `gorm:"type:varchar(32)" json:"dbType"`
	DBName       string `gorm:"type:varchar(128)" json:"dbName"`
	RetainCopies int    `gorm:"type:int;not null;default:0" json:"retainCopies"`

	// log_clean 类型
	LogType string `gorm:"type:varchar(32)" json:"logType"` // operation | login | all

	// 最近一次执行记录，仅列表接口回填展示用，不持久化
	LastRecord *JobRecords `gorm:"-" json:"lastRecord,omitempty"`
}

// JobRecords 计划任务执行记录
type JobRecords struct {
	BaseModel

	CronjobID uint      `gorm:"type:int unsigned;not null;index" json:"cronjobID"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    string    `gorm:"type:varchar(16);not null" json:"status"`
	Message   string    `gorm:"type:longtext" json:"message"`
}
