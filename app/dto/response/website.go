package response

import "time"

type WebsiteRes struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Protocol      string    `json:"protocol"`
	PrimaryDomain string    `json:"primaryDomain"`
	OtherDomains  string    `json:"otherDomains"`
	DefaultServer bool      `json:"defaultServer"`
	Proxy         string    `json:"proxy"`
	IPV6          bool      `json:"ipv6"`
	Type          string    `json:"type"`
	Alias         string    `json:"alias"`
	Remark        string    `json:"remark"`
	Status        string    `json:"status"`
	CodeSource    string    `json:"codeSource"`
	ExpireDate    time.Time `json:"expireDate"`
	SitePath      string    `json:"sitePath"`
	AppName       string    `json:"appName"`
	RuntimeName   string    `json:"runtimeName"`
	RuntimeDir    string    `json:"runtimeDir"`
	AppInstallID  uint      `json:"appInstallId"`
	PipelineID    uint      `json:"pipelineId"`
	RuntimeType   string    `json:"runtimeType"`
}
