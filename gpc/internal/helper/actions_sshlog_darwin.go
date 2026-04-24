//go:build darwin

package helper

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

func (s *Server) actionSSHLoginLogList(ctx context.Context, params map[string]interface{}) (string, error) {
	query := parseSSHLogQuery(params)
	args := []string{
		"show",
		"--style", "syslog",
		"--last", "24h",
		"--predicate", `process == "sshd"`,
	}
	out, err := exec.CommandContext(ctx, "log", args...).CombinedOutput()
	if err != nil {
		return "", err
	}

	lines := filterSSHLogLines(strings.Split(string(out), "\n"))
	result := finalizeSSHLoginResult(parseSSHLogLines(lines, "darwin", "macos-log", time.Now()), query, "darwin", "macos-log", "", false)
	if len(result.Items) == 0 && result.Total == 0 {
		result.Warning = "最近日志中未发现 SSH 登录事件"
	}
	output, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
