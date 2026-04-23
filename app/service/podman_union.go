package service

import (
	"context"

	dockutil "github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

func podmanLinuxCandidateHosts() []string {
	return dockutil.PodmanLinuxCandidateHosts()
}

func listContainersMergedByHost(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return dockutil.ListContainersMerged(ctx, options)
}

func listContainersMergedByHostWithSource(ctx context.Context, options container.ListOptions) ([]types.Container, map[string]string, error) {
	return dockutil.ListContainersMergedWithSource(ctx, options)
}

func listImagesMergedByHost(ctx context.Context) ([]image.Summary, []types.Container, error) {
	return dockutil.ListImagesMerged(ctx)
}

func listImagesMergedByHostWithSource(ctx context.Context) ([]image.Summary, map[string]string, []types.Container, error) {
	return dockutil.ListImagesMergedWithSource(ctx)
}
