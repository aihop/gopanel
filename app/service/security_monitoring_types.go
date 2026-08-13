package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

type securityEvidence struct {
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Count       int      `json:"count"`
	Samples     []string `json:"samples,omitempty"`
}

type securityFinding struct {
	SourceType string
	SourceID   uint
	SourceName string
	EventType  string
	Level      string
	Actor      string
	Summary    string
	Value      float64
	SeenAt     time.Time
	Evidence   []securityEvidence
}

func (finding securityFinding) event() *model.SecurityEvent {
	evidence, _ := json.Marshal(finding.Evidence)
	seenAt := finding.SeenAt
	if seenAt.IsZero() {
		seenAt = time.Now()
	}
	return &model.SecurityEvent{
		SourceType: finding.SourceType, SourceID: finding.SourceID, SourceName: finding.SourceName,
		EventType: finding.EventType, Level: finding.Level, Fingerprint: securityFingerprint(
			finding.SourceType, fmt.Sprintf("%d", finding.SourceID), finding.EventType, finding.Actor,
		),
		Summary: finding.Summary, Evidence: string(evidence), Value: finding.Value,
		FirstSeenAt: seenAt, LastSeenAt: seenAt,
	}
}

func securityFingerprint(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(part)))
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\x00")))
	return hex.EncodeToString(sum[:])
}

func securityLevelRank(level string) int {
	switch level {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func boundedSecuritySamples(samples []string, limit int) []string {
	if limit < 1 {
		return nil
	}
	result := make([]string, 0, min(len(samples), limit))
	for _, sample := range samples {
		sample = strings.TrimSpace(ScrubSecurityLogText(sample))
		if sample == "" {
			continue
		}
		if len([]rune(sample)) > 500 {
			sample = string([]rune(sample)[:500]) + "…"
		}
		result = append(result, sample)
		if len(result) == limit {
			break
		}
	}
	return result
}
