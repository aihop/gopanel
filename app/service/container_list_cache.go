package service

import (
	"context"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"strings"
	"sync"
	"time"
)

func (u *ContainerService) ContainerListStats() ([]dto.ContainerListStats, error) {
	if cached := getContainerListStatsCache(); len(cached) > 0 {
		return cached, nil
	}
	ctx := context.Background()
	list, source, err := getContainerListView(ctx)
	if err != nil {
		return nil, err
	}
	datas := make([]dto.ContainerListStats, len(list))
	var wg sync.WaitGroup
	wg.Add(len(list))
	isPodman := docker.IsPodmanRuntime(ctx)
	var sharedClient *client.Client
	if !isPodman {
		sharedClient, err = docker.NewDockerClient()
		if err != nil {
			return nil, err
		}
		defer sharedClient.Close()
	}
	for i := 0; i < len(list); i++ {
		go func(index int, item types.Container) {
			if isPodman {
				host := strings.TrimSpace(source[item.ID])
				if host == "" || host == "podman-cli" {
					datas[index] = loadContainerListStatPodmanCLI(item.ID)
					wg.Done()
					return
				}
				if !strings.HasPrefix(host, "unix://") {
					wg.Done()
					return
				}
				cli, cliErr := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
				if cliErr != nil {
					wg.Done()
					return
				}
				datas[index] = loadCpuAndMem(cli, item.ID)
				_ = cli.Close()
				wg.Done()
				return
			}
			datas[index] = loadCpuAndMem(sharedClient, item.ID)
			wg.Done()
		}(i, list[i])
	}
	wg.Wait()
	setContainerListStatsCache(datas)
	return datas, nil
}
func getContainerListView(ctx context.Context) ([]types.Container, map[string]string, error) {
	now := time.Now()
	containerListViewCache.mu.RLock()
	entry := containerListViewCache.entry
	refreshing := containerListViewCache.refreshing
	waitCh := containerListViewCache.waitCh
	containerListViewCache.mu.RUnlock()
	if len(entry.items) > 0 && now.Before(entry.expireAt) {
		return cloneContainerList(entry.items), cloneContainerSourceMap(entry.source), nil
	}
	if len(entry.items) > 0 {
		if !refreshing {
			go refreshContainerListView(context.Background())
		}
		return cloneContainerList(entry.items), cloneContainerSourceMap(entry.source), nil
	}
	if refreshing && waitCh != nil {
		select {
		case <-waitCh:
			containerListViewCache.mu.RLock()
			entry = containerListViewCache.entry
			containerListViewCache.mu.RUnlock()
			if len(entry.items) > 0 {
				return cloneContainerList(entry.items), cloneContainerSourceMap(entry.source), nil
			}
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
	entry, err := refreshContainerListView(ctx)
	if err != nil {
		if len(entry.items) > 0 {
			return cloneContainerList(entry.items), cloneContainerSourceMap(entry.source), nil
		}
		return nil, nil, err
	}
	return cloneContainerList(entry.items), cloneContainerSourceMap(entry.source), nil
}
func refreshContainerListView(ctx context.Context) (containerListViewCacheEntry, error) {
	containerListViewCache.mu.Lock()
	if containerListViewCache.refreshing {
		waitCh := containerListViewCache.waitCh
		version := containerListViewCache.version
		containerListViewCache.mu.Unlock()
		if waitCh != nil {
			select {
			case <-waitCh:
				containerListViewCache.mu.RLock()
				entry := containerListViewCache.entry
				ver := containerListViewCache.version
				containerListViewCache.mu.RUnlock()
				// If version changed during our wait, the cache was invalidated
				// and another refresh will or has provided fresh data
				if ver != version {
					return containerListViewCacheEntry{}, nil
				}
				return entry, nil
			case <-ctx.Done():
				return containerListViewCacheEntry{}, ctx.Err()
			}
		}
		containerListViewCache.mu.RLock()
		entry := containerListViewCache.entry
		containerListViewCache.mu.RUnlock()
		return entry, nil
	}
	version := containerListViewCache.version
	waitCh := make(chan struct{})
	containerListViewCache.refreshing = true
	containerListViewCache.waitCh = waitCh
	containerListViewCache.mu.Unlock()
	entry, err := loadContainerListView(ctx)
	containerListViewCache.mu.Lock()
	// If version changed during our work, the cache was invalidated
	// by a container operation. Our data is stale, don't cache it.
	if containerListViewCache.version != version {
		containerListViewCache.refreshing = false
		containerListViewCache.waitCh = nil
		close(waitCh)
		containerListViewCache.mu.Unlock()
		return containerListViewCacheEntry{}, nil
	}
	if err == nil {
		containerListViewCache.entry = entry
	} else {
		entry = containerListViewCache.entry
	}
	containerListViewCache.refreshing = false
	containerListViewCache.waitCh = nil
	close(waitCh)
	containerListViewCache.mu.Unlock()
	return entry, err
}
func loadContainerListView(ctx context.Context) (containerListViewCacheEntry, error) {
	var (
		items  []types.Container
		source map[string]string
		err    error
	)
	if docker.IsPodmanRuntime(ctx) {
		items, source, err = listContainersMergedByHostWithSource(ctx, container.ListOptions{All: true})
	} else {
		var cli *client.Client
		cli, err = docker.NewDockerClient()
		if err == nil {
			defer cli.Close()
			items, err = cli.ContainerList(ctx, container.ListOptions{All: true})
		}
	}
	if err != nil {
		return containerListViewCacheEntry{}, err
	}
	return containerListViewCacheEntry{expireAt: time.Now().Add(containerListViewCacheTTL), items: cloneContainerList(items), source: cloneContainerSourceMap(source)}, nil
}
func getContainerListStatsCache() []dto.ContainerListStats {
	now := time.Now()
	containerListStatsCache.mu.RLock()
	entry := containerListStatsCache.entry
	containerListStatsCache.mu.RUnlock()
	if len(entry.items) == 0 || !now.Before(entry.expireAt) {
		return nil
	}
	return append([]dto.ContainerListStats(nil), entry.items...)
}
func setContainerListStatsCache(items []dto.ContainerListStats) {
	containerListStatsCache.mu.Lock()
	containerListStatsCache.entry = containerListStatsCacheEntry{expireAt: time.Now().Add(containerListStatsCacheTTL), items: append([]dto.ContainerListStats(nil), items...)}
	containerListStatsCache.mu.Unlock()
}
func invalidateContainerListCaches() {
	containerListViewCache.mu.Lock()
	containerListViewCache.entry = containerListViewCacheEntry{}
	containerListViewCache.version++
	if containerListViewCache.waitCh != nil {
		close(containerListViewCache.waitCh)
	}
	containerListViewCache.waitCh = nil
	containerListViewCache.refreshing = false
	containerListViewCache.mu.Unlock()

	containerListStatsCache.mu.Lock()
	containerListStatsCache.entry = containerListStatsCacheEntry{}
	containerListStatsCache.mu.Unlock()
}
func cloneContainerList(items []types.Container) []types.Container {
	if len(items) == 0 {
		return nil
	}
	return append([]types.Container(nil), items...)
}
func cloneContainerSourceMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for k, v := range source {
		cloned[k] = v
	}
	return cloned
}
