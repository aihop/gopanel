package docker

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

func RuntimeCLI(ctx context.Context) (string, error) {
	return DefaultRuntimeAdapter().CLI(ctx)
}

func RuntimeCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	return DefaultRuntimeAdapter().Command(ctx, args...)
}

func RuntimeCommandWithHost(ctx context.Context, host string, args ...string) (*exec.Cmd, error) {
	return DefaultRuntimeAdapter().CommandWithHost(ctx, host, args...)
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
