package model

type SSLPushRule struct {
	BaseModel
	SSLID          uint   `json:"sslId"`          // 关联的证书ID
	CloudAccountID uint   `json:"cloudAccountId"` // 关联的云账号授权ID (复用 CloudAccount 概念)
	TargetDomain   string `json:"targetDomain"`   // 目标部署的云端域名 (如为空，则默认推送到该云账号下的所有匹配域名或主域名)
	Status         string `json:"status"`         // 推送状态：pending, success, error
	Message        string `json:"message"`        // 推送日志或报错信息
}
