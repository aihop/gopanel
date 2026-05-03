package service

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"os/exec"
	"strconv"
	"strings"
)

func composeAvailable() bool {
	if _, err := exec.LookPath("podman"); err == nil {
		return true
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return true
	}
	if _, err := exec.LookPath("podman-compose"); err == nil {
		return true
	}
	return false
}
func hasDockerSockPathSetting() bool {
	var settingItem model.Setting
	if err := global.DB.Where("key = ?", "DockerSockPath").First(&settingItem).Error; err != nil {
		return false
	}
	return strings.TrimSpace(settingItem.Value) != ""
}
func isPodmanInstalled() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}
func splitNonEmptyLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
func parsePercent(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
func parsePairToMB(s string) (float64, float64, bool) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, ok1 := parseSizeToMB(parts[0])
	b, ok2 := parseSizeToMB(parts[1])
	return a, b, ok1 && ok2
}
func parseMemUsagePairToMB(s string) (float64, bool) {
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return 0, false
	}
	a, ok := parseSizeToMB(parts[0])
	return a, ok
}
func parseSizeToMB(s string) (float64, bool) {
	b, ok := parseSizeToBytes(s)
	if !ok {
		return 0, false
	}
	return b / 1024.0 / 1024.0, true
}
func parseSizeToBytes(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.ReplaceAll(s, "I", "")
	s = strings.ReplaceAll(s, "B", "")
	s = strings.TrimSpace(s)
	unit := ""
	switch {
	case strings.HasSuffix(s, "K"):
		unit = "K"
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		unit = "M"
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		unit = "G"
		s = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "T"):
		unit = "T"
		s = strings.TrimSuffix(s, "T")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case "K":
		v *= 1024
	case "M":
		v *= 1024 * 1024
	case "G":
		v *= 1024 * 1024 * 1024
	case "T":
		v *= 1024 * 1024 * 1024 * 1024
	}
	return v, true
}
func stringsToMap(list []string) map[string]string {
	var labelMap = make(map[string]string)
	for _, label := range list {
		if strings.Contains(label, "=") {
			sps := strings.SplitN(label, "=", 2)
			labelMap[sps[0]] = sps[1]
		}
	}
	return labelMap
}
