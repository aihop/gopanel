package api

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

var (
	aiErrorSummaryPattern = regexp.MustCompile(`(?im)^(error|fatal|failed|panic|exception|执行错误)[:：]?\s+(.+)$`)
	aiFilePathPattern     = regexp.MustCompile(`(?i)([A-Za-z0-9_\-./]+\.(go|ts|tsx|js|jsx|vue|json|yaml|yml|md|css|scss|html|dart|py|java|kt|rs|sql))`)
)

func createAITimelineEvent(sessionRepo repo.IAIDevSessionRepo, event *model.AITimelineEvent) {
	if sessionRepo == nil || event == nil || event.SessionID == 0 {
		return
	}
	_ = sessionRepo.CreateTimelineEvent(event)
}

func buildTimelineContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	runes := []rune(content)
	if len(runes) > 240 {
		return strings.TrimSpace(string(runes[:240])) + "..."
	}
	return content
}

func extractAIErrorSummary(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	matches := aiErrorSummaryPattern.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			return strings.TrimSpace(match[2])
		}
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "error") ||
			strings.Contains(lower, "failed") ||
			strings.Contains(lower, "panic") ||
			strings.Contains(lower, "exception") ||
			strings.Contains(trimmed, "执行错误") {
			return trimmed
		}
	}
	return ""
}

func extractAIChangedFiles(output string) []string {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}
	matches := aiFilePathPattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		candidate := strings.TrimSpace(match[1])
		candidate = strings.Trim(candidate, `"'()[]{}:,;`)
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if candidate == "." || candidate == "/" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		files = append(files, candidate)
	}
	sort.Strings(files)
	if len(files) > 8 {
		files = files[:8]
	}
	return files
}

func summarizeAIRecentOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	if len(filtered) == 0 {
		return ""
	}
	if len(filtered) > 3 {
		filtered = filtered[len(filtered)-3:]
	}
	result := strings.Join(filtered, "\n")
	return buildTimelineContent(result)
}
