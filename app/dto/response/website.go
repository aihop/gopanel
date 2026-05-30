package response

import "time"

type WebsiteDeploySummary struct {
	ID               uint      `json:"id"`
	Version          string    `json:"version"`
	ReleaseID        uint      `json:"releaseId"`
	PipelineRecordID uint      `json:"pipelineRecordId"`
	SourceType       string    `json:"sourceType"`
	ImageTag         string    `json:"imageTag"`
	Status           string    `json:"status"`
	IsActive         bool      `json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
}

type WebsiteRes struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Protocol      string    `json:"protocol"`
	PrimaryDomain string    `json:"primaryDomain"`
	OtherDomains  string    `json:"otherDomains"`
	DefaultServer bool      `json:"defaultServer"`
	Proxy         string    `json:"proxy"`
	IPV6          bool      `json:"IPV6"`
	Ipv6          bool      `json:"ipv6"`
	Type          string    `json:"type"`
	Alias         string    `json:"alias"`
	Remark        string    `json:"remark"`
	Status        string    `json:"status"`
	CodeSource    string    `json:"codeSource"`
	ExpireDate    time.Time `json:"expireDate"`
	SitePath      string    `json:"sitePath"`
	AccessLogPath string    `json:"accessLogPath"`
	ErrorLogPath  string    `json:"errorLogPath"`
	AppName       string    `json:"appName"`
	RuntimeName   string    `json:"runtimeName"`
	RuntimeDir    string    `json:"runtimeDir"`
	AppInstallID  uint      `json:"appInstallId"`
	PipelineID    uint      `json:"pipelineId"`
	RuntimeType   string    `json:"runtimeType"`
	RuntimeHost   string    `json:"runtimeHost"`
	RuntimeKind   string    `json:"runtimeKind"`
	RuntimeMode   string    `json:"runtimeMode"`
	RunUser       string    `json:"runUser"`

	AntiCrawler    bool   `json:"antiCrawler"`
	AntiLeech      bool   `json:"antiLeech"`
	RateLimitMode  string `json:"rateLimitMode"`
	WafEnable      bool   `json:"wafEnable"`
	BlockSensitive bool   `json:"blockSensitive"`
	IPAllowlist    string `json:"ipAllowlist"`
	IPBlocklist    string `json:"ipBlocklist"`
	SecurityHeader bool   `json:"securityHeader"`
	HstsEnabled    bool   `json:"hstsEnabled"`
	HttpConfig     string `json:"httpConfig"`

	ActiveRelease      *WebsiteDeploySummary `json:"activeRelease,omitempty"`
	LatestPipelineSync *WebsiteDeploySummary `json:"latestPipelineSync,omitempty"`
}

type WebsiteLogTopIP struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

type WebsiteLogTodayIPStats struct {
	Date          string            `json:"date"`
	UniqueIPCount int               `json:"uniqueIpCount"`
	RequestCount  int               `json:"requestCount"`
	Path          string            `json:"path"`
	TopIPs        []WebsiteLogTopIP `json:"topIps"`
}
