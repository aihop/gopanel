package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

type securityWebsiteLog struct {
	Level   string  `json:"level"`
	TS      float64 `json:"ts"`
	Request struct {
		RemoteIP string              `json:"remote_ip"`
		ClientIP string              `json:"client_ip"`
		Method   string              `json:"method"`
		Host     string              `json:"host"`
		URI      string              `json:"uri"`
		Headers  map[string][]string `json:"headers"`
	} `json:"request"`
	Status   int     `json:"status"`
	Duration float64 `json:"duration"`
}

type websiteActorStats struct {
	Requests      int
	NotFound      int
	LoginFailures int
	MaliciousUA   int
	Samples       []string
}

var securityWebsitePatterns = []struct {
	EventType string
	Pattern   *regexp.Regexp
	Label     string
}{
	{"sqli", regexp.MustCompile(`(?i)(union(?:\s|%20)+select|information_schema|sleep\s*\(|waitfor(?:\s|%20)+delay|(?:'|%27)(?:\s|%20)*or(?:\s|%20)+1=1)`), "SQL 注入探测"},
	{"xss", regexp.MustCompile(`(?i)(<script|%3cscript|javascript:|onerror(?:=|%3d)|onload(?:=|%3d))`), "XSS 探测"},
	{"path_traversal", regexp.MustCompile(`(?i)(\.\./|%2e%2e(?:%2f|/)|/etc/passwd|/proc/self/environ)`), "路径穿越探测"},
	{"sensitive_path", regexp.MustCompile(`(?i)(^|/)(\.env|\.git|\.svn|wp-config\.php|config\.(?:yml|yaml|json)|backup|dump\.sql|phpinfo\.php)(?:$|[/?])`), "敏感文件扫描"},
}

var (
	securityLoginPathPattern   = regexp.MustCompile(`(?i)(/login|/signin|/auth|/session|/wp-login\.php)(?:$|[/?])`)
	securityMaliciousUAPattern = regexp.MustCompile(`(?i)(sqlmap|nikto|dirbuster|gobuster|nmap|masscan|acunetix|nessus|wpscan|zgrab|python-requests|go-http-client)`)
)

func parseSecurityWebsiteLog(line string) (*securityWebsiteLog, bool) {
	jsonStart := strings.Index(line, "{")
	if jsonStart < 0 {
		return nil, false
	}
	var entry securityWebsiteLog
	if err := json.Unmarshal([]byte(line[jsonStart:]), &entry); err != nil {
		return nil, false
	}
	if entry.Request.URI == "" && entry.Status == 0 {
		return nil, false
	}
	return &entry, true
}

func detectWebsiteSecurityFindings(website model.Website, entries []*securityWebsiteLog, config model.SecurityMonitoringConfig) []securityFinding {
	windows := make(map[time.Time][]*securityWebsiteLog)
	cutoff := time.Now().Add(-5 * time.Minute)
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		seenAt := time.Now()
		if entry.TS > 0 {
			seenAt = time.Unix(0, int64(entry.TS*float64(time.Second)))
		}
		if seenAt.Before(cutoff) {
			continue
		}
		window := seenAt.Truncate(time.Minute)
		windows[window] = append(windows[window], entry)
	}
	windowStarts := make([]time.Time, 0, len(windows))
	for window := range windows {
		windowStarts = append(windowStarts, window)
	}
	sort.Slice(windowStarts, func(i, j int) bool { return windowStarts[i].Before(windowStarts[j]) })
	findings := make([]securityFinding, 0)
	for _, window := range windowStarts {
		findings = append(findings, detectWebsiteSecurityWindow(website, windows[window], config, window.Add(time.Minute))...)
	}
	return findings
}

func detectWebsiteSecurityWindow(website model.Website, entries []*securityWebsiteLog, config model.SecurityMonitoringConfig, windowEnd time.Time) []securityFinding {
	actors := make(map[string]*websiteActorStats)
	serverErrors := 0
	serverErrorSamples := make([]string, 0, 5)
	patternActors := make(map[string]map[string][]string)
	latest := windowEnd
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		ip := strings.TrimSpace(entry.Request.ClientIP)
		if ip == "" {
			ip = strings.TrimSpace(entry.Request.RemoteIP)
		}
		if ip == "" {
			ip = "unknown"
		}
		stats := actors[ip]
		if stats == nil {
			stats = &websiteActorStats{}
			actors[ip] = stats
		}
		stats.Requests++
		sample := fmt.Sprintf("%s %s %d from %s", entry.Request.Method, scrubWebsiteSecurityURI(entry.Request.URI), entry.Status, ip)
		if len(stats.Samples) < 5 {
			stats.Samples = append(stats.Samples, sample)
		}
		if entry.Status == 404 {
			stats.NotFound++
		}
		if securityLoginPathPattern.MatchString(entry.Request.URI) && (entry.Status == 401 || entry.Status == 403 || entry.Status == 429) {
			stats.LoginFailures++
		}
		userAgent := strings.Join(entry.Request.Headers["User-Agent"], " ")
		if userAgent == "" {
			userAgent = strings.Join(entry.Request.Headers["user-agent"], " ")
		}
		if securityMaliciousUAPattern.MatchString(userAgent) {
			stats.MaliciousUA++
		}
		if entry.Status >= 500 {
			serverErrors++
			if len(serverErrorSamples) < 5 {
				serverErrorSamples = append(serverErrorSamples, sample)
			}
		}
		for _, rule := range securityWebsitePatterns {
			if !rule.Pattern.MatchString(entry.Request.URI) {
				continue
			}
			if patternActors[rule.EventType] == nil {
				patternActors[rule.EventType] = make(map[string][]string)
			}
			patternActors[rule.EventType][ip] = append(patternActors[rule.EventType][ip], sample)
		}
	}
	findings := make([]securityFinding, 0)
	for _, rule := range securityWebsitePatterns {
		distinctActors := len(patternActors[rule.EventType])
		totalMatches := 0
		for ip, samples := range patternActors[rule.EventType] {
			totalMatches += len(samples)
			findings = append(findings, securityFinding{
				SourceType: "website", SourceID: website.ID, SourceName: website.PrimaryDomain,
				EventType: rule.EventType, Level: "high", Actor: ip,
				Summary: fmt.Sprintf("%s 检测到来自 %s 的%s", website.PrimaryDomain, ip, rule.Label),
				Value:   float64(len(samples)), SeenAt: latest,
				Evidence: []securityEvidence{{Source: "website", Description: rule.Label, Count: len(samples), Samples: boundedSecuritySamples(samples, 5)}},
			})
		}
		if distinctActors >= 5 {
			findings = append(findings, websiteThresholdFinding(website, "distributed_scan", "high", "distributed",
				fmt.Sprintf("%s 检测到 %d 个来源协同进行%s", website.PrimaryDomain, distinctActors, rule.Label), totalMatches, nil, latest))
		}
	}
	actorIPs := make([]string, 0, len(actors))
	for ip := range actors {
		actorIPs = append(actorIPs, ip)
	}
	sort.Strings(actorIPs)
	for _, ip := range actorIPs {
		stats := actors[ip]
		if stats.Requests >= config.RequestPerMinute {
			findings = append(findings, websiteThresholdFinding(website, "request_flood", "medium", ip,
				fmt.Sprintf("%s 在单个采集窗口收到来自 %s 的 %d 次请求", website.PrimaryDomain, ip, stats.Requests), stats.Requests, stats.Samples, latest))
		}
		if stats.NotFound >= config.NotFoundPerMinute {
			findings = append(findings, websiteThresholdFinding(website, "not_found_scan", "medium", ip,
				fmt.Sprintf("%s 检测到来自 %s 的连续路径扫描（%d 次 404）", website.PrimaryDomain, ip, stats.NotFound), stats.NotFound, stats.Samples, latest))
		}
		if stats.LoginFailures >= config.LoginFailurePerMinute {
			findings = append(findings, websiteThresholdFinding(website, "website_login_brute_force", "high", ip,
				fmt.Sprintf("%s 检测到来自 %s 的登录接口爆破（%d 次失败）", website.PrimaryDomain, ip, stats.LoginFailures), stats.LoginFailures, stats.Samples, latest))
		}
		if stats.MaliciousUA > 0 {
			findings = append(findings, websiteThresholdFinding(website, "malicious_user_agent", "medium", ip,
				fmt.Sprintf("%s 检测到来自 %s 的恶意扫描工具 User-Agent", website.PrimaryDomain, ip), stats.MaliciousUA, stats.Samples, latest))
		}
	}
	if serverErrors >= config.ServerErrorPerMinute {
		findings = append(findings, websiteThresholdFinding(website, "server_error_spike", "medium", "all",
			fmt.Sprintf("%s 在单个采集窗口出现 %d 次服务端错误", website.PrimaryDomain, serverErrors), serverErrors, serverErrorSamples, latest))
	}
	return findings
}

func websiteThresholdFinding(website model.Website, eventType, level, actor, summary string, count int, samples []string, seenAt time.Time) securityFinding {
	return securityFinding{
		SourceType: "website", SourceID: website.ID, SourceName: website.PrimaryDomain,
		EventType: eventType, Level: level, Actor: actor, Summary: summary, Value: float64(count), SeenAt: seenAt,
		Evidence: []securityEvidence{{Source: "website", Description: summary, Count: count, Samples: boundedSecuritySamples(samples, 5)}},
	}
}

func scrubWebsiteSecurityURI(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ScrubSecurityLogText(raw)
	}
	query := parsed.Query()
	for key := range query {
		query.Set(key, securityRedacted)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
