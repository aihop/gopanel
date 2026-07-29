package model

type UserNote struct {
	BaseModel
	UserID  uint   `gorm:"not null;uniqueIndex" json:"userID"`
	Content string `gorm:"type:text;not null" json:"content"`
}
