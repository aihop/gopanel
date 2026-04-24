package helper

import (
	"bufio"
	"errors"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type sshLoginEvent struct {
	CreatedAt  string `json:"createdAt"`
	Status     string `json:"status"`
	Username   string `json:"username"`
	SourceIP   string `json:"sourceIp"`
	SourcePort string `json:"sourcePort"`
	AuthMethod string `json:"authMethod"`
	Message    string `json:"message"`
	Raw        string `json:"raw"`
	Platform   string `json:"platform"`
	Source     string `json:"source"`
}

type sshLoginResult struct {
	Supported       bool            `json:"supported"`
	Platform        string          `json:"platform"`
	Source          string          `json:"source"`
	Partial         bool            `json:"partial"`
	Warning         string          `json:"warning"`
	Items           []sshLoginEvent `json:"items"`
	Total           int             `json:"total"`
	SuccessfulCount int             `json:"successfulCount"`
	FailedCount     int             `json:"failedCount"`
}

type sshLogQuery struct {
	Page     int
	Limit    int
	Status   string
	Username string
	IP       string
}

type sshParsedEvent struct {
	Time time.Time
	sshLoginEvent
}

var (
	sshPrefixRe = regexp.MustCompile(`^(.+?)\s+[^\s]+\s+sshd(?:\[\d+\])?:\s+(.*)$`)
	acceptedRe  = regexp.MustCompile(`^Accepted\s+([^\s]+(?:/[^\s]+)?)\s+for\s+(?:invalid user\s+)?([^\s]+)\s+from\s+([^\s]+)\s+port\s+(\d+)`)
	failedRe    = regexp.MustCompile(`^Failed\s+([^\s]+(?:/[^\s]+)?)\s+for\s+(?:invalid user\s+)?([^\s]+)\s+from\s+([^\s]+)\s+port\s+(\d+)`)
	invalidRe   = regexp.MustCompile(`^Invalid user\s+([^\s]+)\s+from\s+([^\s]+)\s+port\s+(\d+)`)
)

func parseSSHLogQuery(params map[string]interface{}) sshLogQuery {
	page, ok := getInt(params, "page")
	if !ok || page < 1 {
		page = 1
	}
	limit, ok := getInt(params, "limit")
	if !ok || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return sshLogQuery{
		Page:     page,
		Limit:    limit,
		Status:   strings.TrimSpace(getString(params, "status")),
		Username: strings.TrimSpace(getString(params, "username")),
		IP:       strings.TrimSpace(getString(params, "ip")),
	}
}

func sshScanLimit(q sshLogQuery) int {
	limit := q.Page * q.Limit * 20
	if limit < 200 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	return limit
}

func finalizeSSHLoginResult(events []sshParsedEvent, q sshLogQuery, platform, source, warning string, partial bool) sshLoginResult {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.After(events[j].Time)
	})

	var filtered []sshParsedEvent
	successCount := 0
	failedCount := 0
	for _, item := range events {
		if q.Status != "" && !strings.EqualFold(item.Status, q.Status) && !strings.EqualFold(q.Status, "all") {
			continue
		}
		if q.Username != "" && !strings.Contains(strings.ToLower(item.Username), strings.ToLower(q.Username)) {
			continue
		}
		if q.IP != "" && !strings.Contains(strings.ToLower(item.SourceIP), strings.ToLower(q.IP)) {
			continue
		}
		filtered = append(filtered, item)
		if item.Status == "Success" {
			successCount++
		} else if item.Status == "Failed" {
			failedCount++
		}
	}

	start := (q.Page - 1) * q.Limit
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + q.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	items := make([]sshLoginEvent, 0, end-start)
	for _, item := range filtered[start:end] {
		items = append(items, item.sshLoginEvent)
	}

	return sshLoginResult{
		Supported:       true,
		Platform:        platform,
		Source:          source,
		Partial:         partial,
		Warning:         warning,
		Items:           items,
		Total:           len(filtered),
		SuccessfulCount: successCount,
		FailedCount:     failedCount,
	}
}

func parseSSHLogLines(lines []string, platform, source string, now time.Time) []sshParsedEvent {
	events := make([]sshParsedEvent, 0, len(lines))
	for _, line := range lines {
		if event, ok := parseSSHLogLine(line, platform, source, now); ok {
			events = append(events, event)
		}
	}
	return events
}

func parseSSHLogLine(line, platform, source string, now time.Time) (sshParsedEvent, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return sshParsedEvent{}, false
	}

	matches := sshPrefixRe.FindStringSubmatch(line)
	if len(matches) != 3 {
		return sshParsedEvent{}, false
	}

	ts, ok := parseSSHLogTimestamp(matches[1], now)
	if !ok {
		return sshParsedEvent{}, false
	}

	message := strings.TrimSpace(matches[2])
	event := sshParsedEvent{
		Time: ts,
		sshLoginEvent: sshLoginEvent{
			CreatedAt:  ts.Format(time.RFC3339),
			Message:    message,
			Raw:        line,
			Platform:   platform,
			Source:     source,
			AuthMethod: "unknown",
		},
	}

	if accepted := acceptedRe.FindStringSubmatch(message); len(accepted) == 5 {
		event.Status = "Success"
		event.AuthMethod = normalizeSSHAuthMethod(accepted[1])
		event.Username = accepted[2]
		event.SourceIP = accepted[3]
		event.SourcePort = accepted[4]
		return event, true
	}
	if failed := failedRe.FindStringSubmatch(message); len(failed) == 5 {
		event.Status = "Failed"
		event.AuthMethod = normalizeSSHAuthMethod(failed[1])
		event.Username = failed[2]
		event.SourceIP = failed[3]
		event.SourcePort = failed[4]
		return event, true
	}
	if invalid := invalidRe.FindStringSubmatch(message); len(invalid) == 4 {
		event.Status = "Failed"
		event.Username = invalid[1]
		event.SourceIP = invalid[2]
		event.SourcePort = invalid[3]
		return event, true
	}
	return sshParsedEvent{}, false
}

func parseSSHLogTimestamp(value string, now time.Time) (time.Time, bool) {
	value = strings.TrimSpace(value)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05.999999-0700",
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts, true
		}
	}

	withYear := strconv.Itoa(now.Year()) + " " + value
	for _, layout := range []string{
		"2006 Jan 2 15:04:05",
		"2006 Jan 02 15:04:05",
		"2006 Jan _2 15:04:05",
	} {
		if ts, err := time.ParseInLocation(layout, withYear, now.Location()); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func normalizeSSHAuthMethod(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "password":
		return "password"
	case "publickey":
		return "publickey"
	case "keyboard-interactive", "keyboard-interactive/pam":
		return "keyboard-interactive"
	default:
		return value
	}
}

func readLastLines(path string, limit int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if limit <= 0 {
		limit = 200
	}

	buf := make([]string, 0, limit)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(buf) < limit {
			buf = append(buf, line)
			continue
		}
		copy(buf, buf[1:])
		buf[len(buf)-1] = line
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, errors.New("empty log")
	}
	return buf, nil
}

func filterSSHLogLines(lines []string) []string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if !strings.Contains(lower, "sshd") {
			continue
		}
		if !(strings.Contains(lower, "accepted ") || strings.Contains(lower, "failed ") || strings.Contains(lower, "invalid user")) {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}
