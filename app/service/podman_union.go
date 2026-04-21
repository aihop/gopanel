package service

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

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
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	seen := make(map[string]struct{})
	var merged []types.Container

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
			merged = append(merged, it)
		}
	}
	if len(merged) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return merged, nil
}

func listImagesMergedByHost(ctx context.Context) ([]image.Summary, []types.Container, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	imgSeen := make(map[string]struct{})
	conSeen := make(map[string]struct{})
	var images []image.Summary
	var containers []types.Container

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
		return nil, nil, lastErr
	}
	return images, containers, nil
}

