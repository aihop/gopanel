package service

import (
	"fmt"
	"github.com/aihop/gopanel/utils/common"
	"os"
	"strings"
	"time"
)

// normalizeComposeTimezoneVolumes strips host timezone bind mounts (/etc/timezone,
// /etc/localtime) that break container startup. When removeMissingOnly is true the
// mount is only dropped if the host source file is absent (e.g. Debian/minimal Linux
// without /etc/timezone), so normal hosts keep their original mounts untouched.
func normalizeComposeTimezoneVolumes(volumes interface{}, removeMissingOnly bool) (interface{}, bool) {
	list, ok := volumes.([]interface{})
	if !ok {
		return volumes, false
	}
	filtered := make([]interface{}, 0, len(list))
	changed := false
	for _, item := range list {
		if source, isTimezone := composeTimezoneVolumeSource(item); isTimezone {
			if !removeMissingOnly || !hostPathExists(source) {
				changed = true
				continue
			}
		}
		filtered = append(filtered, item)
	}
	if !changed {
		return volumes, false
	}
	if len(filtered) == 0 {
		return nil, true
	}
	return filtered, true
}

// composeTimezoneVolumeSource reports whether the volume mounts /etc/timezone or
// /etc/localtime from the host and returns that host source path.
func composeTimezoneVolumeSource(item interface{}) (string, bool) {
	switch v := item.(type) {
	case string:
		raw := strings.TrimSpace(v)
		if strings.HasPrefix(raw, "/etc/timezone:") {
			return "/etc/timezone", true
		}
		if strings.HasPrefix(raw, "/etc/localtime:") {
			return "/etc/localtime", true
		}
		return "", false
	case map[string]interface{}:
		source := strings.TrimSpace(fmt.Sprint(v["source"]))
		if source == "/etc/timezone" || source == "/etc/localtime" {
			return source, true
		}
		return "", false
	default:
		return "", false
	}
}

// hostPathExists reports whether path exists on the host. It uses Lstat so a
// (possibly dangling) /etc/localtime symlink still counts as present, since the
// bind mount of the link itself works.
func hostPathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Lstat(path)
	return err == nil
}
func ensureComposeTimezoneEnv(environment interface{}, timezone string) (interface{}, bool) {
	switch env := environment.(type) {
	case nil:
		return []interface{}{"TZ=" + timezone}, true
	case map[string]interface{}:
		if hasComposeTimezoneEnvMap(env) {
			return environment, false
		}
		env["TZ"] = timezone
		return env, true
	case []interface{}:
		if hasComposeTimezoneEnvList(env) {
			return environment, false
		}
		return append(env, "TZ="+timezone), true
	default:
		return environment, false
	}
}
func hasComposeTimezoneEnvMap(env map[string]interface{}) bool {
	for key, value := range env {
		if strings.EqualFold(strings.TrimSpace(key), "TZ") && strings.TrimSpace(fmt.Sprint(value)) != "" {
			return true
		}
	}
	return false
}
func hasComposeTimezoneEnvList(env []interface{}) bool {
	for _, item := range env {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(fmt.Sprint(item))), "TZ=") {
			return true
		}
	}
	return false
}
func resolveComposeTimezoneCompatValue() string {
	tz := strings.TrimSpace(common.LoadTimeZoneByCmd())
	if tz == "" || strings.EqualFold(tz, "local") {
		return "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "Asia/Shanghai"
	}
	return tz
}
func normalizeComposeCommandShellCompat(command interface{}) (interface{}, bool) {
	commandStr, ok := command.(string)
	if !ok {
		return command, false
	}
	commandStr = strings.TrimSpace(commandStr)
	if commandStr == "" || !composeShellConditionalExprPattern.MatchString(commandStr) {
		return command, false
	}
	compact := strings.TrimSpace(strings.ToLower(commandStr))
	if strings.HasPrefix(compact, "/bin/sh -c ") || strings.HasPrefix(compact, "sh -c ") {
		return command, false
	}
	return []interface{}{"/bin/sh", "-c", "exec " + commandStr}, true
}
func normalizeComposeCommandEnvCompat(command interface{}, envMap map[string]string) (interface{}, bool) {
	switch v := command.(type) {
	case string:
		return expandComposeCommandVariables(v, envMap)
	case []interface{}:
		out := make([]interface{}, len(v))
		changed := false
		for i, item := range v {
			strItem, ok := item.(string)
			if !ok {
				out[i] = item
				continue
			}
			normalized, itemChanged := expandComposeCommandVariables(strItem, envMap)
			out[i] = normalized
			changed = changed || itemChanged
		}
		if !changed {
			return command, false
		}
		return out, true
	default:
		return command, false
	}
}
func expandComposeCommandVariables(command string, envMap map[string]string) (string, bool) {
	expanded, changed := expandComposeConditionalExprs(command, envMap)
	plainExpanded := expandComposePlainVars(expanded, envMap)
	return plainExpanded, changed || plainExpanded != command
}
func expandComposeConditionalExprs(input string, envMap map[string]string) (string, bool) {
	var out strings.Builder
	changed := false
	for i := 0; i < len(input); {
		if !strings.HasPrefix(input[i:], "${") {
			out.WriteByte(input[i])
			i++
			continue
		}
		varName, alt, next, ok := parseComposeConditionalExpr(input, i)
		if !ok {
			out.WriteByte(input[i])
			i++
			continue
		}
		if strings.TrimSpace(envMap[varName]) != "" {
			nestedExpanded, _ := expandComposeConditionalExprs(alt, envMap)
			out.WriteString(expandComposePlainVars(nestedExpanded, envMap))
		}
		changed = true
		i = next
	}
	if !changed {
		return input, false
	}
	return out.String(), true
}
func parseComposeConditionalExpr(input string, start int) (string, string, int, bool) {
	if start < 0 || start+2 > len(input) || input[start:start+2] != "${" {
		return "", "", 0, false
	}
	i := start + 2
	nameStart := i
	for i < len(input) {
		ch := input[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			i++
			continue
		}
		break
	}
	if i == nameStart || i+2 > len(input) || input[i:i+2] != ":+" {
		return "", "", 0, false
	}
	name := input[nameStart:i]
	i += 2
	altStart := i
	nested := 0
	for i < len(input) {
		if i+2 <= len(input) && input[i:i+2] == "${" {
			nested++
			i += 2
			continue
		}
		if input[i] == '}' {
			if nested == 0 {
				return name, input[altStart:i], i + 1, true
			}
			nested--
		}
		i++
	}
	return "", "", 0, false
}
func expandComposePlainVars(input string, envMap map[string]string) string {
	if input == "" {
		return input
	}
	return os.Expand(input, func(key string) string {
		return envMap[key]
	})
}
