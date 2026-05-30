package model

import "time"

type Website struct {
	BaseModel
	Protocol      string    `gorm:"type:varchar;not null" json:"protocol"`
	PrimaryDomain string    `gorm:"type:varchar;not null" json:"primaryDomain"`
	Type          string    `gorm:"type:varchar;not null" json:"type"`
	Alias         string    `gorm:"type:varchar;not null" json:"alias"`
	Remark        string    `gorm:"type:longtext;" json:"remark"`
	Status        string    `gorm:"type:varchar;not null" json:"status"`
	HttpConfig    string    `gorm:"type:text;" json:"httpConfig"`
	ExpireDate    time.Time `json:"expireDate"`

	Proxy         string `gorm:"type:varchar;" json:"proxy"`
	SiteDir       string `gorm:"type:varchar;" json:"siteDir"`
	RuntimeDir    string `gorm:"type:varchar;" json:"runtimeDir"`
	CodeSource    string `gorm:"type:varchar;" json:"codeSource"`
	ErrorLog      bool   `json:"errorLog"`
	AccessLog     bool   `json:"accessLog"`
	DefaultServer bool   `json:"defaultServer"`
	IPV6          bool   `json:"IPV6"`
	Rewrite       string `gorm:"type:varchar" json:"rewrite"`

	AppInstallID uint `gorm:"type:integer" json:"appInstallId"`
	PipelineID   uint `gorm:"type:integer;column:pipeline_id" json:"pipelineId"`

	EngineEnv   string `gorm:"type:varchar;" json:"engineEnv"`
	ContainerID string `gorm:"type:varchar;" json:"containerId"`
	Message     string `gorm:"type:text;" json:"message"`

	AntiCrawler    bool   `json:"antiCrawler" gorm:"type:boolean;default:false"`
	AntiLeech      bool   `json:"antiLeech" gorm:"type:boolean;default:false"`
	RateLimitMode  string `json:"rateLimitMode" gorm:"type:varchar(32);default:'none'"`
	WafEnable      bool   `json:"wafEnable" gorm:"type:boolean;default:false"`
	BlockSensitive bool   `json:"blockSensitive" gorm:"type:boolean;default:false"`
	IPAllowlist    string `json:"ipAllowlist" gorm:"type:text"`
	IPBlocklist    string `json:"ipBlocklist" gorm:"type:text"`
	SecurityHeader bool   `json:"securityHeader" gorm:"type:boolean;default:false"`
	HstsEnabled    bool   `json:"hstsEnabled" gorm:"type:boolean;default:false"`

	User    string           `gorm:"type:varchar;" json:"user"`
	Group   string           `gorm:"type:varchar;" json:"group"`
	Domains []*WebsiteDomain `json:"domains" gorm:"-:migration"`
}

func (Website) TableName() string {
	return "website"
}
