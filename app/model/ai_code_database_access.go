package model

import "time"

type AICodeDatabaseAccess struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	ProjectID    uint            `gorm:"column:project_id;not null;uniqueIndex:idx_code_db_project_alias" json:"projectId"`
	ServerID     uint            `gorm:"column:server_id;not null;index" json:"serverId"`
	DatabaseName string          `gorm:"column:database_name;type:varchar(255);not null" json:"databaseName"`
	Alias        string          `gorm:"column:alias;type:varchar(64);not null;uniqueIndex:idx_code_db_project_alias" json:"alias"`
	ReadOnly     bool            `gorm:"column:read_only;not null;default:true" json:"readOnly"`
	Server       *DatabaseServer `gorm:"foreignKey:ServerID;references:ID" json:"-"`
}

func (AICodeDatabaseAccess) TableName() string {
	return "ai_code_database_accesses"
}
