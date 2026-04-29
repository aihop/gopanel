package docker

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func runtimeBinaryPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("binary name is empty")
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	if runtime.GOOS == "darwin" {
		for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
			candidate := filepath.Join(dir, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", exec.ErrNotFound
}

func PodmanBinaryPath() (string, error) {
	return runtimeBinaryPath("podman")
}

func DockerBinaryPath() (string, error) {
	return runtimeBinaryPath("docker")
}
