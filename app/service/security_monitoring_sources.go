package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpc"
)

func collectWebsiteSecurityFindings(config model.SecurityMonitoringConfig) ([]securityFinding, error) {
	websites, err := repo.NewWebsite().ListBy()
	if err != nil {
		return nil, err
	}
	repository := repo.NewSecurityMonitoring()
	findings := make([]securityFinding, 0)
	for _, website := range websites {
		if !website.AccessLog {
			continue
		}
		cursor, cursorErr := repository.GetCursor("website", website.ID)
		if cursorErr != nil {
			return findings, cursorErr
		}
		lines, readErr := readSecurityLogBatch(websiteAccessLogPath(website.Alias), cursor, config.MaxBatchBytes, config.MaxBatchLines)
		if readErr != nil {
			return findings, readErr
		}
		entries := make([]*securityWebsiteLog, 0, len(lines))
		for _, line := range lines {
			entry, ok := parseSecurityWebsiteLog(line)
			if !ok {
				cursor.Malformed++
				continue
			}
			cursor.Processed++
			entries = append(entries, entry)
		}
		cursor.LastScannedAt = time.Now()
		if len(entries) > 0 {
			cursor.LastEventAt = time.Now()
			findings = append(findings, detectWebsiteSecurityFindings(website, entries, config)...)
		}
		if saveErr := repository.SaveCursor(cursor); saveErr != nil {
			return findings, saveErr
		}
	}
	return findings, nil
}

func collectSSHSecurityFindings(ctx context.Context, config model.SecurityMonitoringConfig) ([]securityFinding, error) {
	repository := repo.NewSecurityMonitoring()
	cursor, err := repository.GetCursor("ssh", 0)
	if err != nil {
		return nil, err
	}
	response, err := gpc.Do(ctx, "SSH_LOGIN_LOG_LIST", map[string]interface{}{"page": 1, "limit": 100})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unsupported platform") {
			return []securityFinding{}, nil
		}
		return nil, err
	}
	var result dto.SSHLoginLogResult
	if err := json.Unmarshal([]byte(response.Output), &result); err != nil {
		return nil, err
	}
	items := append([]dto.SSHLoginLog(nil), result.Items...)
	sort.Slice(items, func(i, j int) bool {
		return parseSecurityTime(items[i].CreatedAt).Before(parseSecurityTime(items[j].CreatedAt))
	})
	windowStart := cursor.LastEventAt
	if windowStart.IsZero() {
		windowStart = time.Now().Add(-5 * time.Minute)
	}
	failedByIP := make(map[string][]dto.SSHLoginLog)
	failedByUser := make(map[string][]dto.SSHLoginLog)
	findings := make([]securityFinding, 0)
	knownBaseline, _ := repository.HasKnownLoginSource("ssh")
	latest := cursor.LastEventAt
	for _, item := range items {
		seenAt := parseSecurityTime(item.CreatedAt)
		if seenAt.IsZero() || !seenAt.After(windowStart) {
			continue
		}
		if seenAt.After(latest) {
			latest = seenAt
		}
		if strings.EqualFold(item.Status, "Failed") {
			failedByIP[item.SourceIP] = append(failedByIP[item.SourceIP], item)
			failedByUser[item.Username] = append(failedByUser[item.Username], item)
			continue
		}
		if !strings.EqualFold(item.Status, "Success") {
			continue
		}
		known, knownErr := repository.IsKnownLoginSource("ssh", item.Username, item.SourceIP)
		if knownErr != nil {
			return findings, knownErr
		}
		if knownBaseline && !known {
			findings = append(findings, sshLoginFinding("ssh_new_source", "medium", item,
				fmt.Sprintf("SSH 账号 %s 从新 IP %s 登录成功", item.Username, item.SourceIP), 1, nil, seenAt))
		}
		if strings.EqualFold(item.Username, "root") {
			findings = append(findings, sshLoginFinding("ssh_root_login", "medium", item,
				fmt.Sprintf("检测到 root 从 %s 登录 SSH", item.SourceIP), 1, nil, seenAt))
		}
		if seenAt.Hour() < 6 {
			findings = append(findings, sshLoginFinding("ssh_unusual_hour_login", "medium", item,
				fmt.Sprintf("SSH 账号 %s 在异常时段从 %s 登录", item.Username, item.SourceIP), 1, nil, seenAt))
		}
		if failed := failedByIP[item.SourceIP]; len(failed) > 0 {
			samples := make([]string, 0, len(failed)+1)
			for _, failure := range failed {
				samples = append(samples, failure.Raw)
			}
			samples = append(samples, item.Raw)
			findings = append(findings, sshLoginFinding("ssh_failure_then_success", "high", item,
				fmt.Sprintf("%s 在 %d 次失败后成功登录 SSH", item.SourceIP, len(failed)), len(failed), samples, seenAt))
		}
		if err := repository.RememberLoginSource("ssh", item.Username, item.SourceIP, seenAt); err != nil {
			return findings, err
		}
	}
	for ip, failures := range failedByIP {
		if len(failures) < config.SSHFailurePerMinute {
			continue
		}
		samples := make([]string, 0, len(failures))
		for _, item := range failures {
			samples = append(samples, item.Raw)
		}
		last := failures[len(failures)-1]
		findings = append(findings, sshLoginFinding("ssh_brute_force", "high", last,
			fmt.Sprintf("检测到来自 %s 的 SSH 暴力破解（%d 次失败）", ip, len(failures)), len(failures), samples, parseSecurityTime(last.CreatedAt)))
	}
	for username, failures := range failedByUser {
		ips := make(map[string]struct{})
		for _, failure := range failures {
			ips[failure.SourceIP] = struct{}{}
		}
		if len(ips) < 5 || len(failures) < config.SSHFailurePerMinute {
			continue
		}
		last := failures[len(failures)-1]
		findings = append(findings, sshLoginFinding("ssh_distributed_brute_force", "high", last,
			fmt.Sprintf("SSH 账号 %s 遭到来自 %d 个 IP 的分布式爆破", username, len(ips)), len(failures), sshFailureSamples(failures), parseSecurityTime(last.CreatedAt)))
	}
	for ip, failures := range failedByIP {
		users := make(map[string]struct{})
		for _, failure := range failures {
			users[failure.Username] = struct{}{}
		}
		if len(users) < 3 {
			continue
		}
		last := failures[len(failures)-1]
		findings = append(findings, sshLoginFinding("ssh_account_enumeration", "high", last,
			fmt.Sprintf("检测到 %s 枚举 %d 个 SSH 账号", ip, len(users)), len(failures), sshFailureSamples(failures), parseSecurityTime(last.CreatedAt)))
	}
	cursor.LastEventAt, cursor.LastScannedAt = latest, time.Now()
	cursor.Processed += int64(len(items))
	if err := repository.SaveCursor(cursor); err != nil {
		return findings, err
	}
	return findings, nil
}

func sshFailureSamples(items []dto.SSHLoginLog) []string {
	samples := make([]string, 0, len(items))
	for _, item := range items {
		samples = append(samples, item.Raw)
	}
	return samples
}

func sshLoginFinding(eventType, level string, item dto.SSHLoginLog, summary string, count int, samples []string, seenAt time.Time) securityFinding {
	if len(samples) == 0 {
		samples = []string{item.Raw}
	}
	return securityFinding{
		SourceType: "ssh", SourceName: "SSH", EventType: eventType, Level: level,
		Actor: item.SourceIP, Summary: summary, Value: float64(count), SeenAt: seenAt,
		Evidence: []securityEvidence{{Source: "ssh", Description: summary, Count: count, Samples: boundedSecuritySamples(samples, 5)}},
	}
}

func collectPanelSecurityFindings(config model.SecurityMonitoringConfig) ([]securityFinding, error) {
	repository := repo.NewSecurityMonitoring()
	cursor, err := repository.GetCursor("panel", 0)
	if err != nil {
		return nil, err
	}
	windowStart := cursor.LastEventAt
	if windowStart.IsZero() {
		windowStart = time.Now().Add(-5 * time.Minute)
	}
	var logs []model.LoginLog
	if err := global.DB.Where("created_at > ?", windowStart).Order("created_at asc").Limit(config.MaxBatchLines).Find(&logs).Error; err != nil {
		return nil, err
	}
	failedByIP := make(map[string][]model.LoginLog)
	failedOperationsByIP := make(map[string][]model.OperationLog)
	latest := cursor.LastEventAt
	for _, item := range logs {
		if item.CreatedAt.After(latest) {
			latest = item.CreatedAt
		}
		if strings.EqualFold(item.Status, "Failed") || strings.EqualFold(item.Status, "failed") {
			failedByIP[item.IP] = append(failedByIP[item.IP], item)
		}
	}
	var operations []model.OperationLog
	if err := global.DB.Where("created_at > ? AND status = ?", windowStart, "Failed").Order("created_at asc").Limit(config.MaxBatchLines).Find(&operations).Error; err != nil {
		return nil, err
	}
	for _, item := range operations {
		failedOperationsByIP[item.IP] = append(failedOperationsByIP[item.IP], item)
		if item.CreatedAt.After(latest) {
			latest = item.CreatedAt
		}
	}
	findings := make([]securityFinding, 0)
	for ip, failures := range failedByIP {
		if len(failures) < config.LoginFailurePerMinute {
			continue
		}
		samples := make([]string, 0, len(failures))
		for _, item := range failures {
			samples = append(samples, fmt.Sprintf("%s panel login failed from %s: %s", item.CreatedAt.Format(time.RFC3339), item.IP, item.Message))
		}
		findings = append(findings, securityFinding{
			SourceType: "panel", SourceName: "GoPanel", EventType: "panel_brute_force", Level: "high", Actor: ip,
			Summary: fmt.Sprintf("检测到来自 %s 的面板登录爆破（%d 次失败）", ip, len(failures)), Value: float64(len(failures)), SeenAt: failures[len(failures)-1].CreatedAt,
			Evidence: []securityEvidence{{Source: "panel", Description: "面板登录失败", Count: len(failures), Samples: boundedSecuritySamples(samples, 5)}},
		})
	}
	for ip, operations := range failedOperationsByIP {
		if len(operations) < config.LoginFailurePerMinute {
			continue
		}
		samples := make([]string, 0, len(operations))
		for _, item := range operations {
			samples = append(samples, fmt.Sprintf("%s %s %s", item.Method, item.Path, item.Message))
		}
		findings = append(findings, securityFinding{
			SourceType: "panel", SourceName: "GoPanel", EventType: "panel_admin_operation_anomaly", Level: "high", Actor: ip,
			Summary: fmt.Sprintf("检测到来自 %s 的异常管理员操作（%d 次失败）", ip, len(operations)), Value: float64(len(operations)), SeenAt: operations[len(operations)-1].CreatedAt,
			Evidence: []securityEvidence{{Source: "panel", Description: "失败管理员操作", Count: len(operations), Samples: boundedSecuritySamples(samples, 5)}},
		})
	}
	cursor.LastEventAt, cursor.LastScannedAt = latest, time.Now()
	cursor.Processed += int64(len(logs) + len(operations))
	if err := repository.SaveCursor(cursor); err != nil {
		return findings, err
	}
	return findings, nil
}

func parseSecurityTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(layout, strings.TrimSpace(value), time.Local); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
