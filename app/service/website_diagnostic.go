package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
)

const (
	websiteDiagnosticDefaultExecutor = "codex"
	websiteDiagnosticDefaultPolicy   = "safe_auto"
)

type WebsiteDiagnosticSettingView struct {
	model.WebsiteDiagnosticSetting
	Configured           bool   `json:"configured"`
	TrackingDir          string `json:"trackingDir"`
	HookSecretConfigured bool   `json:"hookSecretConfigured"`
	RemoteEndpoint       string `json:"remoteEndpoint"`
}

func defaultWebsiteDiagnosticSetting(websiteID uint) model.WebsiteDiagnosticSetting {
	return model.WebsiteDiagnosticSetting{
		WebsiteID: websiteID, CaddyMonitoring: true,
		MonitorHTTP4xx: true, MonitorHTTP5xx: true, MonitorUpstreamErrors: true, MonitorSlowRequests: true,
		MonitorBusinessErrors: true, MonitorBrowserErrors: true, MonitorResourceErrors: true,
		SlowRequestThresholdMS: 1500, TriggerCount: 5, TriggerWindowMinutes: 10, RetentionDays: 7,
		DefaultExecutorID: websiteDiagnosticDefaultExecutor, ApprovalPolicy: websiteDiagnosticDefaultPolicy,
	}
}

func websiteTrackingDir(alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" || alias == "." || alias == ".." || strings.ContainsAny(alias, `/\\`) {
		return "", buserr.New("ErrWebsiteDiagnosticInvalidDirectory")
	}
	root := filepath.Join(global.CONF.System.BaseDir, "log", "website")
	dir := filepath.Join(root, alias, "tracking")
	relative, err := filepath.Rel(root, dir)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", buserr.New("ErrWebsiteDiagnosticInvalidDirectory")
	}
	return dir, nil
}

func ensureWebsiteTrackingDirs(alias string) (string, error) {
	dir, err := websiteTrackingDir(alias)
	if err != nil {
		return "", err
	}
	fileOp := files.NewFileOp()
	for _, name := range []string{"inbox", "processing", "processed", "rejected", "attachments"} {
		if err := fileOp.CreateDir(filepath.Join(dir, name), 0750); err != nil {
			return "", buserr.WithDetail("ErrWebsiteDiagnosticCreateDirectory", err.Error(), err)
		}
	}
	return dir, nil
}

func normalizeWebsiteDiagnosticSetting(setting *model.WebsiteDiagnosticSetting) error {
	if setting == nil {
		return buserr.New("ErrWebsiteDiagnosticEmptySetting")
	}
	setting.DefaultExecutorID = strings.TrimSpace(setting.DefaultExecutorID)
	if setting.DefaultExecutorID == "" {
		setting.DefaultExecutorID = websiteDiagnosticDefaultExecutor
	}
	validExecutors := map[string]bool{"codex": true, "grok": true, "claude": true, "opencode": true, "aider": true}
	if !validExecutors[setting.DefaultExecutorID] {
		return buserr.New("ErrWebsiteDiagnosticInvalidExecutor")
	}
	setting.ApprovalPolicy = strings.TrimSpace(setting.ApprovalPolicy)
	if setting.ApprovalPolicy == "" {
		setting.ApprovalPolicy = websiteDiagnosticDefaultPolicy
	}
	validPolicies := map[string]map[string]bool{
		"codex":    {"manual": true, "safe_auto": true, "full_auto": true},
		"grok":     {"manual": true, "safe_auto": true, "full_auto": true},
		"claude":   {"manual": true, "safe_auto": true, "full_auto": true},
		"opencode": {"full_auto": true},
		"aider":    {"full_auto": true},
	}
	if !validPolicies[setting.DefaultExecutorID][setting.ApprovalPolicy] {
		return buserr.New("ErrWebsiteDiagnosticInvalidApprovalPolicy")
	}
	if setting.SlowRequestThresholdMS < 100 || setting.SlowRequestThresholdMS > 120000 {
		return buserr.New("ErrWebsiteDiagnosticSlowThreshold")
	}
	if setting.TriggerCount < 1 || setting.TriggerCount > 10000 {
		return buserr.New("ErrWebsiteDiagnosticTriggerCount")
	}
	if setting.TriggerWindowMinutes < 1 || setting.TriggerWindowMinutes > 1440 {
		return buserr.New("ErrWebsiteDiagnosticTriggerWindow")
	}
	if setting.RetentionDays < 1 || setting.RetentionDays > 365 {
		return buserr.New("ErrWebsiteDiagnosticRetentionDays")
	}
	if setting.Enabled && !setting.CaddyMonitoring && !setting.ActiveProbes && !setting.BackendHook && !setting.BrowserHook {
		return buserr.New("ErrWebsiteDiagnosticSourceRequired")
	}
	if setting.Enabled && !setting.MonitorHTTP4xx && !setting.MonitorHTTP5xx && !setting.MonitorUpstreamErrors &&
		!setting.MonitorSlowRequests && !setting.MonitorBusinessErrors && !setting.MonitorBrowserErrors && !setting.MonitorResourceErrors {
		return buserr.New("ErrWebsiteDiagnosticContentRequired")
	}
	if setting.AutoAnalysis && setting.CodeProjectID == 0 {
		return buserr.New("ErrWebsiteDiagnosticProjectRequired")
	}
	return nil
}

func (s *WebsiteService) GetDiagnosticSetting(websiteID uint) (*WebsiteDiagnosticSettingView, error) {
	website, err := s.repo.GetFirst(s.repo.WithID(websiteID))
	if err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticWebsiteNotFound")
	}
	setting, err := repo.NewWebsiteDiagnostic(global.DB).GetByWebsiteID(websiteID)
	if err != nil {
		return nil, err
	}
	configured := setting != nil
	if setting == nil {
		defaults := defaultWebsiteDiagnosticSetting(websiteID)
		setting = &defaults
	}
	trackingDir, _ := websiteTrackingDir(website.Alias)
	return &WebsiteDiagnosticSettingView{
		WebsiteDiagnosticSetting: *setting, Configured: configured, TrackingDir: trackingDir,
		HookSecretConfigured: setting.HookSecretEncrypted != "", RemoteEndpoint: "/api/website-diagnostics/" + fmt.Sprint(websiteID) + "/events",
	}, nil
}

func (s *WebsiteService) SaveDiagnosticSetting(setting *model.WebsiteDiagnosticSetting, userID uint, includeAllProjects bool) (*WebsiteDiagnosticSettingView, error) {
	if setting == nil {
		return nil, buserr.New("ErrWebsiteDiagnosticEmptySetting")
	}
	website, err := s.repo.GetFirst(s.repo.WithID(setting.WebsiteID))
	if err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticWebsiteNotFound")
	}
	if err := normalizeWebsiteDiagnosticSetting(setting); err != nil {
		return nil, err
	}
	if setting.CodeProjectID > 0 {
		project, projectErr := repo.NewAIProjectRepo().GetProjectByID(setting.CodeProjectID)
		if projectErr != nil {
			return nil, buserr.New("ErrWebsiteDiagnosticProjectNotFound")
		}
		if !includeAllProjects && project.CreatorID != userID {
			return nil, buserr.New("ErrWebsiteDiagnosticProjectForbidden")
		}
	}
	setting.ConfiguredByUserID = userID
	if existing, loadErr := repo.NewWebsiteDiagnostic(global.DB).GetByWebsiteID(setting.WebsiteID); loadErr == nil && existing != nil {
		setting.HookSecretEncrypted = existing.HookSecretEncrypted
	}
	if setting.BackendHook || setting.BrowserHook {
		if _, err := ensureWebsiteTrackingDirs(website.Alias); err != nil {
			return nil, err
		}
	}
	if err := repo.NewWebsiteDiagnostic(global.DB).Save(setting); err != nil {
		return nil, err
	}
	return s.GetDiagnosticSetting(setting.WebsiteID)
}

func loadWebsiteDiagnosticSummaries(websiteIDs []uint) (map[uint]response.WebsiteDiagnosticSummary, error) {
	repository := repo.NewWebsiteDiagnostic(global.DB)
	settings, err := repository.ListByWebsiteIDs(websiteIDs)
	if err != nil {
		return nil, err
	}
	counts, err := repository.CountIssueSummary(websiteIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uint]response.WebsiteDiagnosticSummary, len(settings))
	for _, setting := range settings {
		summary := response.WebsiteDiagnosticSummary{
			Configured: true, Enabled: setting.Enabled, SourceCount: websiteDiagnosticSourceCount(setting),
			ContentCount: websiteDiagnosticContentCount(setting), CodeProjectID: setting.CodeProjectID, AutoAnalysis: setting.AutoAnalysis,
		}
		summary.OpenCount = counts[setting.WebsiteID]["open"] + counts[setting.WebsiteID]["confirmed"]
		summary.ReopenedCount = counts[setting.WebsiteID]["reopened"]
		summary.ProcessingCount = counts[setting.WebsiteID]["code_processing"] + counts[setting.WebsiteID]["fix_ready"] + counts[setting.WebsiteID]["verifying"]
		result[setting.WebsiteID] = summary
	}
	return result, nil
}

func websiteDiagnosticSourceCount(setting model.WebsiteDiagnosticSetting) int {
	count := 0
	for _, enabled := range []bool{setting.CaddyMonitoring, setting.ActiveProbes, setting.BackendHook, setting.BrowserHook} {
		if enabled {
			count++
		}
	}
	return count
}

func websiteDiagnosticContentCount(setting model.WebsiteDiagnosticSetting) int {
	count := 0
	for _, enabled := range []bool{
		setting.MonitorHTTP4xx, setting.MonitorHTTP5xx, setting.MonitorUpstreamErrors, setting.MonitorSlowRequests,
		setting.MonitorBusinessErrors, setting.MonitorBrowserErrors, setting.MonitorResourceErrors,
	} {
		if enabled {
			count++
		}
	}
	return count
}
