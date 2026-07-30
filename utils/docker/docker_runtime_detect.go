package docker

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

func normalizeHost(ctx context.Context, host string) string {
	h := strings.TrimSpace(host)
	if runtime.GOOS != "darwin" {
		return h
	}
	if strings.Contains(h, "/.local/share/containers/podman/machine/podman.sock") {
		if !unixSockExists(h) {
			if ph := darwinPodmanMachineHost(ctx); ph != "" {
				return ph
			}
		}
	}
	return h
}

func kindFromHost(host string) RuntimeKind {
	h := strings.ToLower(host)
	if strings.Contains(h, "podman") {
		return RuntimePodman
	}
	return RuntimeDocker
}

func IsPodmanRuntime(ctx context.Context) bool {
	return ResolveRuntime(ctx).Kind == RuntimePodman
}

func autoDetectUnixHost(ctx context.Context) string {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
	defer cancel()

	homeDir, _ := os.UserHomeDir()
	dockerHost := "unix:///var/run/docker.sock"

	if runtime.GOOS == "linux" {
		candidates := append(PodmanLinuxCandidateHosts(), dockerHost)
		for _, host := range candidates {
			if canPingHost(pingCtx, host) {
				return host
			}
		}
		for _, host := range candidates {
			if unixSockExists(host) {
				return host
			}
		}
		return ""
	}

	if runtime.GOOS == "darwin" {
		var candidates []string
		if ph := darwinPodmanMachineHost(pingCtx); ph != "" {
			candidates = append(candidates, ph)
		}
		if homeDir != "" {
			candidates = append(candidates,
				"unix://"+filepath.Join(homeDir, ".local/share/containers/podman/machine/podman.sock"),
				"unix://"+filepath.Join(homeDir, ".local/share/containers/podman/podman.sock"),
				"unix://"+filepath.Join(homeDir, ".config/containers/podman.sock"),
			)
		}
		candidates = append(candidates, dockerHost)
		for _, host := range candidates {
			if canPingHost(pingCtx, host) {
				return host
			}
		}
		return ""
	}

	candidates := []string{dockerHost}
	for _, host := range candidates {
		if canPingHost(pingCtx, host) {
			return host
		}
	}
	return ""
}

func detectKind(ctx context.Context, host string) RuntimeKind {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	detectCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return RuntimeDocker
	}
	defer cli.Close()

	ver, err := cli.ServerVersion(detectCtx)
	if err != nil {
		return RuntimeDocker
	}
	if strings.Contains(strings.ToLower(ver.Platform.Name), "podman") {
		return RuntimePodman
	}
	return RuntimeDocker
}

func canPingHost(ctx context.Context, host string) bool {
	if strings.HasPrefix(host, "unix://") {
		sockPath := strings.TrimPrefix(host, "unix://")
		if _, err := os.Stat(sockPath); err != nil {
			return false
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer cli.Close()

	_, err = cli.Ping(ctx)
	return err == nil
}

func CanPingHost(ctx context.Context, host string) bool {
	return canPingHost(ctx, host)
}

func PingHost(ctx context.Context, host string) error {
	if strings.HasPrefix(host, "unix://") {
		sockPath := strings.TrimPrefix(host, "unix://")
		if _, err := os.Stat(sockPath); err != nil {
			return err
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	_, err = cli.Ping(ctx)
	return err
}

func unixSockExists(host string) bool {
	if !strings.HasPrefix(host, "unix://") {
		return false
	}
	sockPath := strings.TrimPrefix(host, "unix://")
	_, err := os.Stat(sockPath)
	return err == nil
}

func darwinPodmanMachineHost(ctx context.Context) string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	podmanPath, err := runtimeBinaryPath("podman")
	if err != nil {
		return ""
	}
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ic, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
	defer cancel()

	out, err := exec.CommandContext(ic, podmanPath, "machine", "inspect").CombinedOutput()
	if err != nil {
		return ""
	}
	type inspectItem struct {
		ConnectionInfo struct {
			PodmanSocket struct {
				Path string `json:"Path"`
			} `json:"PodmanSocket"`
		} `json:"ConnectionInfo"`
	}
	var items []inspectItem
	if err := json.Unmarshal(out, &items); err != nil {
		return ""
	}
	for _, it := range items {
		p := strings.TrimSpace(it.ConnectionInfo.PodmanSocket.Path)
		if p == "" {
			continue
		}
		if filepath.IsAbs(p) {
			return "unix://" + p
		}
	}
	return ""
}
