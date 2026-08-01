package model

type AppDeploy struct {
	BaseModel
	WebsiteID        uint   `gorm:"type:integer;not null;index" json:"websiteId"`
	ReleaseID        uint   `gorm:"type:integer;index" json:"releaseId"`
	PipelineRecordID uint   `gorm:"type:integer;index" json:"pipelineRecordId"`
	Version          string `gorm:"type:varchar;not null" json:"version"`
	SourceType       string `gorm:"type:varchar;not null" json:"sourceType"`
	SourceUrl        string `gorm:"type:varchar;" json:"sourceUrl"`
	ArchiveFile      string `gorm:"type:varchar;" json:"archiveFile"`
	ReleaseDir       string `gorm:"type:varchar;" json:"releaseDir"`
	RuntimeDir       string `gorm:"type:varchar;" json:"runtimeDir"`
	ImageTag         string `gorm:"type:varchar;" json:"imageTag"`
	Status           string `gorm:"type:varchar;not null" json:"status"`
	LogText          string `gorm:"type:longtext;" json:"logText"`
	ContainerID      string `gorm:"type:varchar;" json:"containerId"`
	RuntimeHost      string `gorm:"type:varchar(255);" json:"runtimeHost"`
	Port             int    `gorm:"type:integer;" json:"port"`
	IsActive         bool   `gorm:"type:boolean;default:false" json:"isActive"`
	DockerCompose    string `json:"dockerCompose" gorm:"type:longtext"`
	Env              string `json:"env" gorm:"type:longtext"`
	AppInstallID     uint   `json:"appInstallId"`
}

func (AppDeploy) TableName() string {
	return "app_deploys"
}
