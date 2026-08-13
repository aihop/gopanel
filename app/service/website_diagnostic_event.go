package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/global"
)

const websiteDiagnosticSchema = "gopanel.website-diagnostic.v1"

var (
	diagnosticDynamicNumberPattern  = regexp.MustCompile(`\b\d{2,}\b`)
	diagnosticSensitivePattern      = regexp.MustCompile(`(?i)(authorization|cookie|token|password|passwd|secret|api[_-]?key)(\s*[=:]\s*)([^\s,;]+)`)
	diagnosticSensitiveQueryPattern = regexp.MustCompile(`(?i)([?&](?:token|access_token|password|secret|api_key)=)[^&#\s]+`)
)

type WebsiteDiagnosticEnvelope struct {
	Schema       string                 `json:"schema"`
	EventID      string                 `json:"eventId"`
	WebsiteID    uint                   `json:"websiteId"`
	Source       string                 `json:"source"`
	Kind         string                 `json:"kind"`
	Severity     string                 `json:"severity"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Stack        string                 `json:"stack"`
	RequestID    string                 `json:"requestId"`
	SessionID    string                 `json:"sessionId"`
	Method       string                 `json:"method"`
	Route        string                 `json:"route"`
	HTTPStatus   int                    `json:"httpStatus"`
	BusinessCode string                 `json:"businessCode"`
	DurationMS   int64                  `json:"durationMs"`
	Release      string                 `json:"release"`
	OccurredAt   time.Time              `json:"occurredAt"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func normalizeDiagnosticEnvelope(input *WebsiteDiagnosticEnvelope, websiteID uint) (*model.WebsiteDiagnosticEvent, error) {
	if input == nil || input.Schema != websiteDiagnosticSchema {
		return nil, buserr.New("ErrWebsiteDiagnosticInvalidSchema")
	}
	if input.WebsiteID != 0 && input.WebsiteID != websiteID {
		return nil, buserr.New("ErrWebsiteDiagnosticWebsiteMismatch")
	}
	input.EventID = limitedDiagnosticText(input.EventID, 128)
	input.Source = strings.ToLower(limitedDiagnosticText(input.Source, 32))
	input.Kind = strings.ToLower(limitedDiagnosticText(input.Kind, 64))
	if input.EventID == "" || input.Source == "" || input.Kind == "" {
		return nil, buserr.New("ErrWebsiteDiagnosticInvalidEvent")
	}
	if !map[string]bool{"backend": true, "browser": true, "caddy": true, "probe": true}[input.Source] {
		return nil, buserr.New("ErrWebsiteDiagnosticInvalidSource")
	}
	severity := strings.ToLower(limitedDiagnosticText(input.Severity, 16))
	if !map[string]bool{"info": true, "warning": true, "error": true, "critical": true}[severity] {
		severity = "error"
	}
	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	message := sanitizeDiagnosticText(input.Message, 8192)
	stack := sanitizeDiagnosticText(input.Stack, 16384)
	route := sanitizeDiagnosticRoute(input.Route)
	title := sanitizeDiagnosticText(input.Title, 255)
	if title == "" {
		title = diagnosticDefaultTitle(input.Kind, input.HTTPStatus, input.BusinessCode)
	}
	metadata, _ := json.Marshal(sanitizeDiagnosticValue(input.Metadata, 0))
	event := &model.WebsiteDiagnosticEvent{
		WebsiteID: websiteID, EventID: input.EventID, Source: input.Source, Kind: input.Kind,
		Severity: severity, Title: title, Message: message, Stack: stack,
		RequestID: limitedDiagnosticText(input.RequestID, 128), SessionID: limitedDiagnosticText(input.SessionID, 128),
		Method: strings.ToUpper(limitedDiagnosticText(input.Method, 16)), Route: route,
		HTTPStatus: input.HTTPStatus, BusinessCode: limitedDiagnosticText(input.BusinessCode, 128),
		DurationMS: input.DurationMS, Release: limitedDiagnosticText(input.Release, 128),
		Metadata: string(metadata), OccurredAt: occurredAt,
	}
	event.Fingerprint = websiteDiagnosticFingerprint(event)
	return event, nil
}

func websiteDiagnosticFingerprint(event *model.WebsiteDiagnosticEvent) string {
	stackLines := strings.Split(event.Stack, "\n")
	stack := strings.Join(stackLines[:min(3, len(stackLines))], "\n")
	normalized := strings.ToLower(strings.Join([]string{
		event.Kind, event.BusinessCode, strconv.Itoa(event.HTTPStatus),
		diagnosticDynamicNumberPattern.ReplaceAllString(stack, "#"),
		diagnosticDynamicNumberPattern.ReplaceAllString(event.Route, "#"),
	}, "|"))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func sanitizeDiagnosticText(value string, limit int) string {
	value = diagnosticSensitivePattern.ReplaceAllString(value, "$1$2[REDACTED]")
	value = diagnosticSensitiveQueryPattern.ReplaceAllString(value, "$1[REDACTED]")
	return limitedDiagnosticText(value, limit)
}

func sanitizeDiagnosticRoute(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "?"); index >= 0 {
		value = value[:index]
	}
	return limitedDiagnosticText(value, 512)
}

func sanitizeDiagnosticValue(value interface{}, depth int) interface{} {
	if depth > 4 {
		return "[TRUNCATED]"
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		clean := make(map[string]interface{}, min(len(typed), 50))
		count := 0
		for key, item := range typed {
			if count >= 50 {
				break
			}
			if diagnosticSensitivePattern.MatchString(key + "=value") {
				clean[key] = "[REDACTED]"
			} else {
				clean[limitedDiagnosticText(key, 128)] = sanitizeDiagnosticValue(item, depth+1)
			}
			count++
		}
		return clean
	case []interface{}:
		if len(typed) > 20 {
			typed = typed[:20]
		}
		clean := make([]interface{}, len(typed))
		for index := range typed {
			clean[index] = sanitizeDiagnosticValue(typed[index], depth+1)
		}
		return clean
	case string:
		return sanitizeDiagnosticText(typed, 2048)
	default:
		return typed
	}
}

func limitedDiagnosticText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func diagnosticDefaultTitle(kind string, status int, code string) string {
	if code != "" {
		return code
	}
	if status > 0 {
		return fmt.Sprintf("HTTP %d", status)
	}
	return kind
}

func ingestWebsiteDiagnosticEnvelope(websiteID uint, input *WebsiteDiagnosticEnvelope) (*model.WebsiteIssue, bool, error) {
	event, err := normalizeDiagnosticEnvelope(input, websiteID)
	if err != nil {
		return nil, false, err
	}
	setting, err := repo.NewWebsiteDiagnostic(global.DB).GetByWebsiteID(websiteID)
	if err != nil {
		return nil, false, err
	}
	if setting == nil || !setting.Enabled || !websiteDiagnosticEventAllowed(setting, event) {
		return nil, false, buserr.New("ErrWebsiteDiagnosticEventFiltered")
	}
	return repo.NewWebsiteDiagnostic(global.DB).IngestEvent(event)
}

func websiteDiagnosticEventAllowed(setting *model.WebsiteDiagnosticSetting, event *model.WebsiteDiagnosticEvent) bool {
	sourceEnabled := map[string]bool{
		"caddy":   setting.CaddyMonitoring,
		"probe":   setting.ActiveProbes,
		"backend": setting.BackendHook,
		"browser": setting.BrowserHook,
	}[event.Source]
	if !sourceEnabled {
		return false
	}

	kind := strings.ToLower(event.Kind)
	switch {
	case strings.Contains(kind, "resource"):
		return setting.MonitorResourceErrors
	case event.Source == "browser":
		return setting.MonitorBrowserErrors
	case kind == "http_4xx" || event.HTTPStatus >= 400 && event.HTTPStatus < 500:
		return setting.MonitorHTTP4xx
	case kind == "http_5xx" || event.HTTPStatus >= 500:
		return setting.MonitorHTTP5xx
	case strings.Contains(kind, "upstream"):
		return setting.MonitorUpstreamErrors
	case strings.Contains(kind, "slow_request"):
		return setting.MonitorSlowRequests
	case strings.Contains(kind, "business") || event.BusinessCode != "":
		return setting.MonitorBusinessErrors
	default:
		return true
	}
}
