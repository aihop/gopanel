package model

type CloudAccount struct {
	BaseModel
	Name          string               `json:"name" gorm:"type:varchar(255);not null"`
	Type          string               `json:"type" gorm:"type:varchar(64);not null"` // e.g., aliyun, tencentcloud, cloudflare
	Authorization string               `json:"authorization" gorm:"type:text"`        // JSON serialized credentials
	Services      CloudAccountServices `json:"services" gorm:"serializer:json;type:json"`
}

type CloudAccountServices struct {
	Storage bool `json:"storage"` // 存储 (OSS/S3)
	DNS     bool `json:"dns"`     // 域名解析
	Host    bool `json:"host"`    // 主机
}

type CloudAccountStorage struct {
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
}
