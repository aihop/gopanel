//go:build linux

package helper

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (s *Server) actionSSHLoginLogList(ctx context.Context, params map[string]interface{}) (string, error) {
	query := parseSSHLogQuery(params)
	now := time.Now()
	scanLimit := sshScanLimit(query)

	lines, source, warning, partial, err := collectLinuxSSHLogLines(ctx, scanLimit)
	if err != nil {
		result := sshLoginResult{
			Supported: true,
			Platform:  "linux",
			Source:    source,
			Partial:   partial,
			Warning:   err.Error(),
			Items:     []sshLoginEvent{},
		}
		if warning != "" {
			result.Warning = warning + "; " + err.Error()
		}
		output, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return "", marshalErr
		}
		return string(output), nil
	}

	result := finalizeSSHLoginResult(parseSSHLogLines(lines, "linux", source, now), query, "linux", source, warning, partial)
	if len(result.Items) == 0 && result.Total == 0 && warning == "" {
		result.Warning = "最近日志中未发现 SSH 登录事件"
	}
	output, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func collectLinuxSSHLogLines(ctx context.Context, scanLimit int) ([]string, string, string, bool, error) {
	var warnings []string

	if _, err := exec.LookPath("journalctl"); err == nil {
		args := []string{"--no-pager", "-o", "short-iso", "--since", "24 hours ago", "-n", strconv.Itoa(scanLimit)}
		out, cmdErr := exec.CommandContext(ctx, "journalctl", args...).CombinedOutput()
		if cmdErr == nil {
			lines := filterSSHLogLines(strings.Split(string(out), "\n"))
			if len(lines) > 0 {
				return lines, "journalctl", "", false, nil
			}
			warnings = append(warnings, "journalctl 未解析到 SSH 登录事件，已回退文件日志")
		} else {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = cmdErr.Error()
			}
			warnings = append(warnings, "journalctl 不可用: "+msg)
		}
	}

	for _, path := range []string{"/var/log/auth.log", "/var/log/secure"} {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		lines, readErr := readLastLines(path, scanLimit)
		if readErr != nil {
			warnings = append(warnings, path+" 读取失败: "+readErr.Error())
			continue
		}
		lines = filterSSHLogLines(lines)
		if len(lines) == 0 {
			warnings = append(warnings, path+" 未解析到 SSH 登录事件")
			continue
		}
		return lines, path, strings.Join(warnings, "; "), len(warnings) > 0, nil
	}

	return nil, "", strings.Join(warnings, "; "), len(warnings) > 0, errors.New("未找到可用的 SSH 登录日志源")
}
