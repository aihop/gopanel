package docker

import (
	"context"
	"encoding/json"
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

func commandName(bin string) string {
	name := filepath.Base(strings.TrimSpace(bin))
	return strings.ToLower(name)
}

func withPatchedPath(env []string, prependDirs ...string) []string {
	pathParts := make([]string, 0, len(prependDirs)+8)
	seen := map[string]struct{}{}
	appendPart := func(part string) {
		part = strings.TrimSpace(part)
		if part == "" {
			return
		}
		if _, ok := seen[part]; ok {
			return
		}
		seen[part] = struct{}{}
		pathParts = append(pathParts, part)
	}

	for _, dir := range prependDirs {
		appendPart(dir)
	}

	pathIndex := -1
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			pathIndex = i
			for _, part := range strings.Split(strings.TrimPrefix(kv, "PATH="), string(os.PathListSeparator)) {
				appendPart(part)
			}
			break
		}
	}

	for _, dir := range []string{"/opt/homebrew/bin", "/opt/homebrew/sbin", "/usr/local/bin", "/usr/local/sbin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		appendPart(dir)
	}

	pathValue := "PATH=" + strings.Join(pathParts, string(os.PathListSeparator))
	if pathIndex >= 0 {
		env[pathIndex] = pathValue
		return env
	}
	return append(env, pathValue)
}

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
	alt := "docker"
	if preferred == "docker" {
		alt = "podman"
	}

	if preferredPath, err := runtimeBinaryPath(preferred); err == nil {
		if preferred == "docker" && !dockerDaemonHealthy(ctx) {
			if altPath, aerr := runtimeBinaryPath(alt); aerr == nil {
				return altPath, nil
			}
		}
		return preferredPath, nil
	}
	if altPath, err := runtimeBinaryPath(alt); err == nil {
		return altPath, nil
	}
	return "", errors.New("container runtime cli not found")
}

func dockerDaemonHealthy(ctx context.Context) bool {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	hc, cancel := context.WithTimeout(baseCtx, 900*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(hc, "docker", "version", "--format", "{{.Server.Version}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(out))
	return s != "" && s != "<no value>"
}

func (a *defaultRuntimeAdapter) DockerClient(ctx context.Context) (*client.Client, error) {
	resolved := a.Resolve(ctx)
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if resolved.Host != "" {
		opts = append(opts, client.WithHost(resolved.Host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err == nil {
		return cli, nil
	}
	if resolved.Kind == RuntimePodman && runtime.GOOS == "darwin" {
		return podmanDarwinDockerClient(resolved.Host)
	}
	return nil, err
}

func (a *defaultRuntimeAdapter) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := a.CLI(ctx)
	if err != nil {
		return nil, err
	}
	if commandName(bin) == "podman" && runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = withPatchedPath(os.Environ(), filepath.Dir(bin))
	return cmd, nil
}

func (a *defaultRuntimeAdapter) CommandWithHost(ctx context.Context, host string, args ...string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := a.CLI(ctx)
	if err != nil {
		return nil, err
	}
	if commandName(bin) == "podman" && runtime.GOOS == "darwin" {
		_ = PodmanEnsureReady(ctx)
	}
	h := strings.TrimSpace(host)
	if h != "" && h != "podman-cli" && h != "podman://local" {
		if commandName(bin) == "podman" {
			args = append([]string{"--url", h}, args...)
		} else {
			args = append([]string{"-H", h}, args...)
		}
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = withPatchedPath(os.Environ(), filepath.Dir(bin))
	return cmd, nil
}

func (a *defaultRuntimeAdapter) ResolveComposeCommand(ctx context.Context) (string, []string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := a.Resolve(ctx)
	tryDocker := func() (string, []string, bool) {
		if dockerPath, err := runtimeBinaryPath("docker"); err == nil && dockerComposeAvailable() {
			return dockerPath, []string{"compose"}, true
		}
		return "", nil, false
	}
	tryPodman := func() (string, []string, bool) {
		if podmanPath, err := runtimeBinaryPath("podman"); err == nil && podmanComposeAvailable() {
			return podmanPath, []string{"compose"}, true
		}
		return "", nil, false
	}
	tryPodmanCompose := func() (string, []string, bool) {
		if podmanComposePath, err := runtimeBinaryPath("podman-compose"); err == nil {
			return podmanComposePath, nil, true
		}
		return "", nil, false
	}

	if resolved.Kind == RuntimePodman {
		if bin, prefix, ok := tryPodmanCompose(); ok {
			return bin, prefix, nil
		}
		if bin, prefix, ok := tryPodman(); ok {
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
	resolved := a.Resolve(ctx)
	bin, prefix, err := a.ResolveComposeCommand(ctx)
	if err != nil {
		return nil, err
	}
	binName := commandName(bin)
	allArgs := append(prefix, args...)
	var workDir string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-f" {
			workDir = filepath.Dir(args[i+1])
			break
		}
	}
	var extraEnv []string
	host := strings.TrimSpace(resolved.Host)
	if host != "" && host != "podman-cli" && host != "podman://local" {
		switch binName {
		case "podman":
			allArgs = append([]string{"--url", host}, allArgs...)
		case "docker":
			allArgs = append([]string{"-H", host}, allArgs...)
		case "podman-compose":
			// podman-compose 本身没有统一的 host flag，改走环境变量把目标 socket 传下去。
			extraEnv = append(extraEnv,
				"CONTAINER_HOST="+host,
				"DOCKER_HOST="+host,
				"PODMAN_HOST="+host,
			)
		}
	}
	if binName == "podman" && len(prefix) > 0 && prefix[0] == "compose" {
		extraEnv = append(extraEnv, "PODMAN_COMPOSE_WARNING_LOGS=false")
		if host != "" && host != "podman-cli" && host != "podman://local" {
			extraEnv = append(extraEnv,
				"CONTAINER_HOST="+host,
				"DOCKER_HOST="+host,
				"PODMAN_HOST="+host,
			)
		}
		if _, err := runtimeBinaryPath("podman-compose"); err == nil {
			extraEnv = append(extraEnv, "PODMAN_COMPOSE_PROVIDER=podman-compose")
		} else if _, err := runtimeBinaryPath("docker-compose"); err == nil {
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
	pathDirs := []string{filepath.Dir(bin)}
	if podmanPath, err := runtimeBinaryPath("podman"); err == nil {
		pathDirs = append(pathDirs, filepath.Dir(podmanPath))
	}
	if dockerPath, err := runtimeBinaryPath("docker"); err == nil {
		pathDirs = append(pathDirs, filepath.Dir(dockerPath))
	}
	c.Env = withPatchedPath(append(os.Environ(), extraEnv...), pathDirs...)
	return c, nil
}

func podmanComposeAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	podmanPath, err := runtimeBinaryPath("podman")
	if err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, podmanPath, "compose", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "no compose provider") ||
			strings.Contains(msg, "unknown command") ||
			strings.Contains(msg, "unrecognized command") ||
			strings.Contains(msg, "not a podman command") ||
			strings.Contains(msg, "unknown shorthand flag") ||
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

// podmanDarwinDockerClient creates a Docker client for Podman on macOS.
func podmanDarwinDockerClient(host string) (*client.Client, error) {
	if host != "" {
		cli, err := client.NewClientWithOpts(
			client.WithHost(host),
			client.WithAPIVersionNegotiation(),
		)
		if err == nil {
			return cli, nil
		}
	}

	cmd := exec.Command("podman", "machine", "inspect", "--format", "json")
	out, err := cmd.Output()
	if err == nil {
		var machines []struct {
			ConnectionInfo struct {
				PodmanSocket struct {
					Path string `json:"Path"`
				} `json:"PodmanSocket"`
			} `json:"ConnectionInfo"`
		}
		if err := json.Unmarshal(out, &machines); err == nil && len(machines) > 0 {
			if p := machines[0].ConnectionInfo.PodmanSocket.Path; p != "" {
				cli, err := client.NewClientWithOpts(
					client.WithHost("unix://"+p),
					client.WithAPIVersionNegotiation(),
				)
				if err == nil {
					return cli, nil
				}
			}
		}
	}

	return nil, nil
}
