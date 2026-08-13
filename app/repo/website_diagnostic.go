package repo

import (
	"context"
	"errors"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WebsiteDiagnosticRepo struct {
	db *gorm.DB
}

func (r *WebsiteDiagnosticRepo) DeleteByWebsiteID(ctx context.Context, websiteID uint) error {
	return r.db.WithContext(ctx).Where("website_id = ?", websiteID).Delete(&model.WebsiteDiagnosticSetting{}).Error
}

func NewWebsiteDiagnostic(db *gorm.DB) *WebsiteDiagnosticRepo {
	return &WebsiteDiagnosticRepo{db: db}
}

func (r *WebsiteDiagnosticRepo) GetByWebsiteID(websiteID uint) (*model.WebsiteDiagnosticSetting, error) {
	var setting model.WebsiteDiagnosticSetting
	err := r.db.Where("website_id = ?", websiteID).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &setting, err
}

func (r *WebsiteDiagnosticRepo) ListByWebsiteIDs(websiteIDs []uint) ([]model.WebsiteDiagnosticSetting, error) {
	if len(websiteIDs) == 0 {
		return []model.WebsiteDiagnosticSetting{}, nil
	}
	var settings []model.WebsiteDiagnosticSetting
	err := r.db.Where("website_id IN ?", websiteIDs).Find(&settings).Error
	return settings, err
}

func (r *WebsiteDiagnosticRepo) Save(setting *model.WebsiteDiagnosticSetting) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "website_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"code_project_id", "enabled", "caddy_monitoring", "active_probes", "backend_hook", "browser_hook",
			"auto_analysis", "monitor_http_4xx", "monitor_http_5xx", "monitor_upstream_errors", "monitor_slow_requests",
			"monitor_business_errors", "monitor_browser_errors", "monitor_resource_errors", "slow_request_threshold_ms",
			"trigger_count", "trigger_window_minutes", "retention_days", "default_executor_id", "approval_policy", "updated_at",
		}),
	}).Create(setting).Error
}
