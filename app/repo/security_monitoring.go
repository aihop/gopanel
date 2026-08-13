package repo

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type SecurityMonitoringRepo struct{ db *gorm.DB }

func NewSecurityMonitoring() *SecurityMonitoringRepo {
	return &SecurityMonitoringRepo{db: global.DB}
}

func (r *SecurityMonitoringRepo) MigrateTable() error {
	return r.db.AutoMigrate(
		&model.SecurityMonitoringConfig{},
		&model.SecurityLogCursor{},
		&model.SecurityEvent{},
		&model.SecurityAnalysisRun{},
		&model.SecurityKnownLoginSource{},
	)
}

func defaultSecurityMonitoringConfig() model.SecurityMonitoringConfig {
	return model.SecurityMonitoringConfig{
		Enabled: true, WebsiteEnabled: true, SSHEnabled: true, PanelEnabled: true,
		AIIntervalMinutes: 15, AIDailyTokenBudget: 50000,
		MaxBatchBytes: 2 << 20, MaxBatchLines: 10000,
		RequestPerMinute: 120, NotFoundPerMinute: 30, ServerErrorPerMinute: 20,
		LoginFailurePerMinute: 10, SSHFailurePerMinute: 10,
		DebounceTimes: 2, ResolveAfterMinutes: 10,
	}
}

func (r *SecurityMonitoringRepo) GetConfig() (model.SecurityMonitoringConfig, error) {
	var config model.SecurityMonitoringConfig
	err := r.db.First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return defaultSecurityMonitoringConfig(), nil
	}
	return config, err
}

func (r *SecurityMonitoringRepo) SaveConfig(config *model.SecurityMonitoringConfig) error {
	var existing model.SecurityMonitoringConfig
	if err := r.db.First(&existing).Error; err == nil {
		config.ID, config.CreatedAt = existing.ID, existing.CreatedAt
	}
	return r.db.Select("*").Save(config).Error
}

func (r *SecurityMonitoringRepo) GetCursor(sourceType string, sourceID uint) (*model.SecurityLogCursor, error) {
	var cursor model.SecurityLogCursor
	err := r.db.Where("source_type = ? AND source_id = ?", sourceType, sourceID).First(&cursor).Error
	if err == gorm.ErrRecordNotFound {
		return &model.SecurityLogCursor{SourceType: sourceType, SourceID: sourceID}, nil
	}
	return &cursor, err
}

func (r *SecurityMonitoringRepo) SaveCursor(cursor *model.SecurityLogCursor) error {
	return r.db.Save(cursor).Error
}

func (r *SecurityMonitoringRepo) UpsertEvent(candidate *model.SecurityEvent, debounce int) (*model.SecurityEvent, bool, error) {
	var event model.SecurityEvent
	err := r.db.Where("fingerprint = ?", candidate.Fingerprint).First(&event).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, false, err
	}
	if err == gorm.ErrRecordNotFound {
		event = *candidate
		event.Status = model.SecurityEventPending
		event.AnalysisStatus = model.SecurityAnalysisPending
		event.HitCount = 0
	}
	wasFiring := event.Status == model.SecurityEventFiring
	if event.Status == model.SecurityEventResolved {
		event.Status = model.SecurityEventPending
		event.FirstSeenAt = candidate.FirstSeenAt
		event.HitCount = 0
		event.ResolvedAt = nil
		event.AnalysisStatus = model.SecurityAnalysisPending
	}
	event.SourceName, event.Level = candidate.SourceName, candidate.Level
	event.Summary, event.Evidence, event.Value = candidate.Summary, candidate.Evidence, candidate.Value
	event.LastSeenAt = candidate.LastSeenAt
	event.HitCount++
	if debounce < 1 {
		debounce = 1
	}
	if event.Level == "critical" || event.Level == "high" {
		debounce = 1
	}
	if event.Status == model.SecurityEventPending && event.HitCount >= debounce {
		event.Status = model.SecurityEventFiring
	}
	if err := r.db.Save(&event).Error; err != nil {
		return nil, false, err
	}
	return &event, !wasFiring && event.Status == model.SecurityEventFiring, nil
}

func (r *SecurityMonitoringRepo) ResolveStale(before time.Time) ([]model.SecurityEvent, error) {
	var events []model.SecurityEvent
	if err := r.db.Where("status <> ? AND last_seen_at < ?", model.SecurityEventResolved, before).Find(&events).Error; err != nil {
		return nil, err
	}
	resolved := make([]model.SecurityEvent, 0, len(events))
	now := time.Now()
	for index := range events {
		wasFiring := events[index].Status == model.SecurityEventFiring
		events[index].Status, events[index].ResolvedAt = model.SecurityEventResolved, &now
		if err := r.db.Save(&events[index]).Error; err != nil {
			return resolved, err
		}
		if wasFiring {
			resolved = append(resolved, events[index])
		}
	}
	return resolved, nil
}

func (r *SecurityMonitoringRepo) PageEvents(page, limit int, status, level, sourceType string) (int64, []model.SecurityEvent, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	db := r.db.Model(&model.SecurityEvent{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if level != "" {
		db = db.Where("level = ?", level)
	}
	if sourceType != "" {
		db = db.Where("source_type = ?", sourceType)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var events []model.SecurityEvent
	err := db.Order("CASE level WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, last_seen_at desc").
		Offset((page - 1) * limit).Limit(limit).Find(&events).Error
	return total, events, err
}

func (r *SecurityMonitoringRepo) ActiveSummary(limit int) ([]model.SecurityEvent, error) {
	if limit < 1 || limit > 20 {
		limit = 5
	}
	var events []model.SecurityEvent
	err := r.db.Where("status = ?", model.SecurityEventFiring).
		Order("CASE level WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, last_seen_at desc").
		Limit(limit).Find(&events).Error
	return events, err
}

func (r *SecurityMonitoringRepo) PendingAnalysis(limit int) ([]model.SecurityEvent, error) {
	var events []model.SecurityEvent
	err := r.db.Where("status = ? AND analysis_status IN ?", model.SecurityEventFiring,
		[]string{model.SecurityAnalysisPending, model.SecurityAnalysisFailed}).
		Order("last_seen_at asc").Limit(limit).Find(&events).Error
	return events, err
}

func (r *SecurityMonitoringRepo) DueAnalysis(limit, retryMinutes int) ([]model.SecurityEvent, error) {
	if retryMinutes < 5 {
		retryMinutes = 15
	}
	var events []model.SecurityEvent
	err := r.db.Where(
		"status = ? AND (analysis_status = ? OR (analysis_status = ? AND analyzed_at < ?))",
		model.SecurityEventFiring, model.SecurityAnalysisPending, model.SecurityAnalysisFailed,
		time.Now().Add(-time.Duration(retryMinutes)*time.Minute),
	).Order("last_seen_at asc").Limit(limit).Find(&events).Error
	return events, err
}

func (r *SecurityMonitoringRepo) SaveEvent(event *model.SecurityEvent) error {
	return r.db.Save(event).Error
}

func (r *SecurityMonitoringRepo) GetEvent(id uint) (*model.SecurityEvent, error) {
	var event model.SecurityEvent
	if err := r.db.First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *SecurityMonitoringRepo) EventsNeedingNotification(before time.Time, limit int) ([]model.SecurityEvent, error) {
	var events []model.SecurityEvent
	err := r.db.Where(
		"status = ? AND (notify_status = ? OR last_notified_at IS NULL OR last_notified_at < ?)",
		model.SecurityEventFiring, "failed", before,
	).Order("last_seen_at asc").Limit(limit).Find(&events).Error
	return events, err
}

func (r *SecurityMonitoringRepo) CreateAnalysisRun(run *model.SecurityAnalysisRun) error {
	return r.db.Create(run).Error
}

func (r *SecurityMonitoringRepo) SaveAnalysisRun(run *model.SecurityAnalysisRun) error {
	return r.db.Save(run).Error
}

func (r *SecurityMonitoringRepo) TokensUsedSince(since time.Time) (int64, error) {
	var total int64
	err := r.db.Model(&model.SecurityAnalysisRun{}).Where("created_at >= ?", since).Select("COALESCE(SUM(total_tokens), 0)").Scan(&total).Error
	return total, err
}

func (r *SecurityMonitoringRepo) HasKnownLoginSource(sourceType string) (bool, error) {
	var count int64
	err := r.db.Model(&model.SecurityKnownLoginSource{}).Where("source_type = ?", sourceType).Count(&count).Error
	return count > 0, err
}

func (r *SecurityMonitoringRepo) IsKnownLoginSource(sourceType, username, ip string) (bool, error) {
	var count int64
	err := r.db.Model(&model.SecurityKnownLoginSource{}).
		Where("source_type = ? AND username = ? AND ip = ?", sourceType, username, ip).Count(&count).Error
	return count > 0, err
}

func (r *SecurityMonitoringRepo) RememberLoginSource(sourceType, username, ip string, seenAt time.Time) error {
	var source model.SecurityKnownLoginSource
	err := r.db.Where("source_type = ? AND username = ? AND ip = ?", sourceType, username, ip).First(&source).Error
	if err == gorm.ErrRecordNotFound {
		source = model.SecurityKnownLoginSource{SourceType: sourceType, Username: username, IP: ip, FirstSeenAt: seenAt}
	} else if err != nil {
		return err
	}
	source.LastSeenAt = seenAt
	return r.db.Save(&source).Error
}
