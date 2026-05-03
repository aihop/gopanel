package repo

import "gorm.io/gorm"

type PipelineRepo struct{ db *gorm.DB }

func NewPipeline(db *gorm.DB) *PipelineRepo {
	return &PipelineRepo{db: db}
}

type PipelineRecordRepo struct{ db *gorm.DB }

func NewPipelineRecord(db *gorm.DB) *PipelineRecordRepo {
	return &PipelineRecordRepo{db: db}
}

type ReleaseRepo struct{ db *gorm.DB }

func NewRelease(db *gorm.DB) *ReleaseRepo {
	return &ReleaseRepo{db: db}
}
