package repo

import "gorm.io/gorm"

func NewSnapshot(db *gorm.DB) *SnapshotRepo {
	return &SnapshotRepo{db: db}
}

type SnapshotRepo struct {
	db *gorm.DB
}
