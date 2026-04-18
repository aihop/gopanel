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
	AppInstallID  uint          `json:"appInstallID"`

	CodeSource          string `json:"codeSource"` // upload, git, pipeline, app_store
	GitRepo             string `json:"gitRepo"`    // Git URL or Docker Image Name
	PipelineId          uint   `json:"pipelineId"`
	CodeDir             string `json:"codeDir"`
	CodeDirFallback     string `json:"-"`
	PreviousContainerID string `json:"-"`
}

type NewAppInstall struct {
	Name        string                 `json:"name"`
	AppDetailId uint                   `json:"appDetailID"`
	Params      map[string]interface{} `json:"params"`

	AppContainerConfig
}

type WebsiteUpdate struct {
	ID            uint   `json:"id" validate:"required"`
	PrimaryDomain string `json:"primaryDomain"`
	Protocol      string `json:"protocol"`
	Remark        string `json:"remark"`
	IPV6          bool   `json:"IPV6"`
	OtherDomains  string `json:"otherDomains"`
	Proxy         string `json:"proxy"`
	PipelineId    uint   `json:"pipelineId"`
	CodeSource    string `json:"codeSource"`
}
