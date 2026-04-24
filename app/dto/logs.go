package dto

import (
	"time"
)

type OperationLog struct {
	ID     uint   `json:"id"`
	Source string `json:"source"`

	IP        string `json:"ip"`
	Path      string `json:"path"`
	Method    string `json:"method"`
	UserAgent string `json:"userAgent"`

	Latency time.Duration `json:"latency"`
	Status  string        `json:"status"`
	Message string        `json:"message"`

	DetailZH  string    `json:"detailZH"`
	DetailEN  string    `json:"detailEN"`
	CreatedAt time.Time `json:"createdAt"`
}

type SearchOpLogWithPage struct {
	PageInfo
	Source    string `json:"source"`
	Status    string `json:"status"`
	Operation string `json:"operation"`
}

type SearchLgLogWithPage struct {
	PageInfo
	IP     string `json:"ip"`
	Status string `json:"status"`
}

type SearchSSHLogWithPage struct {
	PageInfo
	IP       string `json:"ip"`
	Status   string `json:"status"`
	Username string `json:"username"`
}

type LoginLog struct {
	ID        uint      `json:"id"`
	IP        string    `json:"ip"`
	Address   string    `json:"address"`
	Agent     string    `json:"agent"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type SSHLoginLog struct {
	CreatedAt  string `json:"createdAt"`
	Status     string `json:"status"`
	Username   string `json:"username"`
	SourceIP   string `json:"sourceIp"`
	SourcePort string `json:"sourcePort"`
	AuthMethod string `json:"authMethod"`
	Message    string `json:"message"`
	Raw        string `json:"raw"`
	Platform   string `json:"platform"`
	Source     string `json:"source"`
}

type SSHLoginLogResult struct {
	Supported       bool          `json:"supported"`
	Platform        string        `json:"platform"`
	Source          string        `json:"source"`
	Partial         bool          `json:"partial"`
	Warning         string        `json:"warning"`
	Items           []SSHLoginLog `json:"items"`
	Total           int           `json:"total"`
	SuccessfulCount int           `json:"successfulCount"`
	FailedCount     int           `json:"failedCount"`
}

type CleanLog struct {
	LogType string `json:"logType" validate:"required,oneof=login operation"`
}

type SearchSystemLog struct {
	Name string `json:"name" validate:"required"`
}
