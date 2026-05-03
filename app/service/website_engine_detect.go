package service

import (
	"fmt"
	"github.com/docker/docker/api/types/image"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func logEngineProgress(progress func(format string, a ...interface{}), format string, a ...interface{}) {
	if progress != nil {
		progress(format, a...)
	}
}
func detectEngineContainerPort(imageInspect image.InspectResponse) string {
	for _, env := range imageInspect.Config.Env {
		if strings.HasPrefix(env, "PORT=") {
			port := strings.TrimSpace(strings.TrimPrefix(env, "PORT="))
			if port != "" {
				return port
			}
		}
	}
	var ports []string
	for port := range imageInspect.Config.ExposedPorts {
		if strings.HasSuffix(string(port), "/tcp") {
			ports = append(ports, strings.TrimSuffix(string(port), "/tcp"))
		}
	}
	sort.Strings(ports)
	if len(ports) > 0 {
		return ports[0]
	}
	return "80"
}
func detectEngineWorkingDir(imageInspect image.InspectResponse) string {
	if strings.TrimSpace(imageInspect.Config.WorkingDir) != "" {
		return strings.TrimSpace(imageInspect.Config.WorkingDir)
	}
	return "/app"
}
func shouldAutoMountCodeDir(imageInspect image.InspectResponse, workingDir, codeDir string) (bool, string) {
	if strings.TrimSpace(codeDir) == "" || strings.TrimSpace(workingDir) == "" {
		return false, "挂载源目录或工作目录为空"
	}
	relativeEntry := detectRelativeEntrypoint(imageInspect)
	if relativeEntry == "" {
		return false, ""
	}
	info, err := os.Stat(codeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("挂载源目录不存在: %s", codeDir)
		}
		return false, fmt.Sprintf("挂载源目录不可访问: %s", err)
	}
	if !info.IsDir() {
		return false, fmt.Sprintf("挂载源路径不是目录: %s", codeDir)
	}
	targetFile := filepath.Join(codeDir, strings.TrimPrefix(relativeEntry, "./"))
	fileInfo, err := os.Stat(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("挂载目录中缺少启动文件 %s", relativeEntry)
		}
		return false, fmt.Sprintf("无法检查启动文件 %s: %s", relativeEntry, err)
	}
	if fileInfo.IsDir() {
		return false, fmt.Sprintf("启动文件 %s 实际是目录", relativeEntry)
	}
	return true, ""
}
func resolveAutoMountCodeDir(imageInspect image.InspectResponse, workingDir string, candidates ...string) (string, string) {
	relativeEntry := detectRelativeEntrypoint(imageInspect)
	if relativeEntry == "" {
		return "", ""
	}
	seen := make(map[string]struct{})
	var reasons []string
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		ok, reason := shouldAutoMountCodeDir(imageInspect, workingDir, candidate)
		if ok {
			return candidate, ""
		}
		if reason != "" {
			reasons = append(reasons, fmt.Sprintf("%s (%s)", candidate, reason))
		}
	}
	if len(reasons) == 0 {
		return "", ""
	}
	return "", strings.Join(reasons, "; ")
}
func detectRelativeEntrypoint(imageInspect image.InspectResponse) string {
	commands := append([]string{}, imageInspect.Config.Entrypoint...)
	commands = append(commands, imageInspect.Config.Cmd...)
	for _, entry := range commands {
		entry = strings.TrimSpace(entry)
		if strings.HasPrefix(entry, "./") {
			return entry
		}
	}
	return ""
}
