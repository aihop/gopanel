package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func (u *ContainerService) PageNetwork(req *dto.SearchWithPage) (int64, interface{}, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.pageNetworkPodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return 0, nil, err
	}
	defer client.Close()
	list, err := client.NetworkList(context.TODO(), network.ListOptions{})
	if err != nil {
		return 0, nil, err
	}
	if len(req.Info) != 0 {
		length, count := len(list), 0
		for count < length {
			if !strings.Contains(list[count].Name, req.Info) {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	var (
		data    []dto.Network
		records []network.Inspect
	)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Created.Before(list[j].Created)
	})
	total, start, end := len(list), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		records = make([]network.Inspect, 0)
	} else {
		if end >= total {
			end = total
		}
		records = list[start:end]
	}

	for _, item := range records {
		tag := make([]string, 0)
		for key, val := range item.Labels {
			tag = append(tag, fmt.Sprintf("%s=%s", key, val))
		}
		var ipam network.IPAMConfig
		if len(item.IPAM.Config) > 0 {
			ipam = item.IPAM.Config[0]
		}
		data = append(data, dto.Network{
			ID:         item.ID,
			CreatedAt:  item.Created,
			Name:       item.Name,
			Driver:     item.Driver,
			IPAMDriver: item.IPAM.Driver,
			Subnet:     ipam.Subnet,
			Gateway:    ipam.Gateway,
			Attachable: item.Attachable,
			Labels:     tag,
		})
	}

	return int64(total), data, nil
}

func (u *ContainerService) ListNetwork() ([]dto.Options, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.listNetworkPodman()
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	list, err := client.NetworkList(context.TODO(), network.ListOptions{})
	if err != nil {
		return nil, err
	}
	var datas []dto.Options
	for _, item := range list {
		datas = append(datas, dto.Options{Option: item.Name})
	}
	sort.Slice(datas, func(i, j int) bool {
		return datas[i].Option < datas[j].Option
	})
	return datas, nil
}

func (u *ContainerService) DeleteNetwork(req *dto.BatchDelete) error {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.deleteNetworkPodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	for _, id := range req.Names {
		if err := client.NetworkRemove(context.TODO(), id); err != nil {
			if strings.Contains(err.Error(), "has active endpoints") {
				return buserr.WithDetail(constant.ErrInUsed, id, nil)
			}
			return err
		}
	}
	return nil
}
func (u *ContainerService) CreateNetwork(req *dto.NetworkCreate) error {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.createNetworkPodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	var (
		ipams    []network.IPAMConfig
		enableV6 bool
	)
	if req.Ipv4 {
		var itemIpam network.IPAMConfig
		if len(req.AuxAddress) != 0 {
			itemIpam.AuxAddress = make(map[string]string)
		}
		if len(req.Subnet) != 0 {
			itemIpam.Subnet = req.Subnet
		}
		if len(req.Gateway) != 0 {
			itemIpam.Gateway = req.Gateway
		}
		if len(req.IPRange) != 0 {
			itemIpam.IPRange = req.IPRange
		}
		for _, addr := range req.AuxAddress {
			itemIpam.AuxAddress[addr.Key] = addr.Value
		}
		ipams = append(ipams, itemIpam)
	}
	if req.Ipv6 {
		enableV6 = true
		var itemIpam network.IPAMConfig
		if len(req.AuxAddress) != 0 {
			itemIpam.AuxAddress = make(map[string]string)
		}
		if len(req.SubnetV6) != 0 {
			itemIpam.Subnet = req.SubnetV6
		}
		if len(req.GatewayV6) != 0 {
			itemIpam.Gateway = req.GatewayV6
		}
		if len(req.IPRangeV6) != 0 {
			itemIpam.IPRange = req.IPRangeV6
		}
		for _, addr := range req.AuxAddressV6 {
			itemIpam.AuxAddress[addr.Key] = addr.Value
		}
		ipams = append(ipams, itemIpam)
	}

	options := network.CreateOptions{
		EnableIPv6: &enableV6,
		Driver:     req.Driver,
		Options:    stringsToMap(req.Options),
		Labels:     stringsToMap(req.Labels),
	}
	if len(ipams) != 0 {
		options.IPAM = &network.IPAM{Config: ipams}
	}
	if _, err := client.NetworkCreate(context.TODO(), req.Name, options); err != nil {
		return err
	}
	return nil
}

func (u *ContainerService) pageNetworkPodman(req *dto.SearchWithPage) (int64, interface{}, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return 0, nil, err
	}
	cmdExec := exec.Command("podman", "network", "ls", "--format", "json")
	out, err := cmdExec.CombinedOutput()
	if err != nil {
		return 0, nil, errors.New(strings.TrimSpace(string(out)))
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return 0, nil, err
	}
	var list []dto.Network
	for _, item := range raw {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" || name == "<nil>" {
			name = strings.TrimSpace(fmt.Sprint(item["Name"]))
		}
		if req.Info != "" && !strings.Contains(name, req.Info) {
			continue
		}
		n := dto.Network{
			ID:     strings.TrimSpace(fmt.Sprint(item["id"])),
			Name:   name,
			Driver: strings.TrimSpace(fmt.Sprint(item["driver"])),
		}
		if n.ID == "" || n.ID == "<nil>" {
			n.ID = strings.TrimSpace(fmt.Sprint(item["ID"]))
		}
		if n.Driver == "" || n.Driver == "<nil>" {
			n.Driver = strings.TrimSpace(fmt.Sprint(item["Driver"]))
		}
		if labels, ok := item["labels"].(map[string]interface{}); ok {
			for k, v := range labels {
				n.Labels = append(n.Labels, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if labels, ok := item["Labels"].(map[string]interface{}); ok && len(n.Labels) == 0 {
			for k, v := range labels {
				n.Labels = append(n.Labels, fmt.Sprintf("%s=%v", k, v))
			}
		}

		if ipamOpts, ok := item["ipam_options"].(map[string]interface{}); ok {
			n.IPAMDriver = strings.TrimSpace(fmt.Sprint(ipamOpts["driver"]))
		}
		if subnets, ok := item["subnets"].([]interface{}); ok && len(subnets) > 0 {
			if s0, ok := subnets[0].(map[string]interface{}); ok {
				n.Subnet = strings.TrimSpace(fmt.Sprint(s0["subnet"]))
				n.Gateway = strings.TrimSpace(fmt.Sprint(s0["gateway"]))
			}
		}
		if created := strings.TrimSpace(fmt.Sprint(item["created"])); created != "" && created != "<nil>" {
			if t, e := time.Parse(time.RFC3339Nano, created); e == nil {
				n.CreatedAt = t
			}
		}
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	total := len(list)
	start, end := (req.Page-1)*req.Limit, req.Page*req.Limit
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	if start > total {
		return int64(total), []dto.Network{}, nil
	}
	return int64(total), list[start:end], nil
}

func (u *ContainerService) listNetworkPodman() ([]dto.Options, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	cmdExec := exec.Command("podman", "network", "ls", "--format", "json")
	out, err := cmdExec.CombinedOutput()
	if err != nil {
		return nil, errors.New(strings.TrimSpace(string(out)))
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	var datas []dto.Options
	for _, item := range raw {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" || name == "<nil>" {
			name = strings.TrimSpace(fmt.Sprint(item["Name"]))
		}
		if name == "" || name == "<nil>" {
			continue
		}
		datas = append(datas, dto.Options{Option: name})
	}
	sort.Slice(datas, func(i, j int) bool { return datas[i].Option < datas[j].Option })
	return datas, nil
}

func (u *ContainerService) deleteNetworkPodman(req *dto.BatchDelete) error {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return err
	}
	for _, id := range req.Names {
		c := exec.Command("podman", "network", "rm", id)
		out, err := c.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if strings.Contains(strings.ToLower(msg), "in use") || strings.Contains(strings.ToLower(msg), "active endpoints") {
				return buserr.WithDetail(constant.ErrInUsed, id, nil)
			}
			if msg == "" {
				return err
			}
			return errors.New(msg)
		}
	}
	return nil
}

func (u *ContainerService) createNetworkPodman(req *dto.NetworkCreate) error {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return err
	}
	args := []string{"network", "create", "--driver", req.Driver}
	if req.Ipv6 {
		args = append(args, "--ipv6")
	}
	if req.Subnet != "" {
		args = append(args, "--subnet", req.Subnet)
	}
	if req.Gateway != "" {
		args = append(args, "--gateway", req.Gateway)
	}
	if req.IPRange != "" {
		args = append(args, "--ip-range", req.IPRange)
	}
	if req.SubnetV6 != "" {
		args = append(args, "--subnet", req.SubnetV6)
	}
	if req.GatewayV6 != "" {
		args = append(args, "--gateway", req.GatewayV6)
	}
	if req.IPRangeV6 != "" {
		args = append(args, "--ip-range", req.IPRangeV6)
	}
	for k, v := range stringsToMap(req.Options) {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range stringsToMap(req.Labels) {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, req.Name)
	c := exec.Command("podman", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return errors.New(msg)
	}
	return nil
}

func isStaticIPSubnetError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "requested static ip") && strings.Contains(msg, "not in any subnet")
}

func cloneNetworkingConfigWithoutStaticIP(src *network.NetworkingConfig) *network.NetworkingConfig {
	if src == nil {
		return nil
	}
	dst := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings, len(src.EndpointsConfig)),
	}
	for name, endpoint := range src.EndpointsConfig {
		if endpoint == nil {
			dst.EndpointsConfig[name] = nil
			continue
		}
		cloned := *endpoint
		if endpoint.IPAMConfig != nil {
			ipam := *endpoint.IPAMConfig
			ipam.IPv4Address = ""
			ipam.IPv6Address = ""
			cloned.IPAMConfig = &ipam
		}
		dst.EndpointsConfig[name] = &cloned
	}
	return dst
}

func networkConfigStaticIPChanged(current *network.NetworkingConfig, original *network.NetworkingConfig) bool {
	if current == nil || original == nil {
		return false
	}
	for name, endpoint := range original.EndpointsConfig {
		if endpoint == nil || endpoint.IPAMConfig == nil {
			continue
		}
		oldIPv4 := strings.TrimSpace(endpoint.IPAMConfig.IPv4Address)
		oldIPv6 := strings.TrimSpace(endpoint.IPAMConfig.IPv6Address)
		newEndpoint := current.EndpointsConfig[name]
		if newEndpoint == nil || newEndpoint.IPAMConfig == nil {
			if oldIPv4 != "" || oldIPv6 != "" {
				return true
			}
			continue
		}
		if oldIPv4 != strings.TrimSpace(newEndpoint.IPAMConfig.IPv4Address) || oldIPv6 != strings.TrimSpace(newEndpoint.IPAMConfig.IPv6Address) {
			return true
		}
	}
	return false
}

func sanitizeStaticIPForContainerNetwork(ctx context.Context, client *client.Client, req *dto.ContainerOperate) {
	if req == nil || strings.TrimSpace(req.Network) == "" {
		return
	}

	if ip := strings.TrimSpace(req.Ipv4); ip != "" {
		ok, subnet, err := ipv4BelongsToDockerNetwork(ctx, client, req.Network, ip)
		if err != nil {
			global.LOG.Warnf("inspect network %s for ipv4 validation failed: %v", req.Network, err)
		} else if !ok {
			global.LOG.Warnf("static ipv4 %s does not belong to network %s subnet %s, fallback to dynamic ip", ip, req.Network, subnet)
			req.Ipv4 = ""
		}
	}

	if ip := strings.TrimSpace(req.Ipv6); ip != "" {
		ok, subnet, err := ipv6BelongsToDockerNetwork(ctx, client, req.Network, ip)
		if err != nil {
			global.LOG.Warnf("inspect network %s for ipv6 validation failed: %v", req.Network, err)
		} else if !ok {
			global.LOG.Warnf("static ipv6 %s does not belong to network %s subnet %s, fallback to dynamic ip", ip, req.Network, subnet)
			req.Ipv6 = ""
		}
	}
}

func ipv4BelongsToDockerNetwork(ctx context.Context, client *client.Client, networkName, ip string) (bool, string, error) {
	networkInfo, err := client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return false, "", err
	}
	return ipBelongsToDockerNetwork(networkInfo, ip, false)
}

func ipv6BelongsToDockerNetwork(ctx context.Context, client *client.Client, networkName, ip string) (bool, string, error) {
	networkInfo, err := client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return false, "", err
	}
	return ipBelongsToDockerNetwork(networkInfo, ip, true)
}

func ipBelongsToDockerNetwork(networkInfo network.Inspect, ip string, wantIPv6 bool) (bool, string, error) {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false, "", fmt.Errorf("invalid ip: %s", ip)
	}

	var matchedSubnets []string
	for _, cfg := range networkInfo.IPAM.Config {
		subnet := strings.TrimSpace(cfg.Subnet)
		if subnet == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(subnet)
		if err != nil {
			continue
		}
		isIPv6Subnet := ipNet.IP.To4() == nil
		if isIPv6Subnet != wantIPv6 {
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true, subnet, nil
		}
		matchedSubnets = append(matchedSubnets, subnet)
	}

	if len(matchedSubnets) > 0 {
		return false, strings.Join(matchedSubnets, ","), nil
	}
	return true, "", nil
}

func retryCreateWithoutStaticIPOnSubnetError(
	ctx context.Context,
	client *client.Client,
	containerName string,
	config *container.Config,
	hostConf *container.HostConfig,
	networkConf *network.NetworkingConfig,
	createErr error,
) (container.CreateResponse, error, bool) {
	if !isStaticIPSubnetError(createErr) || networkConf == nil || len(networkConf.EndpointsConfig) == 0 {
		return container.CreateResponse{}, createErr, false
	}

	clonedNetworkConf := cloneNetworkingConfigWithoutStaticIP(networkConf)
	if !networkConfigStaticIPChanged(clonedNetworkConf, networkConf) {
		return container.CreateResponse{}, createErr, false
	}

	global.LOG.Warnf("create container %s failed with static ip subnet mismatch, retry with dynamic ip: %v", containerName, createErr)
	con, retryErr := client.ContainerCreate(ctx, config, hostConf, clonedNetworkConf, &v1.Platform{}, containerName)
	if retryErr != nil {
		return container.CreateResponse{}, retryErr, true
	}
	return con, nil, true
}

func retryStartWithoutStaticIPOnSubnetError(
	ctx context.Context,
	client *client.Client,
	containerName string,
	containerID string,
	config *container.Config,
	hostConf *container.HostConfig,
	networkConf *network.NetworkingConfig,
	startErr error,
) (error, bool) {
	if !isStaticIPSubnetError(startErr) || networkConf == nil || len(networkConf.EndpointsConfig) == 0 {
		return startErr, false
	}

	clonedNetworkConf := cloneNetworkingConfigWithoutStaticIP(networkConf)
	if !networkConfigStaticIPChanged(clonedNetworkConf, networkConf) {
		return startErr, false
	}

	_ = client.ContainerRemove(ctx, containerID, container.RemoveOptions{RemoveVolumes: true, Force: true})
	global.LOG.Warnf("start container %s failed with static ip subnet mismatch, retry with dynamic ip: %v", containerName, startErr)
	con, err := client.ContainerCreate(ctx, config, hostConf, clonedNetworkConf, &v1.Platform{}, containerName)
	if err != nil {
		return fmt.Errorf("retry container create without static ip failed, err: %v", err), true
	}
	if err := client.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		_ = client.ContainerRemove(ctx, con.ID, container.RemoveOptions{RemoveVolumes: true, Force: true})
		return fmt.Errorf("retry container start without static ip failed, err: %v", err), true
	}
	return nil, true
}
