package model

type CloudAccount struct {
	BaseModel
	Name          string `json:"name" gorm:"type:varchar(255);not null"`
	Type          string `json:"type" gorm:"type:varchar(64);not null"` // e.g., aliyun, tencentcloud, cloudflare
	Authorization string `json:"authorization" gorm:"type:text"`        // JSON serialized credentials
}

type CloudAccountStorage struct {
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
}
