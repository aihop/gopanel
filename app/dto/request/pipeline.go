package request

type PipelineCreate struct {
	Name         string `json:"name" validate:"required"`
	Description  string `json:"description"`
	RepoUrl      string `json:"repoUrl"` // 设为非必填
	Branch       string `json:"branch" validate:"required"`
	Version      string `json:"version" validate:"required"` // 新增版本号
	AuthType     string `json:"authType" validate:"required"`
	AuthData     string `json:"authData"`
	BuildImage   string `json:"buildImage" validate:"required"`
	BuildScript  string `json:"buildScript" validate:"required"`
	OutputImage  string `json:"outputImage"`
	ArtifactPath string `json:"artifactPath"`
	ExposePort   int    `json:"exposePort"`
}

type PipelineUpdate struct {
	ID           uint   `json:"id" validate:"required"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RepoUrl      string `json:"repoUrl"`
	Branch       string `json:"branch"`
	Version      string `json:"version"` // 新增版本号
	AuthType     string `json:"authType"`
	AuthData     string `json:"authData"`
	BuildImage   string `json:"buildImage"`
	BuildScript  string `json:"buildScript"`
	OutputImage  string `json:"outputImage"`
	ArtifactPath string `json:"artifactPath"`
	ExposePort   int    `json:"exposePort"`
}

type PipelineRun struct {
	ID      uint   `json:"id" validate:"required"`
	Version string `json:"version" validate:"required"` // 触发时必须指定本次执行的版本号
}
