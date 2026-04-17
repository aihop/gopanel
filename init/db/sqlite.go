package db

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type SQLite struct {
	Db *gorm.DB
}

func NewSQLite(dsn string) (*SQLite, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &SQLite{Db: db}, nil
}

func (s *SQLite) Close() error {
	sqlDB, err := s.Db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
