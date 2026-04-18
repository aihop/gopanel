package model

type AcmeAccount struct {
	BaseModel
	Email      string `gorm:"type:varchar(255)" json:"email"`
	URL        string `gorm:"type:varchar(255)" json:"url"`
	Type       string `gorm:"type:varchar(64)" json:"type"`
	PrivateKey string `gorm:"type:text" json:"privateKey"`
}
