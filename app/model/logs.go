package model

import (
	"time"
)

type OperationLog struct {
	BaseModel
	Source    string `json:"source" gorm:"type:varchar(64)"`
	IP        string `json:"ip" gorm:"type:varchar(64)"`
	Path      string `json:"path" gorm:"type:varchar(255)"`
	Method    string `json:"method" gorm:"type:varchar(64)"`
	UserAgent string `json:"userAgent" gorm:"type:varchar(255)"`

	Latency time.Duration `json:"latency"`
	Status  string        `json:"status" gorm:"type:varchar(64)"`
	Message string        `json:"message" gorm:"type:text"`

	DetailZH string `json:"detailZH" gorm:"type:varchar(255)"`
	DetailEN string `json:"detailEN" gorm:"type:varchar(255)"`
}

type LoginLog struct {
	BaseModel
	IP      string `json:"ip" gorm:"type:varchar(64)"`
	Address string `json:"address" gorm:"type:varchar(255)"`
	Agent   string `json:"agent" gorm:"type:varchar(255)"`
	Status  string `json:"status" gorm:"type:varchar(64)"`
	Message string `json:"message" gorm:"type:text"`
}