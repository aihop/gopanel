package compose

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func resolveComposeCommand() (string, []string, error) {
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", []string{"compose"}, nil
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", []string{"compose"}, nil
	}
	if _, err := exec.LookPath("podman-compose"); err == nil {
		return "podman-compose", nil, nil
	}
	return "", nil, fmt.Errorf("no compose command found (docker/podman/podman-compose)")
}

func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	bin, prefix, err := resolveComposeCommand()
	if err != nil {
		return nil, err
	}
	allArgs := append(prefix, args...)
	if ctx == nil {
		return exec.Command(bin, allArgs...), nil
	}
	return exec.CommandContext(ctx, bin, allArgs...), nil
}

func Exec(ctx context.Context, args ...string) (string, error) {
	c, err := Command(ctx, args...)
	if err != nil {
		return "", err
	}
	out, runErr := c.CombinedOutput()
	if runErr != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return string(out), runErr
		}
		return string(out), fmt.Errorf("%w: %s", runErr, msg)
	}
	return string(out), nil
}

func Pull(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "pull")
}

func Up(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "up", "-d")
}

func Down(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "down", "--remove-orphans")
}

func Start(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "start")
}

func Stop(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "stop")
}

func Restart(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "restart")
}

func Operate(filePath, operation string) (string, error) {
	opArgs := strings.Fields(operation)
	args := append([]string{"-f", filePath}, opArgs...)
	return Exec(context.Background(), args...)
}
