package docker

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func ListContainersMergedWithSource(ctx context.Context, options container.ListOptions) ([]types.Container, map[string]string, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	resolved := ResolveRuntime(baseCtx)
	pinnedHost := RuntimeHostPinned()

	seen := make(map[string]struct{})
	source := make(map[string]string)
	var merged []types.Container
	addContainer := func(it types.Container, from string) {
		if strings.TrimSpace(it.ID) == "" {
			return
		}
		if _, ok := seen[it.ID]; ok {
			return
		}
		seen[it.ID] = struct{}{}
		source[it.ID] = from
		merged = append(merged, it)
	}

	var lastErr error
	for _, host := range runtimeViewAPIHosts(resolved) {
		items, err := listContainersFromHost(baseCtx, host, options)
		if err != nil {
			lastErr = err
			continue
		}
		for _, it := range items {
			addContainer(it, host)
		}
	}

	if resolved.Kind == RuntimePodman && !pinnedHost {
		if rootless, err := PodmanListContainers(baseCtx, options.All); err == nil {
			for _, it := range rootless {
				addContainer(podmanCLIContainerToDockerTypes(it), "podman-cli")
			}
		} else {
			lastErr = err
		}
	}

	if len(merged) == 0 && lastErr != nil {
		return nil, nil, lastErr
	}
	return merged, source, nil
}

func ListContainersMerged(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	merged, _, err := ListContainersMergedWithSource(ctx, options)
	return merged, err
}

func ListImagesMergedWithSource(ctx context.Context) ([]image.Summary, map[string]string, []types.Container, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	resolved := ResolveRuntime(baseCtx)
	pinnedHost := RuntimeHostPinned()

	imgSeen := make(map[string]struct{})
	conSeen := make(map[string]struct{})
	imgSource := make(map[string]string)
	var images []image.Summary
	var containers []types.Container
	addImage := func(it image.Summary, from string) {
		if strings.TrimSpace(it.ID) == "" {
			return
		}
		if _, ok := imgSeen[it.ID]; ok {
			return
		}
		imgSeen[it.ID] = struct{}{}
		imgSource[it.ID] = from
		images = append(images, it)
	}
	addContainer := func(it types.Container) {
		if strings.TrimSpace(it.ID) == "" {
			return
		}
		if _, ok := conSeen[it.ID]; ok {
			return
		}
		conSeen[it.ID] = struct{}{}
		containers = append(containers, it)
	}

	var lastErr error
	if resolved.Kind == RuntimePodman && !pinnedHost {
		if rootlessImages, err := PodmanListImages(baseCtx); err == nil {
			for _, it := range rootlessImages {
				addImage(podmanCLIImageToDockerSummary(it), "podman-cli")
			}
		} else {
			lastErr = err
		}
		if rootlessContainers, err := PodmanListContainers(baseCtx, true); err == nil {
			for _, it := range rootlessContainers {
				addContainer(podmanCLIContainerToDockerTypes(it))
			}
		} else if lastErr == nil {
			lastErr = err
		}
	}

	for _, host := range runtimeViewAPIHosts(resolved) {
		imgs, cons, err := listImagesAndContainersFromHost(baseCtx, host)
		if err != nil {
			lastErr = err
			continue
		}
		for _, it := range imgs {
			addImage(it, host)
		}
		for _, it := range cons {
			addContainer(it)
		}
	}

	if len(images) == 0 && lastErr != nil {
		return nil, nil, nil, lastErr
	}
	return images, imgSource, containers, nil
}

func ListImagesMerged(ctx context.Context) ([]image.Summary, []types.Container, error) {
	images, _, containers, err := ListImagesMergedWithSource(ctx)
	return images, containers, err
}

func runtimeViewAPIHosts(resolved ResolvedRuntime) []string {
	seen := make(map[string]struct{})
	var hosts []string
	add := func(host string) {
		host = strings.TrimSpace(host)
		if host == "" || strings.HasPrefix(host, "podman://") {
			return
		}
		if _, ok := seen[host]; ok {
			return
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}

	add(resolved.Host)
	if resolved.Kind == RuntimePodman && runtime.GOOS == "linux" && !RuntimeHostPinned() {
		for _, host := range PodmanLinuxCandidateHosts() {
			add(host)
		}
	}
	return hosts
}

func listContainersFromHost(ctx context.Context, host string, options container.ListOptions) ([]types.Container, error) {
	cli, err := runtimeViewClientForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	return cli.ContainerList(ctx, options)
}

func listImagesAndContainersFromHost(ctx context.Context, host string) ([]image.Summary, []types.Container, error) {
	cli, err := runtimeViewClientForHost(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	defer cli.Close()

	imgs, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	cons, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, nil, err
	}
	return imgs, cons, nil
}

func runtimeViewClientForHost(ctx context.Context, host string) (*client.Client, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
	defer cancel()
	if err := PingHost(pingCtx, host); err != nil {
		return nil, err
	}
	return client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
}

func podmanCLIContainerToDockerTypes(it PodmanContainer) types.Container {
	name := strings.TrimSpace(it.Name)
	names := []string{}
	if name != "" {
		names = []string{"/" + strings.TrimPrefix(name, "/")}
	}
	return types.Container{
		ID:      it.ID,
		Names:   names,
		Image:   it.Image,
		ImageID: it.ImageID,
		Created: it.Created.Unix(),
		State:   strings.ToLower(strings.TrimSpace(it.State)),
		Status:  strings.TrimSpace(it.Status),
		Ports:   parsePodmanPorts(it.Ports),
		Labels:  it.Labels,
	}
}

func podmanCLIImageToDockerSummary(it PodmanImage) image.Summary {
	var sizeBytes int64
	if s := strings.TrimSpace(it.Size); s != "" {
		if n, ok := ParsePodmanImageSizeBytes(s); ok {
			sizeBytes = n
		}
	}
	return image.Summary{
		ID:       it.ID,
		RepoTags: it.Tags,
		Created:  it.Created.Unix(),
		Size:     sizeBytes,
	}
}

func parsePodmanPorts(items []string) []types.Port {
	var res []types.Port
	for _, raw := range items {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}

		left, right, ok := strings.Cut(s, "->")
		if !ok {
			privatePort, proto := parsePortProto(s)
			if privatePort == 0 {
				continue
			}
			res = append(res, types.Port{
				PrivatePort: privatePort,
				Type:        proto,
			})
			continue
		}

		privatePort, proto := parsePortProto(right)
		if privatePort == 0 {
			continue
		}

		hostIP := ""
		publicPort := uint16(0)
		l := strings.TrimSpace(left)
		if idx := strings.LastIndex(l, ":"); idx >= 0 {
			hostIP = strings.TrimSpace(l[:idx])
			publicPort = parseUint16(strings.TrimSpace(l[idx+1:]))
		} else {
			publicPort = parseUint16(l)
		}

		res = append(res, types.Port{
			IP:          hostIP,
			PublicPort:  publicPort,
			PrivatePort: privatePort,
			Type:        proto,
		})
	}
	return res
}

func parsePortProto(s string) (uint16, string) {
	val := strings.TrimSpace(s)
	if val == "" {
		return 0, ""
	}
	portPart, proto, ok := strings.Cut(val, "/")
	if !ok {
		portPart = val
		proto = ""
	}
	p := parseUint16(strings.TrimSpace(portPart))
	return p, strings.TrimSpace(proto)
}

func parseUint16(s string) uint16 {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 16)
	if err != nil {
		return 0
	}
	return uint16(n)
}
