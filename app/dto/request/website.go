package request

type WebsiteCreate struct {
	PrimaryDomain string        `json:"primaryDomain" validate:"required"`
	Type          string        `json:"type" validate:"required"`
	Alias         string        `json:"alias" validate:"required"`
	Remark        string        `json:"remark"`
	OtherDomains  string        `json:"otherDomains"`
	Proxy         string        `json:"proxy"`
	IPV6          bool          `json:"IPV6"`
	Protocol      string        `json:"protocol"`
	AppInstall    NewAppInstall `json:"appInstall"`
	AppID         uint          `json:"appID"`
	AppInstallID  uint          `json:"appInstallId"`

	CodeSource          string                 `json:"codeSource"` // upload, git(legacy image source), pipeline, app_store
	GitRepo             string                 `json:"gitRepo"`    // legacy field name, currently used as Docker image reference
	PipelineId          uint                   `json:"pipelineId"`
	CodeDir             string                 `json:"codeDir"`
	CodeDirFallback     string                 `json:"-"`
	PreviousContainerID string                 `json:"-"`
	PipelineKey         string                 `json:"-"`
	RunnerConfig        map[string]interface{} `json:"-"`

	AntiCrawler    bool   `json:"antiCrawler"`
	AntiLeech      bool   `json:"antiLeech"`
	RateLimitMode  string `json:"rateLimitMode"`
	WafEnable      bool   `json:"wafEnable"`
	BlockSensitive bool   `json:"blockSensitive"`
	IPAllowlist    string `json:"ipAllowlist"`
	IPBlocklist    string `json:"ipBlocklist"`
	SecurityHeader bool   `json:"securityHeader"`
	HstsEnabled    bool   `json:"hstsEnabled"`

	HttpConfig string `json:"httpConfig"`
}

type NewAppInstall struct {
	Name        string                 `json:"name"`
	AppDetailId uint                   `json:"appDetailID"`
	Params      map[string]interface{} `json:"params"`

	AppContainerConfig
}

type WebsiteUpdate struct {
	ID            uint   `json:"id"`
	PrimaryDomain string `json:"primaryDomain"`
	Protocol      string `json:"protocol"`
	Remark        string `json:"remark"`
	IPV6          bool   `json:"IPV6"`
	OtherDomains  string `json:"otherDomains"`
	Proxy         string `json:"proxy"`
	PipelineId    uint   `json:"pipelineId"`
	CodeSource    string `json:"codeSource"`

	AntiCrawler    bool   `json:"antiCrawler"`
	AntiLeech      bool   `json:"antiLeech"`
	RateLimitMode  string `json:"rateLimitMode"`
	WafEnable      bool   `json:"wafEnable"`
	BlockSensitive bool   `json:"blockSensitive"`
	IPAllowlist    string `json:"ipAllowlist"`
	IPBlocklist    string `json:"ipBlocklist"`
	SecurityHeader bool   `json:"securityHeader"`
	HstsEnabled    bool   `json:"hstsEnabled"`

	HttpConfig string `json:"httpConfig"`
}

type WebsiteLogRead struct {
	WebsiteID uint   `json:"websiteId" validate:"required"`
	Page      int    `json:"page" validate:"required"`
	Limit     int    `json:"limit" validate:"required"`
	Latest    bool   `json:"latest"`
	LogType   string `json:"logType"`
}

type WebsiteLogTodayIPStats struct {
	WebsiteID uint `json:"websiteId" validate:"required"`
}
