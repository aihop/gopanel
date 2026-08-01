package request

type WebsiteUpstream struct {
	Address        string `json:"address"`
	Scheme         string `json:"scheme"`
	Weight         int    `json:"weight"`
	Enabled        bool   `json:"enabled"`
	Backup         bool   `json:"backup"`
	HealthURI      string `json:"healthUri"`
	HealthInterval string `json:"healthInterval"`
	HealthTimeout  string `json:"healthTimeout"`
	Transport      string `json:"transport"`
}

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

	CodeSource string `json:"codeSource"` // upload, git(legacy image source), app_store
	GitRepo    string `json:"gitRepo"`    // legacy field name, currently used as Docker image reference
	CodeDir    string `json:"codeDir"`

	AntiCrawler              bool   `json:"antiCrawler"`
	AntiLeech                bool   `json:"antiLeech"`
	RateLimitMode            string `json:"rateLimitMode"`
	WafEnable                bool   `json:"wafEnable"`
	BlockSensitive           bool   `json:"blockSensitive"`
	IPAllowlist              string `json:"ipAllowlist"`
	IPBlocklist              string `json:"ipBlocklist"`
	SecurityHeader           bool   `json:"securityHeader"`
	HstsEnabled              bool   `json:"hstsEnabled"`
	RedirectCode             int    `json:"redirectCode"`
	RedirectDomainsToPrimary bool   `json:"redirectDomainsToPrimary"`

	HttpConfig string            `json:"httpConfig"`
	Upstreams  []WebsiteUpstream `json:"upstreams"`
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
	CodeSource    string `json:"codeSource"`

	AntiCrawler              bool   `json:"antiCrawler"`
	AntiLeech                bool   `json:"antiLeech"`
	RateLimitMode            string `json:"rateLimitMode"`
	WafEnable                bool   `json:"wafEnable"`
	BlockSensitive           bool   `json:"blockSensitive"`
	IPAllowlist              string `json:"ipAllowlist"`
	IPBlocklist              string `json:"ipBlocklist"`
	SecurityHeader           bool   `json:"securityHeader"`
	HstsEnabled              bool   `json:"hstsEnabled"`
	RedirectCode             int    `json:"redirectCode"`
	RedirectDomainsToPrimary bool   `json:"redirectDomainsToPrimary"`

	HttpConfig string            `json:"httpConfig"`
	Upstreams  []WebsiteUpstream `json:"upstreams"`
}

type WebsiteDomainBindingUpdate struct {
	WebsiteID                uint   `json:"websiteId" validate:"required"`
	PrimaryDomain            string `json:"primaryDomain" validate:"required"`
	OtherDomains             string `json:"otherDomains"`
	RedirectDomainsToPrimary bool   `json:"redirectDomainsToPrimary"`
	Confirm                  bool   `json:"confirm" validate:"required"`
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
