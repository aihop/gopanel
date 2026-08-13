package repo

import (
	"context"
	"errors"
	"time"

	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WebsiteDiagnosticRepo struct {
	db *gorm.DB
}

func (r *WebsiteDiagnosticRepo) DeleteByWebsiteID(ctx context.Context, websiteID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, target := range []interface{}{
			&model.WebsiteDiagnosticTimeline{}, &model.WebsiteDiagnosticEvent{}, &model.WebsiteProbe{}, &model.WebsiteDiagnosticNonce{},
			&model.WebsiteIssue{}, &model.WebsiteDiagnosticSetting{},
		} {
			if err := tx.Where("website_id = ?", websiteID).Delete(target).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *WebsiteDiagnosticRepo) SaveHookSecret(websiteID uint, encrypted string) error {
	return r.db.Model(&model.WebsiteDiagnosticSetting{}).Where("website_id = ?", websiteID).Update("hook_secret_encrypted", encrypted).Error
}

func (r *WebsiteDiagnosticRepo) ClaimNonce(websiteID uint, nonce string, expiresAt time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", time.Now()).Delete(&model.WebsiteDiagnosticNonce{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.WebsiteDiagnosticNonce{WebsiteID: websiteID, Nonce: nonce, ExpiresAt: expiresAt}).Error
	})
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
			"code_project_id", "configured_by_user_id", "enabled", "caddy_monitoring", "active_probes", "backend_hook", "browser_hook",
			"auto_analysis", "monitor_http_4xx", "monitor_http_5xx", "monitor_upstream_errors", "monitor_slow_requests",
			"monitor_business_errors", "monitor_browser_errors", "monitor_resource_errors", "slow_request_threshold_ms",
			"trigger_count", "trigger_window_minutes", "retention_days", "default_executor_id", "approval_policy", "updated_at",
		}),
	}).Create(setting).Error
}

func (r *WebsiteDiagnosticRepo) ListEnabled() ([]model.WebsiteDiagnosticSetting, error) {
	var settings []model.WebsiteDiagnosticSetting
	err := r.db.Where("enabled = ?", true).Find(&settings).Error
	return settings, err
}

func (r *WebsiteDiagnosticRepo) ListConfigured() ([]model.WebsiteDiagnosticSetting, error) {
	var settings []model.WebsiteDiagnosticSetting
	err := r.db.Find(&settings).Error
	return settings, err
}

func (r *WebsiteDiagnosticRepo) IngestEvent(event *model.WebsiteDiagnosticEvent) (*model.WebsiteIssue, bool, error) {
	var issue model.WebsiteIssue
	created := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing model.WebsiteDiagnosticEvent
		err := tx.Where("website_id = ? AND event_id = ?", event.WebsiteID, event.EventID).First(&existing).Error
		if err == nil {
			return tx.First(&issue, existing.IssueID).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		err = tx.Where("website_id = ? AND fingerprint = ?", event.WebsiteID, event.Fingerprint).First(&issue).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			issue = model.WebsiteIssue{
				WebsiteID: event.WebsiteID, Fingerprint: event.Fingerprint, Status: "open",
				Severity: event.Severity, Title: event.Title, Kind: event.Kind, Route: event.Route,
				HTTPStatus: event.HTTPStatus, BusinessCode: event.BusinessCode, OccurrenceCount: 1,
				FirstRelease: event.Release, LatestRelease: event.Release,
				FirstSeenAt: event.OccurredAt, LastSeenAt: event.OccurredAt,
			}
			if event.SessionID != "" {
				issue.SessionCount = 1
			}
			if err = tx.Create(&issue).Error; err != nil {
				return err
			}
			created = true
		} else if err != nil {
			return err
		} else {
			updates := map[string]interface{}{
				"occurrence_count": gorm.Expr("occurrence_count + 1"), "last_seen_at": event.OccurredAt,
				"latest_release": event.Release, "severity": event.Severity, "updated_at": time.Now(),
			}
			if issue.Status == "resolved" && (issue.ResolvedAt == nil || event.OccurredAt.After(*issue.ResolvedAt)) {
				updates["status"] = "reopened"
				updates["resolved_at"] = nil
			}
			if issue.Status == "verifying" && event.Release != "" && event.Release == issue.VerifyRelease {
				updates["status"] = "reopened"
			}
			if err = tx.Model(&issue).Updates(updates).Error; err != nil {
				return err
			}
		}
		event.IssueID = issue.ID
		if err = tx.Create(event).Error; err != nil {
			return err
		}
		if event.SessionID != "" && !created {
			var count int64
			if err = tx.Model(&model.WebsiteDiagnosticEvent{}).
				Where("issue_id = ? AND session_key <> ''", issue.ID).
				Distinct("session_key").Count(&count).Error; err != nil {
				return err
			}
			if err = tx.Model(&issue).Update("session_count", count).Error; err != nil {
				return err
			}
		}
		return tx.First(&issue, issue.ID).Error
	})
	return &issue, created, err
}

func (r *WebsiteDiagnosticRepo) ListIssues(websiteID uint, status string, page, limit int) ([]model.WebsiteIssue, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	query := r.db.Model(&model.WebsiteIssue{}).Where("website_id = ?", websiteID)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var issues []model.WebsiteIssue
	err := query.Order("last_seen_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&issues).Error
	return issues, total, err
}

func (r *WebsiteDiagnosticRepo) GetIssue(websiteID, issueID uint) (*model.WebsiteIssue, error) {
	var issue model.WebsiteIssue
	err := r.db.Where("id = ? AND website_id = ?", issueID, websiteID).First(&issue).Error
	return &issue, err
}

func (r *WebsiteDiagnosticRepo) GetIssueEvidence(issueID uint, limit int) ([]model.WebsiteDiagnosticEvent, []model.WebsiteDiagnosticTimeline, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}
	var events []model.WebsiteDiagnosticEvent
	if err := r.db.Where("issue_id = ?", issueID).Order("occurred_at DESC").Limit(limit).Find(&events).Error; err != nil {
		return nil, nil, err
	}
	var timeline []model.WebsiteDiagnosticTimeline
	err := r.db.Where("issue_id = ?", issueID).Order("created_at DESC").Limit(100).Find(&timeline).Error
	return events, timeline, err
}

func (r *WebsiteDiagnosticRepo) UpdateIssue(issue *model.WebsiteIssue) error {
	return r.db.Save(issue).Error
}

func (r *WebsiteDiagnosticRepo) AddTimeline(entry *model.WebsiteDiagnosticTimeline) error {
	return r.db.Create(entry).Error
}

func (r *WebsiteDiagnosticRepo) SaveProbes(websiteID uint, probes []model.WebsiteProbe) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("website_id = ?", websiteID).Delete(&model.WebsiteProbe{}).Error; err != nil {
			return err
		}
		for index := range probes {
			probes[index].ID = 0
			probes[index].WebsiteID = websiteID
			if err := tx.Create(&probes[index]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *WebsiteDiagnosticRepo) ListProbes(websiteID uint) ([]model.WebsiteProbe, error) {
	var probes []model.WebsiteProbe
	err := r.db.Where("website_id = ?", websiteID).Order("id ASC").Find(&probes).Error
	return probes, err
}

func (r *WebsiteDiagnosticRepo) ListDueProbes(now time.Time) ([]model.WebsiteProbe, error) {
	var candidates []model.WebsiteProbe
	if err := r.db.Where("enabled = ?", true).Order("last_run_at ASC").Limit(500).Find(&candidates).Error; err != nil {
		return nil, err
	}
	probes := make([]model.WebsiteProbe, 0, min(len(candidates), 100))
	for _, probe := range candidates {
		if probe.LastRunAt == nil || probe.LastRunAt.Add(time.Duration(probe.IntervalSeconds)*time.Second).Before(now) {
			probes = append(probes, probe)
			if len(probes) == 100 {
				break
			}
		}
	}
	return probes, nil
}

func (r *WebsiteDiagnosticRepo) CountIssueSummary(websiteIDs []uint) (map[uint]map[string]int64, error) {
	type row struct {
		WebsiteID uint
		Status    string
		Count     int64
	}
	var rows []row
	if len(websiteIDs) == 0 {
		return map[uint]map[string]int64{}, nil
	}
	err := r.db.Model(&model.WebsiteIssue{}).Select("website_id, status, COUNT(*) AS count").
		Where("website_id IN ?", websiteIDs).Group("website_id, status").Scan(&rows).Error
	result := make(map[uint]map[string]int64)
	for _, item := range rows {
		if result[item.WebsiteID] == nil {
			result[item.WebsiteID] = make(map[string]int64)
		}
		result[item.WebsiteID][item.Status] = item.Count
	}
	return result, err
}
