package service

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	dockutil "github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func podmanLinuxCandidateHosts() []string {
	uid := os.Getuid()
	return []string{
		"unix:///run/user/" + strconv.Itoa(uid) + "/podman/podman.sock",
		"unix:///run/podman/podman.sock",
	}
}

func listContainersMergedByHost(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	merged, _, err := listContainersMergedByHostWithSource(ctx, options)
	return merged, err
}

func listContainersMergedByHostWithSource(ctx context.Context, options container.ListOptions) ([]types.Container, map[string]string, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	seen := make(map[string]struct{})
	source := make(map[string]string)
	var merged []types.Container

	if rootless, err := dockutil.PodmanListContainers(baseCtx, options.All); err == nil {
		for _, it := range rootless {
			if strings.TrimSpace(it.ID) == "" {
				continue
			}
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
			source[it.ID] = "podman-cli"
			merged = append(merged, podmanCLIContainerToDockerTypes(it))
		}
	}

	var lastErr error
	for _, host := range podmanLinuxCandidateHosts() {
		if strings.HasPrefix(host, "unix://") {
			if _, err := os.Stat(strings.TrimPrefix(host, "unix://")); err != nil {
				lastErr = err
				continue
			}
		}
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
		if err != nil {
			lastErr = err
			continue
		}
		pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
		_, pingErr := cli.Ping(pingCtx)
		cancel()
		if pingErr != nil {
			lastErr = pingErr
			_ = cli.Close()
			continue
		}
		items, err := cli.ContainerList(baseCtx, options)
		_ = cli.Close()
		if err != nil {
			lastErr = err
			continue
		}
		for _, it := range items {
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
			source[it.ID] = host
			merged = append(merged, it)
		}
	}
	if len(merged) == 0 && lastErr != nil {
		return nil, nil, lastErr
	}
	return merged, source, nil
}

func listImagesMergedByHost(ctx context.Context) ([]image.Summary, []types.Container, error) {
	images, _, containers, err := listImagesMergedByHostWithSource(ctx)
	return images, containers, err
}

func listImagesMergedByHostWithSource(ctx context.Context) ([]image.Summary, map[string]string, []types.Container, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	imgSeen := make(map[string]struct{})
	conSeen := make(map[string]struct{})
	imgSource := make(map[string]string)
	var images []image.Summary
	var containers []types.Container

	if rootlessImages, err := dockutil.PodmanListImages(baseCtx); err == nil {
		for _, it := range rootlessImages {
			if strings.TrimSpace(it.ID) == "" {
				continue
			}
			if _, ok := imgSeen[it.ID]; ok {
				continue
			}
			imgSeen[it.ID] = struct{}{}
			imgSource[it.ID] = "podman-cli"
			images = append(images, podmanCLIImageToDockerSummary(it))
		}
	}
	if rootlessContainers, err := dockutil.PodmanListContainers(baseCtx, true); err == nil {
		for _, it := range rootlessContainers {
			if strings.TrimSpace(it.ID) == "" {
				continue
			}
			if _, ok := conSeen[it.ID]; ok {
				continue
			}
			conSeen[it.ID] = struct{}{}
			containers = append(containers, podmanCLIContainerToDockerTypes(it))
		}
	}

	var lastErr error
	for _, host := range podmanLinuxCandidateHosts() {
		if strings.HasPrefix(host, "unix://") {
			if _, err := os.Stat(strings.TrimPrefix(host, "unix://")); err != nil {
				lastErr = err
				continue
			}
		}
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
		if err != nil {
			lastErr = err
			continue
		}
		pingCtx, cancel := context.WithTimeout(baseCtx, 800*time.Millisecond)
		_, pingErr := cli.Ping(pingCtx)
		cancel()
		if pingErr != nil {
			lastErr = pingErr
			_ = cli.Close()
			continue
		}

		imgs, err := cli.ImageList(baseCtx, image.ListOptions{})
		if err != nil {
			lastErr = err
			_ = cli.Close()
			continue
		}
		cons, _ := cli.ContainerList(baseCtx, container.ListOptions{All: true})
		_ = cli.Close()

		for _, it := range imgs {
			if _, ok := imgSeen[it.ID]; ok {
				continue
			}
			imgSeen[it.ID] = struct{}{}
			imgSource[it.ID] = host
			images = append(images, it)
		}
		for _, it := range cons {
			if _, ok := conSeen[it.ID]; ok {
				continue
			}
			conSeen[it.ID] = struct{}{}
			containers = append(containers, it)
		}
	}
	if len(images) == 0 && lastErr != nil {
		return nil, nil, nil, lastErr
	}
	return images, imgSource, containers, nil
}

func podmanCLIContainerToDockerTypes(it dockutil.PodmanContainer) types.Container {
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

func podmanCLIImageToDockerSummary(it dockutil.PodmanImage) image.Summary {
	var sizeBytes int64
	if s := strings.TrimSpace(it.Size); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
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
