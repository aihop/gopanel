package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
)

func splitLabelFilter(filter string) (string, string, bool) {
	s := strings.TrimSpace(filter)
	if s == "" {
		return "", "", false
	}
	if strings.Contains(s, "=") {
		parts := strings.SplitN(s, "=", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return s, "", true
}

func inspectPodman(req *dto.InspectReq) (string, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return "", err
	}
	args := make([]string, 0, 8)
	if host := strings.TrimSpace(req.RuntimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	switch req.Type {
	case "container":
		args = append(args, "container", "inspect", req.ID, "--format", "json")
	case "image":
		args = append(args, "image", "inspect", req.ID, "--format", "json")
	case "network":
		args = append(args, "network", "inspect", req.ID, "--format", "json")
	case "volume":
		args = append(args, "volume", "inspect", req.ID, "--format", "json")
	default:
		return "", errors.New("invalid inspect type")
	}
	c := exec.Command("podman", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", err
		}
		return "", errors.New(msg)
	}
	return string(out), nil
}

func containerStatsPodman(id string, runtimeHost string) (*dto.ContainerStats, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	args := make([]string, 0, 10)
	if host := strings.TrimSpace(runtimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	args = append(args, "stats", "--no-stream", "--format", "{{.ID}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}", id)
	c := exec.Command("podman", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, err
		}
		return nil, errors.New(msg)
	}
	lines := splitNonEmptyLines(string(out))
	if len(lines) == 0 {
		return nil, errors.New("no stats")
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) < 6 {
		return nil, errors.New("invalid stats output")
	}
	memUsage, _ := parseMemUsagePairToMB(parts[2])
	netRx, netTx, _ := parsePairToMB(parts[4])
	ioRead, ioWrite, _ := parsePairToMB(parts[5])
	return &dto.ContainerStats{
		CPUPercent: parsePercent(parts[1]),
		Memory:     memUsage,
		Cache:      0,
		IORead:     ioRead,
		IOWrite:    ioWrite,
		NetworkRX:  netRx,
		NetworkTX:  netTx,
		ShotTime:   time.Now(),
	}, nil
}

func loadContainerListStatPodmanCLI(containerID string) dto.ContainerListStats {
	c := exec.Command("podman", "stats", "--no-stream", "--format", "{{.ID}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}", containerID)
	out, err := c.CombinedOutput()
	if err != nil {
		return dto.ContainerListStats{ContainerID: containerID}
	}
	lines := splitNonEmptyLines(string(out))
	if len(lines) == 0 {
		return dto.ContainerListStats{ContainerID: containerID}
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) < 4 {
		return dto.ContainerListStats{ContainerID: containerID}
	}
	memUsage, _ := parseMemUsagePairToMB(parts[2])
	return dto.ContainerListStats{
		ContainerID:   strings.TrimSpace(containerID),
		CPUPercent:    parsePercent(parts[1]),
		MemoryPercent: parsePercent(parts[3]),
		MemoryUsage:   uint64(memUsage * 1024 * 1024),
	}
}

func podmanLinuxContainerHostMaps(ctx context.Context) (map[string]string, map[string]string, error) {
	list, source, err := listContainersMergedByHostWithSource(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, nil, err
	}
	idToHost := make(map[string]string)
	nameToHost := make(map[string]string)
	for _, it := range list {
		host := strings.TrimSpace(source[it.ID])
		if host != "" {
			idToHost[it.ID] = host
		}
		for _, n := range it.Names {
			name := strings.TrimPrefix(strings.TrimSpace(n), "/")
			if name == "" {
				continue
			}
			if _, ok := nameToHost[name]; ok {
				continue
			}
			nameToHost[name] = host
		}
	}
	return idToHost, nameToHost, nil
}

func resolveLinuxPodmanContainerHost(ctx context.Context, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	idToHost, nameToHost, err := podmanLinuxContainerHostMaps(ctx)
	if err != nil {
		return "", err
	}
	if host := strings.TrimSpace(idToHost[key]); host != "" {
		return host, nil
	}
	if host := strings.TrimSpace(nameToHost[key]); host != "" {
		return host, nil
	}
	return "", nil
}

func containerInfoPodman(req *dto.OperationWithName) (*dto.ContainerOperate, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	args := make([]string, 0, 8)
	if host := strings.TrimSpace(req.RuntimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	args = append(args, "container", "inspect", req.Name, "--format", "json")
	out, err := exec.Command("podman", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, err
		}
		return nil, errors.New(msg)
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("container not found")
	}
	m := items[0]

	var data dto.ContainerOperate
	data.ContainerID = strings.TrimSpace(fmt.Sprint(m["Id"]))
	if data.ContainerID == "" {
		data.ContainerID = strings.TrimSpace(fmt.Sprint(m["ID"]))
	}
	data.Name = strings.TrimPrefix(strings.TrimSpace(fmt.Sprint(m["Name"])), "/")
	if data.Name == "" {
		data.Name = strings.TrimSpace(req.Name)
	}

	if cfg, ok := m["Config"].(map[string]interface{}); ok {
		data.Image = strings.TrimSpace(fmt.Sprint(cfg["Image"]))
		data.Cmd = toStringSlice(cfg["Cmd"])
		data.Entrypoint = toStringSlice(cfg["Entrypoint"])
		data.Env = toStringSlice(cfg["Env"])
		data.OpenStdin = toBool(cfg["OpenStdin"])
		data.Tty = toBool(cfg["Tty"])
		if labels, ok := cfg["Labels"].(map[string]interface{}); ok {
			for k, v := range labels {
				data.Labels = append(data.Labels, fmt.Sprintf("%s=%v", k, v))
			}
		}
	}

	if hostCfg, ok := m["HostConfig"].(map[string]interface{}); ok {
		data.AutoRemove = toBool(hostCfg["AutoRemove"])
		data.Privileged = toBool(hostCfg["Privileged"])
		data.PublishAllPorts = toBool(hostCfg["PublishAllPorts"])
		if rp, ok := hostCfg["RestartPolicy"].(map[string]interface{}); ok {
			data.RestartPolicy = strings.TrimSpace(fmt.Sprint(rp["Name"]))
		} else {
			data.RestartPolicy = strings.TrimSpace(fmt.Sprint(hostCfg["RestartPolicy"]))
		}
		if n := toInt64(hostCfg["NanoCpus"]); n != 0 {
			data.NanoCPUs = float64(n) / 1000000000
		}
		if n := toInt64(hostCfg["NanoCPUs"]); n != 0 {
			data.NanoCPUs = float64(n) / 1000000000
		}
		data.CPUShares = toInt64(hostCfg["CpuShares"])
		if data.CPUShares == 0 {
			data.CPUShares = toInt64(hostCfg["CPUShares"])
		}
		if mem := toInt64(hostCfg["Memory"]); mem != 0 {
			data.Memory = float64(mem) / 1024 / 1024
		}
	}

	if net, ok := m["NetworkSettings"].(map[string]interface{}); ok {
		if networks, ok := net["Networks"].(map[string]interface{}); ok {
			for name, v := range networks {
				data.Network = name
				if nm, ok := v.(map[string]interface{}); ok {
					data.Ipv4 = strings.TrimSpace(fmt.Sprint(nm["IPAddress"]))
					data.Ipv6 = strings.TrimSpace(fmt.Sprint(nm["GlobalIPv6Address"]))
				}
				break
			}
		}
	}

	data.ExposedPorts = podmanPortsForInfo(req.Name, req.RuntimeHost)
	data.Volumes = podmanMountsForInfo(m)
	return &data, nil
}

func podmanPortsForInfo(container string, runtimeHost string) []dto.PortHelper {
	args := make([]string, 0, 6)
	if host := strings.TrimSpace(runtimeHost); strings.HasPrefix(host, "unix://") {
		args = append(args, "--url", host)
	}
	args = append(args, "port", container)
	out, err := exec.Command("podman", args...).CombinedOutput()
	if err != nil {
		return nil
	}
	lines := splitNonEmptyLines(string(out))
	var res []dto.PortHelper
	for _, line := range lines {
		parts := strings.Split(line, "->")
		if len(parts) != 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		cp := ""
		proto := "tcp"
		if strings.Contains(left, "/") {
			lp := strings.SplitN(left, "/", 2)
			cp = strings.TrimSpace(lp[0])
			proto = strings.TrimSpace(lp[1])
		} else {
			cp = left
		}

		hostIP := ""
		hostPort := ""
		if strings.Contains(right, ":") {
			rp := strings.SplitN(right, ":", 2)
			hostIP = strings.TrimSpace(rp[0])
			hostPort = strings.TrimSpace(rp[1])
		} else {
			hostPort = right
		}

		res = append(res, dto.PortHelper{
			HostIP:        hostIP,
			HostPort:      hostPort,
			ContainerPort: cp,
			Protocol:      proto,
		})
	}
	return res
}

func podmanMountsForInfo(inspect map[string]interface{}) []dto.VolumeHelper {
	mountsAny, ok := inspect["Mounts"]
	if !ok || mountsAny == nil {
		return nil
	}
	var res []dto.VolumeHelper
	switch mounts := mountsAny.(type) {
	case []interface{}:
		for _, it := range mounts {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			res = append(res, dto.VolumeHelper{
				Type:         strings.TrimSpace(fmt.Sprint(m["Type"])),
				SourceDir:    strings.TrimSpace(fmt.Sprint(m["Source"])),
				ContainerDir: strings.TrimSpace(fmt.Sprint(m["Destination"])),
				Mode:         strings.TrimSpace(fmt.Sprint(m["Mode"])),
			})
		}
	}
	return res
}

func containerOperationPodmanCLI(ctx context.Context, runtimeHost string, names []string, op string) error {
	if err := docker.PodmanEnsureReady(ctx); err != nil {
		return err
	}
	host := strings.TrimSpace(runtimeHost)
	for _, item := range names {
		global.LOG.Infof("start container %s operation %s", item, op)
		var args []string
		switch op {
		case constant.ContainerOpStart:
			args = []string{"start", item}
		case constant.ContainerOpStop:
			args = []string{"stop", item}
		case constant.ContainerOpRestart:
			args = []string{"restart", item}
		case constant.ContainerOpKill:
			args = []string{"kill", "-s", "SIGKILL", item}
		case constant.ContainerOpPause:
			args = []string{"pause", item}
		case constant.ContainerOpUnpause:
			args = []string{"unpause", item}
		case constant.ContainerOpRemove:
			args = []string{"rm", "-f", "-v", item}
		default:
			return errors.New("operation is empty")
		}
		c, err := docker.RuntimeCommandWithHost(ctx, host, args...)
		if err != nil {
			return err
		}
		out, err := c.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				return err
			}
			return errors.New(msg)
		}
	}
	return nil
}
