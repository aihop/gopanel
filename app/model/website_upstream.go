package model

type WebsiteUpstream struct {
	BaseModel
	WebsiteID      uint   `gorm:"column:website_id;type:integer;not null;index" json:"websiteId"`
	Address        string `gorm:"type:varchar(255);not null" json:"address"`
	Scheme         string `gorm:"type:varchar(16);default:'http'" json:"scheme"`
	Weight         int    `gorm:"type:integer;default:1" json:"weight"`
	Enabled        bool   `gorm:"type:boolean;default:true" json:"enabled"`
	Backup         bool   `gorm:"type:boolean;default:false" json:"backup"`
	HealthURI      string `gorm:"column:health_uri;type:varchar(255)" json:"healthUri"`
	HealthInterval string `gorm:"column:health_interval;type:varchar(32)" json:"healthInterval"`
	HealthTimeout  string `gorm:"column:health_timeout;type:varchar(32)" json:"healthTimeout"`
	Transport      string `gorm:"type:varchar(32);default:'http'" json:"transport"`
	Sort           int    `gorm:"type:integer;default:0" json:"sort"`
}

func (WebsiteUpstream) TableName() string {
	return "website_upstream"
}
