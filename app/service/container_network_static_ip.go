package service

import (
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/global"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"net"
	"strings"
)

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
	dst := &network.NetworkingConfig{EndpointsConfig: make(map[string]*network.EndpointSettings, len(src.EndpointsConfig))}
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
func retryCreateWithoutStaticIPOnSubnetError(ctx context.Context, client *client.Client, containerName string, config *container.Config, hostConf *container.HostConfig, networkConf *network.NetworkingConfig, createErr error) (container.CreateResponse, error, bool) {
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
func retryStartWithoutStaticIPOnSubnetError(ctx context.Context, client *client.Client, containerName string, containerID string, config *container.Config, hostConf *container.HostConfig, networkConf *network.NetworkingConfig, startErr error) (error, bool) {
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
