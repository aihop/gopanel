package model

import "time"

type Pipeline struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Name        string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description string    `gorm:"column:description;type:varchar(255)" json:"description"`
	RepoUrl     string    `gorm:"column:repo_url;type:varchar(255)" json:"repoUrl"` // 设为非必填，为空则代表纯脚本模式
	Branch      string    `gorm:"column:branch;type:varchar(100);not null;default:'main'" json:"branch"`

	Version string `gorm:"column:version;type:varchar(50);not null;default:'1.0.0'" json:"version"` // 当前版本号

	// authType: "none", "password", "token"
	AuthType string `gorm:"column:auth_type;type:varchar(20);not null;default:'none'" json:"authType"`
	AuthData string `gorm:"column:auth_data;type:text" json:"authData"` // JSON format credentials

	BuildImage   string `gorm:"column:build_image;type:varchar(100);not null;default:'node:18-alpine'" json:"buildImage"`
	BuildScript  string `gorm:"column:build_script;type:text" json:"buildScript"`
	OutputImage  string `gorm:"column:output_image;type:varchar(255)" json:"outputImage"`
	ArtifactPath string `gorm:"column:artifact_path;type:varchar(255);not null;default:'dist/'" json:"artifactPath"`
	ExposePort   int    `gorm:"column:expose_port;type:int;not null;default:80" json:"exposePort"`
}

func (Pipeline) TableName() string {
	return "pipelines"
}

type PipelineRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	PipelineID   uint      `gorm:"column:pipeline_id;type:integer;not null" json:"pipelineId"`
	Status       string    `gorm:"column:status;type:varchar(20);not null;default:'pending'" json:"status"` // pending, cloning, building, deploying, success, failed
	Version      string    `gorm:"column:version;type:varchar(50)" json:"version"`                          // 记录本次执行的版本号
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"errorMessage"`
	ArchiveFile  string    `gorm:"column:archive_file;type:varchar(255)" json:"archiveFile"` // Path to the zip backup
}

func (PipelineRecord) TableName() string {
	return "pipeline_records"
}
