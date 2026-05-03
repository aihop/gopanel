package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func loadVolumeBinds(binds []types.MountPoint) []dto.VolumeHelper {
	var datas []dto.VolumeHelper
	for _, bind := range binds {
		var volumeItem dto.VolumeHelper
		volumeItem.Type = string(bind.Type)
		if bind.Type == "volume" {
			volumeItem.SourceDir = bind.Name
		} else {
			volumeItem.SourceDir = bind.Source
		}
		volumeItem.ContainerDir = bind.Destination
		volumeItem.Mode = "ro"
		if bind.RW {
			volumeItem.Mode = "rw"
		}
		datas = append(datas, volumeItem)
	}
	return datas
}
func loadPortByInspect(id string, client *client.Client) ([]types.Port, error) {
	container, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}
	var itemPorts []types.Port
	portBindings := container.NetworkSettings.Ports
	if len(portBindings) == 0 && container.ContainerJSONBase != nil && container.ContainerJSONBase.HostConfig != nil {
		portBindings = container.ContainerJSONBase.HostConfig.PortBindings
	}
	for key, val := range portBindings {
		if !strings.Contains(string(key), "/") {
			continue
		}
		item := strings.Split(string(key), "/")
		itemPort, _ := strconv.ParseUint(item[0], 10, 16)
		for _, itemVal := range val {
			publicPort, _ := strconv.ParseUint(itemVal.HostPort, 10, 16)
			itemPorts = append(itemPorts, types.Port{PrivatePort: uint16(itemPort), Type: item[1], PublicPort: uint16(publicPort), IP: itemVal.HostIP})
		}
	}
	return itemPorts, nil
}
func loadContainerPortForInfo(itemPorts []types.Port) []dto.PortHelper {
	var exposedPorts []dto.PortHelper
	samePortMap := make(map[string]dto.PortHelper)
	ports := transPortToStr(itemPorts)
	for _, item := range ports {
		itemStr := strings.Split(item, "->")
		if len(itemStr) < 2 {
			continue
		}
		var itemPort dto.PortHelper
		lastIndex := strings.LastIndex(itemStr[0], ":")
		if lastIndex == -1 {
			itemPort.HostPort = itemStr[0]
		} else {
			itemPort.HostIP = itemStr[0][0:lastIndex]
			itemPort.HostPort = itemStr[0][lastIndex+1:]
		}
		itemContainer := strings.Split(itemStr[1], "/")
		if len(itemContainer) != 2 {
			continue
		}
		itemPort.ContainerPort = itemContainer[0]
		itemPort.Protocol = itemContainer[1]
		keyItem := fmt.Sprintf("%s->%s/%s", itemPort.HostPort, itemPort.ContainerPort, itemPort.Protocol)
		if val, ok := samePortMap[keyItem]; ok {
			val.HostIP = ""
			samePortMap[keyItem] = val
		} else {
			samePortMap[keyItem] = itemPort
		}
	}
	for _, val := range samePortMap {
		exposedPorts = append(exposedPorts, val)
	}
	return exposedPorts
}
func transPortToStr(ports []types.Port) []string {
	var (
		ipv4Ports []types.Port
		ipv6Ports []types.Port
	)
	for _, port := range ports {
		if strings.Contains(port.IP, ":") {
			ipv6Ports = append(ipv6Ports, port)
		} else {
			ipv4Ports = append(ipv4Ports, port)
		}
	}
	list1 := simplifyPort(ipv4Ports)
	list2 := simplifyPort(ipv6Ports)
	return append(list1, list2...)
}
func simplifyPort(ports []types.Port) []string {
	var datas []string
	if len(ports) == 0 {
		return datas
	}
	if len(ports) == 1 {
		ip := ""
		if len(ports[0].IP) != 0 {
			ip = ports[0].IP + ":"
		}
		itemPortStr := fmt.Sprintf("%s%v/%s", ip, ports[0].PrivatePort, ports[0].Type)
		if ports[0].PublicPort != 0 {
			itemPortStr = fmt.Sprintf("%s%v->%v/%s", ip, ports[0].PublicPort, ports[0].PrivatePort, ports[0].Type)
		}
		datas = append(datas, itemPortStr)
		return datas
	}
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].PrivatePort < ports[j].PrivatePort
	})
	start := ports[0]
	for i := 1; i < len(ports); i++ {
		if ports[i].PrivatePort != ports[i-1].PrivatePort+1 || ports[i].IP != ports[i-1].IP || ports[i].PublicPort != ports[i-1].PublicPort+1 || ports[i].Type != ports[i-1].Type {
			if ports[i-1].PrivatePort == start.PrivatePort {
				itemPortStr := fmt.Sprintf("%s:%v/%s", start.IP, start.PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v->%v/%s", start.IP, start.PublicPort, start.PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			} else {
				itemPortStr := fmt.Sprintf("%s:%v-%v/%s", start.IP, start.PrivatePort, ports[i-1].PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v-%v->%v-%v/%s", start.IP, start.PublicPort, ports[i-1].PublicPort, start.PrivatePort, ports[i-1].PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			}
			start = ports[i]
		}
		if i == len(ports)-1 {
			if ports[i].PrivatePort == start.PrivatePort {
				itemPortStr := fmt.Sprintf("%s:%v/%s", start.IP, start.PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v->%v/%s", start.IP, start.PublicPort, start.PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			} else {
				itemPortStr := fmt.Sprintf("%s:%v-%v/%s", start.IP, start.PrivatePort, ports[i].PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v-%v->%v-%v/%s", start.IP, start.PublicPort, ports[i].PublicPort, start.PrivatePort, ports[i].PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			}
		}
	}
	return datas
}
func normalizeContainerIPList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		ip := strings.TrimSpace(item)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}
func inferContainerRuntimeMode(runtimeKind string, runtimeHost string) string {
	host := strings.TrimSpace(runtimeHost)
	if runtime.GOOS != "linux" {
		return "default"
	}
	if host != "" && strings.Contains(host, "/run/user/") {
		return "rootless"
	}
	switch strings.TrimSpace(runtimeKind) {
	case string(docker.RuntimeDocker), string(docker.RuntimePodman):
		return "rootful"
	default:
		return "default"
	}
}
func inspectContainerIPsForList(ctx context.Context, dockerClient *client.Client, containerID string, runtimeHost string) []string {
	if dockerClient != nil {
		inspect, err := dockerClient.ContainerInspect(ctx, containerID)
		if err == nil && inspect.NetworkSettings != nil {
			networks := make([]string, 0, len(inspect.NetworkSettings.Networks))
			for _, endpoint := range inspect.NetworkSettings.Networks {
				if endpoint == nil {
					continue
				}
				if ip := strings.TrimSpace(endpoint.IPAddress); ip != "" {
					networks = append(networks, ip)
				}
				if ip := strings.TrimSpace(endpoint.GlobalIPv6Address); ip != "" {
					networks = append(networks, ip)
				}
			}
			return normalizeContainerIPList(networks)
		}
	}
	if strings.TrimSpace(runtimeHost) == "" {
		return nil
	}
	raw, err := inspectPodman(&dto.InspectReq{ID: containerID, Type: "container", RuntimeHost: runtimeHost})
	if err != nil {
		return nil
	}
	return extractContainerIPsFromInspectJSON(raw)
}
func extractContainerIPsFromInspectJSON(raw string) []string {
	type endpointView struct {
		IPAddress         string `json:"IPAddress"`
		GlobalIPv6Address string `json:"GlobalIPv6Address"`
	}
	type networkSettingsView struct {
		Networks map[string]endpointView `json:"Networks"`
	}
	type inspectView struct {
		NetworkSettings networkSettingsView `json:"NetworkSettings"`
	}
	parse := func(items []inspectView) []string {
		if len(items) == 0 {
			return nil
		}
		networks := make([]string, 0)
		for _, item := range items {
			for _, endpoint := range item.NetworkSettings.Networks {
				if ip := strings.TrimSpace(endpoint.IPAddress); ip != "" {
					networks = append(networks, ip)
				}
				if ip := strings.TrimSpace(endpoint.GlobalIPv6Address); ip != "" {
					networks = append(networks, ip)
				}
			}
		}
		return normalizeContainerIPList(networks)
	}
	var list []inspectView
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return parse(list)
	}
	var single inspectView
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return parse([]inspectView{single})
	}
	return nil
}
func inspectContainerRunUserForList(ctx context.Context, dockerClient *client.Client, containerID string, runtimeHost string) string {
	if dockerClient != nil {
		inspect, err := dockerClient.ContainerInspect(ctx, containerID)
		if err == nil && inspect.Config != nil {
			return strings.TrimSpace(inspect.Config.User)
		}
	}
	if strings.TrimSpace(runtimeHost) == "" {
		return ""
	}
	raw, err := inspectPodman(&dto.InspectReq{ID: containerID, Type: "container", RuntimeHost: runtimeHost})
	if err != nil {
		return ""
	}
	return extractContainerUserFromInspectJSON(raw)
}
func extractContainerUserFromInspectJSON(raw string) string {
	type configView struct {
		User string `json:"User"`
	}
	type inspectView struct {
		Config configView `json:"Config"`
	}
	parse := func(items []inspectView) string {
		for _, item := range items {
			if v := strings.TrimSpace(item.Config.User); v != "" {
				return v
			}
		}
		return ""
	}
	var list []inspectView
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return parse(list)
	}
	var single inspectView
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return parse([]inspectView{single})
	}
	return ""
}
func normalizeContainerLabelFilter(filter string, isPodman bool) string {
	f := strings.TrimSpace(filter)
	if !isPodman || f == "" {
		return f
	}
	f = strings.ReplaceAll(f, composeProjectLabel, podmanComposeProjectLabel)
	f = strings.ReplaceAll(f, composeConfigLabel, podmanComposeConfigLabel)
	f = strings.ReplaceAll(f, composeWorkdirLabel, podmanComposeWorkdirLabel)
	return f
}
func containerPrimaryName(item types.Container) string {
	if len(item.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(item.Names[0], "/")
}
func toStringSlice(v interface{}) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		var out []string
		for _, it := range x {
			s := strings.TrimSpace(fmt.Sprint(it))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}
func toBool(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		b, _ := strconv.ParseBool(strings.TrimSpace(x))
		return b
	default:
		return false
	}
}
func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	default:
		return 0
	}
}
