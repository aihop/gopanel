package docker

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func RuntimeCLI(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := ResolveRuntime(ctx)
	var preferred string
	if resolved.Kind == RuntimePodman {
		preferred = "podman"
	} else {
		preferred = "docker"
	}
	if _, err := exec.LookPath(preferred); err == nil {
		return preferred, nil
	}
	alt := "docker"
	if preferred == "docker" {
		alt = "podman"
	}
	if _, err := exec.LookPath(alt); err == nil {
		return alt, nil
	}
	return "", errors.New("container runtime cli not found")
}

func RuntimeCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := RuntimeCLI(ctx)
	if err != nil {
		return nil, err
	}
	if bin == "podman" && runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	c := exec.CommandContext(ctx, bin, args...)
	return c, nil
}

func RuntimeCommandWithHost(ctx context.Context, host string, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := RuntimeCLI(ctx)
	if err != nil {
		return nil, err
	}
	if bin == "podman" && runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	h := strings.TrimSpace(host)
	if h != "" && h != "podman-cli" && strings.HasPrefix(h, "unix://") {
		if bin == "podman" {
			args = append([]string{"--url", h}, args...)
		} else {
			args = append([]string{"-H", h}, args...)
		}
	}
	c := exec.CommandContext(ctx, bin, args...)
	return c, nil
}

func InspectContainerRunning(ctx context.Context, containerName string) (bool, bool, error) {
	containerName = strings.TrimSpace(containerName)
	if containerName == "" {
		return false, false, errors.New("container name is empty")
	}
	c, err := RuntimeCommand(ctx, "inspect", "-f", "{{.State.Running}}", containerName)
	if err != nil {
		return false, false, err
	}
	out, runErr := c.CombinedOutput()
	if runErr != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(msg, "no such object") ||
			strings.Contains(msg, "no such container") ||
			strings.Contains(msg, "not found") ||
			strings.Contains(msg, "does not exist") {
			return false, false, nil
		}
		if msg == "" {
			return false, true, runErr
		}
		return false, true, errors.New(msg)
	}
	s := strings.ToLower(strings.TrimSpace(string(out)))
	if s == "true" {
		return true, true, nil
	}
	if s == "false" {
		return false, true, nil
	}
	return false, true, errors.New("unexpected inspect output: " + s)
}
