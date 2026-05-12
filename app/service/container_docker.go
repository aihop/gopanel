package service

import (
	"context"
	"encoding/json"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
	"os"
	"runtime"
	"strings"
	"time"
)

type DockerService struct{}
type IDockerService interface {
	UpdateConf(req dto.SettingUpdate) error
	UpdateLogOption(req dto.LogOption) error
	UpdateIpv6Option(req dto.Ipv6Option) error
	UpdateConfByFile(info dto.DaemonJsonUpdateByFile) error
	LoadDockerStatus() string
	LoadDockerConf() *dto.DaemonJsonConf
	OperateDocker(req dto.DockerOperation) error
}

func NewIDockerService() IDockerService {
	return &DockerService{}
}

type daemonJsonItem struct {
	Status       string    `json:"status"`
	Mirrors      []string  `json:"registry-mirrors"`
	Registries   []string  `json:"insecure-registries"`
	LiveRestore  bool      `json:"live-restore"`
	Ipv6         bool      `json:"ipv6"`
	FixedCidrV6  string    `json:"fixed-cidr-v6"`
	Ip6Tables    bool      `json:"ip6tables"`
	Experimental bool      `json:"experimental"`
	IPTables     bool      `json:"iptables"`
	ExecOpts     []string  `json:"exec-opts"`
	LogOption    logOption `json:"log-opts"`
}
type logOption struct {
	LogMaxSize string `json:"max-size"`
	LogMaxFile string `json:"max-file"`
}

func (u *DockerService) LoadDockerStatus() string {
	ctx := context.Background()
	resolved := docker.ResolveRuntime(ctx)
	if resolved.Kind == docker.RuntimePodman {
		if runtime.GOOS == "darwin" {
			if err := docker.PodmanEnsureReady(ctx); err != nil {
				return constant.Stopped
			}
			return constant.StatusRunning
		}
		serviceActive := podmanServiceActiveForHost(ctx, resolved.Host)
		pingCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
		defer cancel()
		if docker.CanPingHost(pingCtx, resolved.Host) {
			return constant.StatusRunning
		}
		if serviceActive {
			// service active 仅表示 socket/unit 存在，不代表当前 runtime host API 已可用。
			// 这里做一次更长超时的复检，避免出现“显示 Running 但实际 API 不通”。
			retryCtx, retryCancel := context.WithTimeout(ctx, 1800*time.Millisecond)
			defer retryCancel()
			if docker.CanPingHost(retryCtx, resolved.Host) {
				return constant.StatusRunning
			}
		}
		return constant.Stopped
	}
	client, err := docker.NewRuntimeAPIClient()
	if err != nil {
		return constant.Stopped
	}
	defer client.Close()
	if _, err := client.Ping(context.Background()); err != nil {
		return constant.Stopped
	}
	return constant.StatusRunning
}
func (u *DockerService) LoadDockerConf() *dto.DaemonJsonConf {
	ctx := context.Background()
	var data dto.DaemonJsonConf
	data.IPTables = true
	data.ContainerType = "docker"
	data.RuntimeKind = "docker"
	data.Status = constant.Stopped
	data.Version = "-"
	resolved := docker.ResolveRuntime(ctx)
	data.RuntimeKind = string(resolved.Kind)
	data.RuntimeHost = resolved.Host
	data.ConfiguredHost = docker.ConfiguredDockerSockPath()
	data.HostPinned = strings.TrimSpace(data.ConfiguredHost) != ""
	data.RootlessHost = docker.IsRootlessPodmanHost(resolved.Host)
	data.Capabilities = dto.RuntimeCapabilities{DaemonJson: resolved.Kind == docker.RuntimeDocker && runtime.GOOS == "linux", DockerAPI: !(resolved.Kind == docker.RuntimePodman && runtime.GOOS == "darwin"), PodmanCLI: resolved.Kind == docker.RuntimePodman, PodmanRegistriesConf: resolved.Kind == docker.RuntimePodman && (runtime.GOOS == "linux" || runtime.GOOS == "darwin"), Compose: composeAvailable()}
	if resolved.Kind == docker.RuntimePodman {
		data.ContainerType = "podman"
		if runtime.GOOS == "darwin" {
			if err := docker.PodmanEnsureReady(ctx); err == nil {
				data.Status = constant.StatusRunning
				data.ServiceActive = true
				if mirrors, err := podmanMachineRegistriesGet(ctx); err == nil {
					data.Mirrors = append(data.Mirrors, mirrors...)
				} else {
					data.Capabilities.PodmanRegistriesConf = false
				}
			} else {
				data.Capabilities.PodmanRegistriesConf = false
			}
			if v, err := docker.PodmanVersion(ctx); err == nil && strings.TrimSpace(v) != "" {
				data.Version = v
			}
			data.ApiReady = false
		} else {
			data.RootlessHost = podmanRootlessExpected(resolved.Host)
			data.ServiceActive = podmanServiceActiveForHost(ctx, resolved.Host)
			pingCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
			defer cancel()
			data.ApiReady = docker.CanPingHost(pingCtx, resolved.Host)
			if !data.ApiReady && data.ServiceActive {
				// socket/unit active 但 API 可能需要按需激活，给一次更长的复检窗口。
				retryCtx, retryCancel := context.WithTimeout(ctx, 1800*time.Millisecond)
				data.ApiReady = docker.CanPingHost(retryCtx, resolved.Host)
				retryCancel()
			}
			if data.ApiReady {
				client, err := docker.NewRuntimeAPIClient()
				if err == nil {
					defer client.Close()
					data.Status = constant.StatusRunning
					itemVersion, err := client.ServerVersion(ctx)
					if err == nil {
						data.Version = itemVersion.Version
					}
				}
			}
			if data.Status != constant.StatusRunning && data.ServiceActive && data.ApiReady {
				data.Status = constant.StatusRunning
			}
			if runtime.GOOS == "linux" {
				var params map[string]interface{}
				if home := podmanRootlessHomeFromHost(resolved.Host); home != "" {
					params = map[string]interface{}{"home": home}
				}
				out, err := gpc.Do(ctx, "PODMAN_REGISTRIES_GET", params)
				if err != nil {
					data.Capabilities.PodmanRegistriesConf = false
				} else {
					var pmRes map[string]interface{}
					if json.Unmarshal([]byte(out.Output), &pmRes) == nil {
						if m, ok := pmRes["mirrors"].([]interface{}); ok {
							for _, v := range m {
								if s, ok := v.(string); ok {
									data.Mirrors = append(data.Mirrors, s)
								}
							}
						}
					}
				}
			}
		}
		return &data
	}
	client, err := docker.NewRuntimeAPIClient()
	if err != nil {
		data.Status = constant.Stopped
	} else {
		defer client.Close()
		if _, err := client.Ping(ctx); err != nil {
			data.Status = constant.Stopped
		} else {
			data.Status = constant.StatusRunning
		}
		itemVersion, err := client.ServerVersion(ctx)
		if err == nil {
			data.Version = itemVersion.Version
		}
		info, err := client.Info(ctx)
		if err == nil {
			data.IsSwarm = strings.ToLower(string(info.Swarm.LocalNodeState)) == "active"
		}
	}
	data.ServiceActive = data.Status == constant.StatusRunning
	data.ApiReady = data.Status == constant.StatusRunning
	if runtime.GOOS != "linux" {
		return &data
	}
	if _, err := os.Stat(constant.DaemonJsonPath); err != nil {
		return &data
	}
	file, err := os.ReadFile(constant.DaemonJsonPath)
	if err != nil {
		return &data
	}
	var conf daemonJsonItem
	daemonMap := make(map[string]interface{})
	if err := json.Unmarshal(file, &daemonMap); err != nil {
		return &data
	}
	arr, err := json.Marshal(daemonMap)
	if err != nil {
		return &data
	}
	if err := json.Unmarshal(arr, &conf); err != nil {
		return &data
	}
	if _, ok := daemonMap["iptables"]; !ok {
		conf.IPTables = true
	}
	data.CgroupDriver = "cgroupfs"
	for _, opt := range conf.ExecOpts {
		if strings.HasPrefix(opt, "native.cgroupdriver=") {
			data.CgroupDriver = strings.ReplaceAll(opt, "native.cgroupdriver=", "")
			break
		}
	}
	data.Ipv6 = conf.Ipv6
	data.FixedCidrV6 = conf.FixedCidrV6
	data.Ip6Tables = conf.Ip6Tables
	data.Experimental = conf.Experimental
	data.LogMaxSize = conf.LogOption.LogMaxSize
	data.LogMaxFile = conf.LogOption.LogMaxFile
	data.Mirrors = conf.Mirrors
	data.Registries = conf.Registries
	data.IPTables = conf.IPTables
	data.LiveRestore = conf.LiveRestore
	return &data
}
