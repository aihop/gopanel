package service

import (
	"net/url"
	"regexp"
	"strings"
)

const securityRedacted = "[REDACTED]"

var (
	securityHeaderSecretPattern = regexp.MustCompile(`(?i)(authorization|proxy-authorization|cookie|set-cookie)\s*[:=]\s*[^\s,;]+`)
	securityKeyValuePattern     = regexp.MustCompile(`(?i)(api[_-]?key|token|password|passwd|secret|credential)\s*[:=]\s*[^&\s,;]+`)
	securityJWTTokenPattern     = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)
	securityPrivateKeyPattern   = regexp.MustCompile(`(?s)-----BEGIN[^\n]*PRIVATE KEY-----.*?-----END[^\n]*PRIVATE KEY-----`)
)

func ScrubSecurityLogText(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	value = securityPrivateKeyPattern.ReplaceAllString(value, securityRedacted)
	value = securityHeaderSecretPattern.ReplaceAllString(value, `$1: `+securityRedacted)
	value = securityKeyValuePattern.ReplaceAllString(value, `$1=`+securityRedacted)
	value = securityJWTTokenPattern.ReplaceAllString(value, securityRedacted)
	return scrubSecurityURLQuery(value)
}

func scrubSecurityURLQuery(value string) string {
	fields := strings.Fields(value)
	for index, field := range fields {
		trimmed := strings.Trim(field, `"'(),[]`)
		parsed, err := url.Parse(trimmed)
		if err != nil || parsed.RawQuery == "" {
			continue
		}
		query := parsed.Query()
		changed := false
		for key := range query {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "key") ||
				strings.Contains(lower, "pass") || strings.Contains(lower, "secret") ||
				strings.Contains(lower, "auth") || strings.Contains(lower, "session") {
				query.Set(key, securityRedacted)
				changed = true
			}
		}
		if changed {
			parsed.RawQuery = query.Encode()
			fields[index] = strings.Replace(field, trimmed, parsed.String(), 1)
		}
	}
	return strings.Join(fields, " ")
}
