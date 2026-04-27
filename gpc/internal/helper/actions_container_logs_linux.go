//go:build linux

package helper

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

type podmanContainerJournalLogsResult struct {
	Lines  []string `json:"lines"`
	Cursor string   `json:"cursor,omitempty"`
}

func (s *Server) actionPodmanContainerJournalLogs(ctx context.Context, params map[string]interface{}) (string, error) {
	containerID := strings.TrimSpace(getString(params, "container_id"))
	containerName := strings.TrimSpace(getString(params, "container_name"))
	if containerID == "" && containerName == "" {
		return "", errors.New("invalid params: container_id is empty")
	}
	if _, err := exec.LookPath("journalctl"); err != nil {
		return "", err
	}

	args := []string{"--no-pager", "-q", "-o", "short-iso", "--show-cursor"}
	afterCursor := strings.TrimSpace(getString(params, "after_cursor"))
	if afterCursor != "" {
		args = append(args, "--after-cursor", afterCursor)
	} else {
		if since := normalizeGPCJournalSince(getString(params, "since")); since != "" {
			args = append(args, "--since", since)
		}
		if tail := strings.TrimSpace(getString(params, "tail")); tail != "" && tail != "0" {
			args = append(args, "-n", tail)
		}
	}
	if containerID != "" {
		args = append(args, "CONTAINER_ID_FULL="+containerID)
	} else {
		args = append(args, "CONTAINER_NAME="+containerName)
	}

	out, err := exec.CommandContext(ctx, "journalctl", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}

	result := parseJournalctlOutput(string(out))
	if result.Lines == nil {
		result.Lines = []string{}
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeGPCJournalSince(since string) string {
	since = strings.TrimSpace(since)
	if since == "" || since == "all" {
		return ""
	}
	if dur, err := time.ParseDuration(since); err == nil {
		return time.Now().Add(-dur).Format(time.RFC3339)
	}
	return since
}

func parseJournalctlOutput(raw string) podmanContainerJournalLogsResult {
	result := podmanContainerJournalLogsResult{
		Lines: []string{},
	}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r", ""), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "-- cursor: ") {
			result.Cursor = strings.TrimSpace(strings.TrimPrefix(trimmed, "-- cursor: "))
			continue
		}
		result.Lines = append(result.Lines, line)
	}
	return result
}
