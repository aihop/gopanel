package model

type WebsiteDomain struct {
	BaseModel
	WebsiteID uint   `gorm:"column:website_id;type:varchar(64);not null;" json:"websiteId"`
	Domain    string `gorm:"type:varchar(128);not null" json:"domain"`
	Port      int    `gorm:"type:integer" json:"port"`
	IsPrimary uint   `gorm:"type:tinyint(3);default:10;" json:"isPrimary"`
}
