package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime" // 添加 runtime 包用于获取操作系统信息
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

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

func NewRuntimeClient() (Client, error) {
	cli, err := DefaultRuntimeAdapter().DockerClient(context.Background())
	if err != nil {
		return Client{}, err
	}

	return Client{
		cli: cli,
	}, nil
}

func NewClient() (Client, error) {
	return NewRuntimeClient()
}

func (c Client) Close() {
	_ = c.cli.Close()
}

func NewRuntimeAPIClient() (*client.Client, error) {
	return DefaultRuntimeAdapter().DockerClient(context.Background())
}

func NewDockerClient() (*client.Client, error) {
	return NewRuntimeAPIClient()
}

func (c Client) ListContainersByName(names []string) ([]types.Container, error) {
	var (
		options  container.ListOptions
		namesMap = make(map[string]bool)
		res      []types.Container
	)
	options.All = true
	if len(names) > 0 {
		var array []filters.KeyValuePair
		for _, n := range names {
			namesMap["/"+n] = true
			array = append(array, filters.Arg("name", n))
		}
		options.Filters = filters.NewArgs(array...)
	}
	containers, err := c.cli.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}
	for _, con := range containers {
		if _, ok := namesMap[con.Names[0]]; ok {
			res = append(res, con)
		}
	}
	return res, nil
}
func (c Client) ListAllContainers() ([]types.Container, error) {
	var (
		options container.ListOptions
	)
	options.All = true
	containers, err := c.cli.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (c Client) CreateNetwork(name string) error {
	_, err := c.cli.NetworkCreate(context.Background(), name, network.CreateOptions{
		Driver: "bridge",
	})
	return err
}

func (c Client) DeleteImage(imageID string) error {
	if _, err := c.cli.ImageRemove(context.Background(), imageID, image.RemoveOptions{Force: true}); err != nil {
		return err
	}
	return nil
}

func (c Client) InspectContainer(containerID string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(context.Background(), containerID)
}

func (c Client) PullImage(imageName string, force bool) error {
	if !force {
		exist, err := c.CheckImageExist(imageName)
		if err != nil {
			return err
		}
		if exist {
			return nil
		}
	}
	if _, err := c.cli.ImagePull(context.Background(), imageName, image.PullOptions{}); err != nil {
		return err
	}
	return nil
}

func (c Client) GetImageIDByName(imageName string) (string, error) {
	filter := filters.NewArgs()
	filter.Add("reference", imageName)
	list, err := c.cli.ImageList(context.Background(), image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return "", err
	}
	if len(list) > 0 {
		return list[0].ID, nil
	}
	return "", nil
}

func (c Client) CheckImageExist(imageName string) (bool, error) {
	filter := filters.NewArgs()
	filter.Add("reference", imageName)
	list, err := c.cli.ImageList(context.Background(), image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return false, err
	}
	return len(list) > 0, nil
}

func (c Client) NetworkExist(name string) bool {
	var options network.ListOptions
	options.Filters = filters.NewArgs(filters.Arg("name", name))
	networks, err := c.cli.NetworkList(context.Background(), options)
	if err != nil {
		return false
	}
	return len(networks) > 0
}

func CreateDefaultDockerNetwork() error {
	resolved := ResolveRuntime(context.Background())
	if resolved.Kind == RuntimePodman {
		if err := ensurePodmanNetwork("gopanel-network"); err == nil {
			return nil
		} else {
			cli, cerr := NewClient()
			if cerr == nil {
				defer cli.Close()
				if !cli.NetworkExist("gopanel-network") {
					if nerr := cli.CreateNetwork("gopanel-network"); nerr == nil {
						return nil
					}
				}
			}
			global.LOG.Errorf("create default podman network error %s", err.Error())
			return err
		}
	}
	cli, err := NewClient()
	if err != nil {
		global.LOG.Errorf("init docker client error %s", err.Error())
		return err
	}
	defer cli.Close()
	if !cli.NetworkExist("gopanel-network") {
		if err := cli.CreateNetwork("gopanel-network"); err != nil {
			global.LOG.Errorf("create default docker network  error %s", err.Error())
			return err
		}
	}
	return nil
}

func EnsureNetwork(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	resolved := ResolveRuntime(context.Background())
	if resolved.Kind == RuntimePodman {
		if err := ensurePodmanNetwork(name); err == nil {
			return nil
		} else {
			cli, cerr := NewClient()
			if cerr == nil {
				defer cli.Close()
				if !cli.NetworkExist(name) {
					if nerr := cli.CreateNetwork(name); nerr == nil {
						return nil
					}
				}
			}
			return err
		}
	}
	cli, err := NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	if !cli.NetworkExist(name) {
		return cli.CreateNetwork(name)
	}
	return nil
}

func ensurePodmanNetwork(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	ctx := context.Background()
	if err := PodmanEnsureReady(ctx); err != nil {
		return err
	}
	if _, err := runPodman(ctx, "network", "inspect", name); err == nil {
		return nil
	}
	out, err := runPodman(ctx, "network", "create", name)
	if err != nil {
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
		}
		return err
	}
	return nil
}
