package model

import "github.com/aihop/gopanel/constant"

type BackupAccount struct {
	BaseModel
	Type       string `gorm:"type:varchar(64);unique;not null" json:"type"`
	Bucket     string `gorm:"type:varchar(256)" json:"bucket"`
	AccessKey  string `gorm:"type:varchar(256)" json:"accessKey"`
	Credential string `gorm:"type:varchar(256)" json:"credential"`
	BackupPath string `gorm:"type:varchar(256)" json:"backupPath"`
	Vars       string `gorm:"type:longText" json:"vars"`
}

type BackupSource string

const (
	BackupSourceS3       BackupSource = constant.S3
	BackupSourceOSS      BackupSource = constant.OSS
	BackupSourceSFTP     BackupSource = constant.Sftp
	BackupSourceOneDrive BackupSource = constant.OneDrive
	BackupSourceMinIO    BackupSource = constant.MinIo
	BackupSourceCOS      BackupSource = constant.Cos
	BackupSourceKODO     BackupSource = constant.Kodo
	BackupSourceWebDAV   BackupSource = constant.WebDAV
	BackupSourceLOCAL    BackupSource = constant.Local
)

type BackupRecord struct {
	BaseModel
	From       string       `gorm:"type:varchar(64)" json:"from"`
	CronjobID  uint         `gorm:"type:decimal" json:"cronjobID"`
	Type       string       `gorm:"type:varchar(64);not null" json:"type"`
	Name       string       `gorm:"type:varchar(64);not null" json:"name"`
	DetailName string       `gorm:"type:varchar(256)" json:"detailName"`
	Source     BackupSource `gorm:"type:varchar(60)" json:"source"`
	BackupType string       `gorm:"type:varchar(256)" json:"backupType"`
	FileDir    string       `gorm:"type:varchar(256)" json:"fileDir"`
	FileName   string       `gorm:"type:varchar(256)" json:"fileName"`
}
