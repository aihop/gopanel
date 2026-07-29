package model

import "time"

type MobilePairing struct {
	BaseModel
	UserID    uint       `gorm:"not null;index" json:"userId"`
	CodeHash  string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expiresAt"`
	UsedAt    *time.Time `gorm:"index" json:"usedAt,omitempty"`
}

func (MobilePairing) TableName() string {
	return "mobile_pairings"
}

type MobileDevice struct {
	BaseModel
	UserID     uint       `gorm:"not null;index" json:"userId"`
	Name       string     `gorm:"type:varchar(128);not null" json:"name"`
	TokenHash  string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expiresAt"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
	LastIP     string     `gorm:"type:varchar(64)" json:"lastIp"`
	LastAgent  string     `gorm:"type:varchar(255)" json:"lastAgent"`
	RevokedAt  *time.Time `gorm:"index" json:"revokedAt,omitempty"`
}

func (MobileDevice) TableName() string {
	return "mobile_devices"
}
