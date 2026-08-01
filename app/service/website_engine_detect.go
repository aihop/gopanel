package service

import (
	"github.com/docker/docker/api/types/image"
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
