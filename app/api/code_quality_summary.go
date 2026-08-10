package api

import "strings"

func codeQualityFailureSummary(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	selected := codeQualityFailureLines(filtered)
	if len(selected) == 0 {
		for index := len(filtered) - 1; index >= 0 && len(selected) < 4; index-- {
			line := filtered[index]
			if line == "FAIL" || strings.Contains(line, "[no test files]") {
				continue
			}
			selected = append([]string{line}, selected...)
		}
	}
	summary := strings.Join(selected, " | ")
	if runes := []rune(summary); len(runes) > 500 {
		summary = string(runes[:500])
	}
	return summary
}

func codeQualityFailureLines(lines []string) []string {
	selected := make([]string, 0, 6)
	seen := make(map[string]struct{})
	appendLine := func(line string) {
		if line == "" || line == "FAIL" {
			return
		}
		if _, exists := seen[line]; exists {
			return
		}
		seen[line] = struct{}{}
		selected = append(selected, line)
	}
	for index, line := range lines {
		if strings.HasPrefix(line, "--- FAIL:") {
			appendLine(line)
			for next := index + 1; next < len(lines) && next <= index+2; next++ {
				if strings.HasPrefix(lines[next], "--- ") || strings.HasPrefix(lines[next], "FAIL\t") {
					break
				}
				appendLine(lines[next])
			}
			continue
		}
		if strings.HasPrefix(line, "panic:") || strings.Contains(line, "[build failed]") ||
			strings.Contains(line, ": undefined:") || strings.Contains(line, ": syntax error:") {
			appendLine(line)
			continue
		}
		if strings.HasPrefix(line, "FAIL\t") || strings.HasPrefix(line, "FAIL ") {
			appendLine(line)
		}
	}
	return selected
}
