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

	ActionType   string `gorm:"column:action_type;type:varchar(32);not null;default:'none'" json:"actionType"`
	ActionParams string `gorm:"column:action_params;type:text" json:"actionParams"`
	BuildImage   string `gorm:"column:build_image;type:varchar(100);not null" json:"buildImage"`
	BuildScript  string `gorm:"column:build_script;type:text" json:"buildScript"`
	ArtifactPath string `gorm:"column:artifact_path;type:varchar(255);not null;default:'dist/'" json:"artifactPath"`
	PipelineKey  string `gorm:"column:pipeline_key;uniqueIndex;type:varchar(100)" json:"pipelineKey"`

	RunnerMode   string `gorm:"column:runner_mode;type:varchar(32)" json:"runnerMode"`
	RunnerConfig string `gorm:"column:runner_config;type:longtext" json:"runnerConfig"`
	RuntimeHost  string `gorm:"-" json:"runtimeHost"`
	RuntimeKind  string `gorm:"-" json:"runtimeKind"`
	RuntimeMode  string `gorm:"-" json:"runtimeMode"`
	RunUser      string `gorm:"-" json:"runUser"`
}

func (Pipeline) TableName() string {
	return "pipelines"
}

type PipelineRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	PipelineID     uint      `gorm:"column:pipeline_id;type:integer;not null" json:"pipelineId"`
	Status         string    `gorm:"column:status;type:varchar(20);not null;default:'pending'" json:"status"` // pending, cloning, building, deploying, success, failed
	Version        string    `gorm:"column:version;type:varchar(50)" json:"version"`                          // 记录本次执行的版本号
	ExpectedCommit string    `gorm:"column:expected_commit;type:varchar(64);index" json:"expectedCommit"`
	CommitHash     string    `gorm:"column:commit_hash;type:varchar(64)" json:"commitHash"`
	// Changelog 本次构建包含的提交标题，一行一条，来自「上次成功构建的 commit..HEAD」。
	// 只存纯文本，展示端不要按 HTML 渲染——内容来自仓库提交信息，属于外部输入。
	Changelog    string `gorm:"column:changelog;type:text" json:"changelog"`
	ErrorMessage string `gorm:"column:error_message;type:text" json:"errorMessage"`
	ArchiveFile  string `gorm:"column:archive_file;type:varchar(255)" json:"archiveFile"` // Path to the zip backup
	ImageTag     string `gorm:"column:image_tag;type:varchar(255)" json:"imageTag"`
	ImageID      string `gorm:"column:image_id;type:varchar(255)" json:"imageId"`
	ImageDigest  string `gorm:"column:image_digest;type:varchar(255);index" json:"imageDigest"`
	ImageRef     string `gorm:"column:image_ref;type:varchar(512)" json:"imageRef"`

	RunnerReleaseDir  string `gorm:"column:runner_release_dir;type:varchar(255)" json:"runnerReleaseDir"`
	RunnerContainerID string `gorm:"column:runner_container_id;type:varchar(128)" json:"runnerContainerId"`
	RunnerHostPort    int    `gorm:"column:runner_host_port;type:int" json:"runnerHostPort"`
	RuntimeHost       string `gorm:"-" json:"runtimeHost"`
	RuntimeKind       string `gorm:"-" json:"runtimeKind"`
	RuntimeMode       string `gorm:"-" json:"runtimeMode"`
	RunUser           string `gorm:"-" json:"runUser"`
	Released          bool   `gorm:"-" json:"released"`
}

func (PipelineRecord) TableName() string {
	return "pipeline_records"
}

type Release struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	PipelineID       uint      `gorm:"column:pipeline_id;type:integer;not null;index" json:"pipelineId"`
	PipelineRecordID uint      `gorm:"column:pipeline_record_id;type:integer;not null;uniqueIndex:uniq_release_pipeline_record" json:"pipelineRecordId"`
	Version          string    `gorm:"column:version;type:varchar(50);not null;index" json:"version"`
	CommitHash       string    `gorm:"column:commit_hash;type:varchar(64);index" json:"commitHash"`
	Changelog        string    `gorm:"column:changelog;type:text" json:"changelog"` // 发布时从构建记录复制，一行一条提交标题
	SourceType       string    `gorm:"column:source_type;type:varchar(32);not null;default:'archive';index" json:"sourceType"`
	ImageTag         string    `gorm:"column:image_tag;type:varchar(255);index" json:"imageTag"`
	ImageDigest      string    `gorm:"column:image_digest;type:varchar(255);index" json:"imageDigest"`
	ArchiveFile      string    `gorm:"column:archive_file;type:varchar(255)" json:"archiveFile"`
	ReleaseDir       string    `gorm:"column:release_dir;type:varchar(255)" json:"releaseDir"`
	ArtifactDigest   string    `gorm:"column:artifact_digest;type:varchar(255);index" json:"artifactDigest"`
	ArtifactManifest string    `gorm:"column:artifact_manifest;type:longtext" json:"artifactManifest"`
	ArtifactMeta     string    `gorm:"column:artifact_meta;type:longtext" json:"artifactMeta"`
	Status           string    `gorm:"column:status;type:varchar(32);not null;default:'ready';index" json:"status"`
	Remark           string    `gorm:"column:remark;type:varchar(255)" json:"remark"`
}

func (Release) TableName() string {
	return "releases"
}
