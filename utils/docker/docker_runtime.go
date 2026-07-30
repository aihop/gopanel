package docker

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/docker/docker/api/types/container"
)

type RuntimeKind string

const (
	RuntimeDocker RuntimeKind = "docker"
	RuntimePodman RuntimeKind = "podman"
)

type ResolvedRuntime struct {
	Kind RuntimeKind
	Host string
}

func ConfiguredDockerSockPath() string {
	var settingItem model.Setting
	_ = global.DB.Where("key = ?", "DockerSockPath").First(&settingItem).Error
	return strings.TrimSpace(settingItem.Value)
}

func RuntimeHostPinned() bool {
	return ConfiguredDockerSockPath() != ""
}

func strictCurrentUserRootlessPodman() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Geteuid() == 0 {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(global.CONF.System.ContainerRuntime))
	return mode == "podman"
}

func StrictCurrentUserRootlessPodman() bool {
	return strictCurrentUserRootlessPodman()
}

func ResolveRuntime(ctx context.Context) ResolvedRuntime {
	mode := strings.ToLower(strings.TrimSpace(global.CONF.System.ContainerRuntime))
	if mode == "" {
		mode = "auto"
	}
	strictRootless := strictCurrentUserRootlessPodman()

	var settingItem model.Setting
	_ = global.DB.Where("key = ?", "DockerSockPath").First(&settingItem).Error
	if len(settingItem.Value) > 0 && mode == "auto" {
		host := normalizeHost(ctx, settingItem.Value)
		baseCtx := ctx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
		defer cancel()

		if host != "" && !canPingHost(pingCtx, host) {
			if kindFromHost(host) == RuntimePodman {
				candidates := PodmanLinuxCandidateHosts()
				for _, c := range candidates {
					if canPingHost(pingCtx, c) {
						host = c
						break
					}
				}
				if host == normalizeHost(ctx, settingItem.Value) {
					for _, c := range candidates {
						if unixSockExists(c) {
							host = c
							break
						}
					}
					if host == normalizeHost(ctx, settingItem.Value) {
						host = ""
					}
				}
			} else {
				host = ""
			}
		}
		if host != settingItem.Value {
			_ = global.DB.Model(&model.Setting{}).Where("key = ?", "DockerSockPath").Update("value", host).Error
		}
		if host == "" {
			goto auto_detect
		}
		k := kindFromHost(host)
		if k == RuntimeDocker {
			k = detectKind(ctx, host)
		}
		return ResolvedRuntime{
			Kind: k,
			Host: host,
		}
	}

auto_detect:
	if mode == "docker" {
		host := strings.TrimSpace(settingItem.Value)
		if host != "" {
			host = normalizeHost(ctx, host)
			if kindFromHost(host) != RuntimeDocker {
				host = ""
			}
		}
		switch runtime.GOOS {
		case "windows":
			if host == "" {
				host = "npipe:////./pipe/docker_engine"
			}
			return ResolvedRuntime{Kind: RuntimeDocker, Host: host}
		default:
			if host == "" {
				host = autoDetectDockerUnixHost(ctx)
			}
			if host == "" {
				host = "unix:///var/run/docker.sock"
			}
			return ResolvedRuntime{Kind: RuntimeDocker, Host: host}
		}
	}

	if mode == "podman" {
		switch runtime.GOOS {
		case "linux":
			host := strings.TrimSpace(settingItem.Value)
			if host != "" {
				host = normalizeHost(ctx, host)
				if kindFromHost(host) != RuntimePodman {
					host = ""
				}
				if strictRootless && host != "" && !IsRootlessPodmanHost(host) {
					host = ""
				}
			}
			baseCtx := ctx
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
			defer cancel()
			candidates := PodmanLinuxCandidateHosts()
			if strictRootless {
				candidates = PodmanLinuxUserCandidateHosts()
			}

			if host == "" {
				for _, c := range candidates {
					if canPingHost(pingCtx, c) {
						host = c
						break
					}
				}
				if host == "" && len(candidates) > 0 {
					host = candidates[0]
				}
			} else if IsRootlessPodmanHost(host) || os.Getuid() != 0 {
				if !canPingHost(pingCtx, host) {
					for _, c := range candidates {
						if canPingHost(pingCtx, c) {
							host = c
							break
						}
					}
				}
			}
			return ResolvedRuntime{Kind: RuntimePodman, Host: host}
		case "darwin":
			host := strings.TrimSpace(settingItem.Value)
			if host != "" {
				host = normalizeHost(ctx, host)
				if kindFromHost(host) != RuntimePodman {
					host = ""
				}
			}
			if host == "" {
				host = darwinPodmanMachineHost(ctx)
			}
			if host == "" {
				host = "podman://local"
			}
			return ResolvedRuntime{Kind: RuntimePodman, Host: host}
		default:
			return ResolvedRuntime{Kind: RuntimePodman, Host: "podman://local"}
		}
	}

	switch runtime.GOOS {
	case "windows":
		host := "npipe:////./pipe/docker_engine"
		return ResolvedRuntime{Kind: kindFromHost(host), Host: host}
	default:
		host := autoDetectUnixHost(ctx)
		if host == "" {
			host = "unix:///var/run/docker.sock"
		}
		k := kindFromHost(host)
		if k == RuntimeDocker {
			k = detectKind(ctx, host)
		}
		return ResolvedRuntime{Kind: k, Host: host}
	}
}

func autoDetectDockerUnixHost(ctx context.Context) string {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
	defer cancel()

	dockerHost := "unix:///var/run/docker.sock"
	if canPingHost(pingCtx, dockerHost) {
		return dockerHost
	}
	return ""
}

func PodmanLinuxCandidateHosts() []string {
	return podmanLinuxCandidateHosts(true)
}

func PodmanLinuxUserCandidateHosts() []string {
	return podmanLinuxCandidateHosts(false)
}

func podmanLinuxCandidateHosts(includeSystem bool) []string {
	if runtime.GOOS != "linux" {
		return nil
	}

	uid := os.Getuid()
	seen := make(map[string]struct{})
	var hosts []string

	addSock := func(sockPath string) {
		sockPath = strings.TrimSpace(sockPath)
		if sockPath == "" {
			return
		}
		if !filepath.IsAbs(sockPath) {
			return
		}
		host := "unix://" + sockPath
		if _, ok := seen[host]; ok {
			return
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}

	if uid != 0 {
		if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
			addSock(filepath.Join(runtimeDir, "podman", "podman.sock"))
		}
		addSock(filepath.Join("/run/user", strconv.Itoa(uid), "podman", "podman.sock"))
	}

	if includeSystem {
		addSock("/run/podman/podman.sock")
	}
	return hosts
}

func IsRootlessPodmanHost(host string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	host = strings.TrimSpace(host)
	if !strings.HasPrefix(host, "unix://") {
		return false
	}
	sockPath := strings.TrimPrefix(host, "unix://")
	if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
		if strings.HasPrefix(sockPath, filepath.Clean(runtimeDir)+string(os.PathSeparator)) {
			return true
		}
	}
	return strings.Contains(sockPath, "/run/user/") && strings.HasSuffix(sockPath, "/podman/podman.sock")
}

func EnsureContainerLogConfig(hostConf *container.HostConfig) {
	if hostConf == nil || runtime.GOOS != "linux" {
		return
	}
	resolved := ResolveRuntime(context.Background())
	if resolved.Kind != RuntimePodman || !IsRootlessPodmanHost(resolved.Host) {
		return
	}
	current := strings.TrimSpace(hostConf.LogConfig.Type)
	if current != "" && !strings.EqualFold(current, "journald") {
		return
	}
	hostConf.LogConfig.Type = "k8s-file"
	if hostConf.LogConfig.Config == nil {
		hostConf.LogConfig.Config = map[string]string{}
	}
}
