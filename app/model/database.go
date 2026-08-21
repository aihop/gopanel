package model

import (
	"time"
)

type DatabaseServerStatus uint

const (
	DatabaseServerStatusValid   DatabaseServerStatus = 20
	DatabaseServerStatusInvalid DatabaseServerStatus = 10
)

type DatabaseType string
type DatabaseMode string

const (
	DatabaseTypeMysql      DatabaseType = "mysql"
	DatabaseTypeMariaDB    DatabaseType = "mariadb"
	DatabaseTypePostgresql DatabaseType = "postgresql"
	DatabaseTypeMongoDB    DatabaseType = "mongodb"
	DatabaseSQLite         DatabaseType = "sqlite"
	DatabaseTypeRedis      DatabaseType = "redis"
)

const (
	DatabaseModeLocal  DatabaseMode = "local"
	DatabaseModeRemote DatabaseMode = "remote"
)

type Database struct {
	Type     DatabaseType `json:"type"`
	Name     string       `json:"name"`
	Server   string       `json:"server"`
	ServerID uint         `json:"serverId"`
	Encoding string       `json:"encoding"`
	Comment  string       `json:"comment"`
}

type DatabaseListWarning struct {
	ServerID   uint         `json:"serverId"`
	ServerName string       `json:"serverName"`
	ServerType DatabaseType `json:"serverType"`
	Code       string       `json:"code"`
	Message    string       `json:"message"`
}

type DatabaseListResult struct {
	Total    int64                 `json:"total"`
	Items    []*Database           `json:"items"`
	Warnings []DatabaseListWarning `json:"warnings"`
}

type DatabaseServer struct {
	ID        uint                 `gorm:"primaryKey" json:"id"`
	Name      string               `gorm:"not null;default:'';unique" json:"name"`
	Type      DatabaseType         `gorm:"not null;default:''" json:"type"`
	Host      string               `gorm:"not null;default:''" json:"host"`
	Port      uint                 `gorm:"not null;default:0" json:"port"`
	Username  string               `gorm:"not null;default:''" json:"username"`
	Password  string               `gorm:"not null;default:''" json:"-"`
	Status    DatabaseServerStatus `gorm:"-:all" json:"status"`
	Remark    string               `gorm:"not null;default:''" json:"remark"`
	Mode      DatabaseMode         `gorm:"not null;default:''" json:"mode"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

type DatabaseUserStatus uint

const (
	DatabaseUserStatusUnknown DatabaseUserStatus = 0
	DatabaseUserStatusValid   DatabaseUserStatus = 20
	DatabaseUserStatusInvalid DatabaseUserStatus = 10
)

type DatabaseUserAccessScope string

const (
	DatabaseUserAccessScopeUnknown  DatabaseUserAccessScope = "unknown"
	DatabaseUserAccessScopeLocal    DatabaseUserAccessScope = "local"
	DatabaseUserAccessScopeSpecific DatabaseUserAccessScope = "specific"
	DatabaseUserAccessScopeAll      DatabaseUserAccessScope = "all"
)

type DatabaseUser struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	ServerID        uint                    `gorm:"not null;default:0" json:"serverId"`
	Username        string                  `gorm:"not null;default:''" json:"username"`
	Password        string                  `gorm:"not null;default:''" json:"-"`
	Host            string                  `gorm:"not null;default:''" json:"host"` // 仅 mysql
	Status          DatabaseUserStatus      `gorm:"-:all" json:"status"`             // 仅显示
	AccessScope     DatabaseUserAccessScope `gorm:"-:all" json:"accessScope"`        // 仅显示
	PasswordManaged bool                    `gorm:"-:all" json:"passwordManaged"`    // 仅显示
	Privileges      []string                `gorm:"-:all" json:"privileges"`         // 仅显示
	Remark          string                  `gorm:"not null;default:''" json:"remark"`
	CreatedAt       time.Time               `json:"createdAt"`
	UpdatedAt       time.Time               `json:"updatedAt"`

	Server *DatabaseServer `gorm:"foreignKey:ServerID;references:ID" json:"server"`
}
