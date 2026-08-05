package model

import "time"

type AIGitCredential struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatorID uint      `gorm:"column:creator_id;not null;index" json:"creatorId"`
	Name      string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Username  string    `gorm:"column:username;type:varchar(255);not null" json:"username"`
	Secret    string    `gorm:"column:secret;type:text;not null" json:"-"`
}

func (AIGitCredential) TableName() string { return "ai_git_credentials" }
