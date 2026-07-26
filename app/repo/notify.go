package repo

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type NotifyRepo struct{}

func NewNotify() *NotifyRepo {
	return &NotifyRepo{}
}

func (r *NotifyRepo) MigrateTable() error {
	return global.DB.AutoMigrate(&model.NotifyConfig{}, &model.AlertEvent{})
}

// GetConfig 通知配置是全局单行，取不到就返回一份带默认值的空配置，
// 让调用方不必到处判空。
func (r *NotifyRepo) GetConfig() (model.NotifyConfig, error) {
	var cfg model.NotifyConfig
	err := global.DB.First(&cfg).Error
	if err != nil {
		return model.NotifyConfig{
			SMTPPort:        587,
			SMTPTLSMode:     model.SMTPTLSStartTLS,
			DebounceTimes:   2,
			SilenceHours:    6,
			NotifyResolved:  true,
			EnableDisk:      true,
			EnableContainer: true,
			EnableOffline:   true,
		}, err
	}
	return cfg, nil
}

// SaveConfig 保存配置。
// Select("*") 是必需的：布尔开关关闭时是零值，不显式选中所有列的话
// GORM 会跳过它们，用户的「关闭」保存不进去。
func (r *NotifyRepo) SaveConfig(cfg *model.NotifyConfig) error {
	var existing model.NotifyConfig
	if err := global.DB.First(&existing).Error; err != nil {
		return global.DB.Select("*").Create(cfg).Error
	}
	cfg.ID = existing.ID
	cfg.CreatedAt = existing.CreatedAt
	return global.DB.Select("*").Save(cfg).Error
}

// ActiveEvents 取所有未恢复的事件，告警评估每轮都要跟它们比对
func (r *NotifyRepo) ActiveEvents() ([]model.AlertEvent, error) {
	var list []model.AlertEvent
	err := global.DB.Where("status <> ?", model.AlertStatusResolved).Find(&list).Error
	return list, err
}

func (r *NotifyRepo) SaveEvent(event *model.AlertEvent) error {
	return global.DB.Save(event).Error
}

// PageEvents 事件列表，最近的在前
func (r *NotifyRepo) PageEvents(page, limit int) (int64, []model.AlertEvent, error) {
	var total int64
	if err := global.DB.Model(&model.AlertEvent{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	var list []model.AlertEvent
	err := global.DB.Model(&model.AlertEvent{}).
		Order("updated_at desc").
		Offset((page - 1) * limit).Limit(limit).
		Find(&list).Error
	return total, list, err
}

// CleanupResolved 清理很久以前已恢复的事件，避免表无限增长
func (r *NotifyRepo) CleanupResolved(before time.Time) error {
	return global.DB.Where("status = ? AND resolved_at < ?", model.AlertStatusResolved, before).
		Delete(&model.AlertEvent{}).Error
}
