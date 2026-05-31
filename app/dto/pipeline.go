package dto

type PipelineActionParams struct {
	OutputImage string `json:"outputImage"` // 产出镜像名
	ExposePort  int    `json:"exposePort"`  // 服务端口
}
