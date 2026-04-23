package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

type RuntimeAdapter interface {
	Resolve(ctx context.Context) ResolvedRuntime
	CLI(ctx context.Context) (string, error)
	DockerClient(ctx context.Context) (*client.Client, error)
	Command(ctx context.Context, args ...string) (*exec.Cmd, error)
	CommandWithHost(ctx context.Context, host string, args ...string) (*exec.Cmd, error)
	ResolveComposeCommand(ctx context.Context) (string, []string, error)
	ComposeCommand(ctx context.Context, args ...string) (*exec.Cmd, error)
}

type defaultRuntimeAdapter struct{}

var runtimeAdapter RuntimeAdapter = &defaultRuntimeAdapter{}

func DefaultRuntimeAdapter() RuntimeAdapter {
	return runtimeAdapter
}

func SetRuntimeAdapter(adapter RuntimeAdapter) {
	if adapter == nil {
		runtimeAdapter = &defaultRuntimeAdapter{}
		return
	}
	runtimeAdapter = adapter
}

func (a *defaultRuntimeAdapter) Resolve(ctx context.Context) ResolvedRuntime {
	return ResolveRuntime(ctx)
}

func (a *defaultRuntimeAdapter) CLI(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := a.Resolve(ctx)
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

func (a *defaultRuntimeAdapter) DockerClient(ctx context.Context) (*client.Client, error) {
	resolved := a.Resolve(ctx)
	if resolved.Kind == RuntimePodman && runtime.GOOS == "darwin" {
		return nil, errors.New("podman on darwin does not support docker api client")
	}
	return client.NewClientWithOpts(client.FromEnv, client.WithHost(resolved.Host), client.WithAPIVersionNegotiation())
}

func (a *defaultRuntimeAdapter) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := a.CLI(ctx)
	if err != nil {
		return nil, err
	}
	if bin == "podman" && runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	return exec.CommandContext(ctx, bin, args...), nil
}

func (a *defaultRuntimeAdapter) CommandWithHost(ctx context.Context, host string, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := a.CLI(ctx)
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
	return exec.CommandContext(ctx, bin, args...), nil
}

func (a *defaultRuntimeAdapter) ResolveComposeCommand(ctx context.Context) (string, []string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := a.Resolve(ctx)
	tryDocker := func() (string, []string, bool) {
		if _, err := exec.LookPath("docker"); err == nil && dockerComposeAvailable() {
			return "docker", []string{"compose"}, true
		}
		return "", nil, false
	}
	tryPodman := func() (string, []string, bool) {
		if _, err := exec.LookPath("podman"); err == nil && podmanComposeAvailable() {
			return "podman", []string{"compose"}, true
		}
		return "", nil, false
	}
	tryPodmanCompose := func() (string, []string, bool) {
		if _, err := exec.LookPath("podman-compose"); err == nil {
			return "podman-compose", nil, true
		}
		return "", nil, false
	}

	if resolved.Kind == RuntimePodman {
		if bin, prefix, ok := tryPodman(); ok {
			return bin, prefix, nil
		}
		if bin, prefix, ok := tryPodmanCompose(); ok {
			return bin, prefix, nil
		}
		if bin, prefix, ok := tryDocker(); ok {
			return bin, prefix, nil
		}
	} else {
		if bin, prefix, ok := tryDocker(); ok {
			return bin, prefix, nil
		}
		if bin, prefix, ok := tryPodman(); ok {
			return bin, prefix, nil
		}
		if bin, prefix, ok := tryPodmanCompose(); ok {
			return bin, prefix, nil
		}
	}
	return "", nil, fmt.Errorf("no compose command found (docker/podman/podman-compose)")
}

func (a *defaultRuntimeAdapter) ComposeCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	bin, prefix, err := a.ResolveComposeCommand(ctx)
	if err != nil {
		return nil, err
	}
	allArgs := append(prefix, args...)
	var workDir string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-f" {
			workDir = filepath.Dir(args[i+1])
			break
		}
	}
	var extraEnv []string
	if bin == "podman" && len(prefix) > 0 && prefix[0] == "compose" {
		extraEnv = append(extraEnv, "PODMAN_COMPOSE_WARNING_LOGS=false")
		if _, err := exec.LookPath("podman-compose"); err == nil {
			extraEnv = append(extraEnv, "PODMAN_COMPOSE_PROVIDER=podman-compose")
		} else if _, err := exec.LookPath("docker-compose"); err == nil {
			if runtime.GOOS == "darwin" {
				if home, e := os.UserHomeDir(); e == nil && home != "" {
					if _, se := os.Stat(filepath.Join(home, ".docker", "run", "docker.sock")); se != nil {
						return nil, fmt.Errorf("podman compose is using docker-compose provider but docker daemon socket is missing; install podman-compose or start docker daemon")
					}
				}
			}
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c := exec.CommandContext(ctx, bin, allArgs...)
	if workDir != "" {
		c.Dir = workDir
	}
	if len(extraEnv) > 0 {
		c.Env = append(os.Environ(), extraEnv...)
	}
	return c, nil
}

func podmanComposeAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "podman", "compose", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "no compose provider") ||
			strings.Contains(msg, "unknown command") ||
			strings.Contains(msg, "podman-compose") ||
			strings.Contains(msg, "docker-compose") {
			return false
		}
		if ctx.Err() != nil {
			return false
		}
	}
	return true
}

func dockerComposeAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "unknown command") ||
			strings.Contains(msg, "is not a docker command") ||
			strings.Contains(msg, "not found") ||
			strings.Contains(msg, "compose plugin") {
			return false
		}
		if ctx.Err() != nil {
			return false
		}
	}
	return true
}
