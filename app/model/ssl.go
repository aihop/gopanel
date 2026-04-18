package model

import (
	"fmt"
	"path"
	"time"

	"github.com/aihop/gopanel/constant"
)

type SSL struct {
	BaseModel
	PrimaryDomain  string        `json:"primaryDomain"`
	PrivateKey     string        `json:"privateKey"`
	Pem            string        `json:"pem"`
	Domains        string        `json:"domains"`
	CertURL        string        `json:"certURL"`
	Type           string        `json:"type"`
	Provider       string        `json:"provider"`
	Organization   string        `json:"organization"`
	CloudAccountID uint          `json:"cloudAccountId"`
	AcmeAccountID  uint          `gorm:"column:acme_account_id" json:"acmeAccountId" `
	DnsAccountID   uint          `json:"dnsAccountId"` // 专门用于域名DNS验证的云账号
	CaID           uint          `json:"caId"`
	AutoRenew      bool          `json:"autoRenew"`
	ExpireDate     time.Time     `json:"expireDate"`
	StartDate      time.Time     `json:"startDate"`
	Status         string        `json:"status"`
	Message        string        `json:"message"`
	KeyType        string        `json:"keyType"`
	PushDir        bool          `json:"pushDir"`
	Dir            string        `json:"dir"`
	Description    string        `json:"description"`
	SkipDNS        bool          `json:"skipDNS"`
	Nameserver1    string        `json:"nameserver1"`
	Nameserver2    string        `json:"nameserver2"`
	DisableCNAME   bool          `json:"disableCNAME"`
	ExecShell      bool          `json:"execShell"`
	Shell          string        `json:"shell"`
	Websites       []Website     `json:"websites" gorm:"-"`
	PushRules      []SSLPushRule `json:"pushRules" gorm:"foreignKey:SSLID"`
}

func (w SSL) GetLogPath() string {
	return path.Join(constant.SSLLogDir, fmt.Sprintf("%s-ssl-%d.log", w.PrimaryDomain, w.ID))
}
