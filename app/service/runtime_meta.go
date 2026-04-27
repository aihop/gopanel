package service

import (
	"context"
	"strings"

	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
)

type ContainerRuntimeMeta struct {
	RuntimeHost string
	RuntimeKind string
	RuntimeMode string
	RunUser     string
}

type containerRuntimeLookup struct {
	defaultMeta ContainerRuntimeMeta
	dockerClient *dockerclient.Client
	containersByID map[string]types.Container
	containersByName map[string]types.Container
	sourceByID map[string]string
}

func FillAppInstallRuntimeMeta(ctx context.Context, installs []model.AppInstall) {
	if len(installs) == 0 {
		return
	}
	lookup, err := newContainerRuntimeLookup(ctx)
	if err != nil {
		return
	}
	defer lookup.Close()
	for i := range installs {
		meta := lookup.metaForContainerNames(installs[i].ContainerName)
		installs[i].RuntimeHost = meta.RuntimeHost
		installs[i].RuntimeKind = meta.RuntimeKind
		installs[i].RuntimeMode = meta.RuntimeMode
		installs[i].RunUser = meta.RunUser
	}
}

func FillPipelineRuntimeMeta(ctx context.Context, pipelines []model.Pipeline) {
	if len(pipelines) == 0 {
		return
	}
	lookup, err := newContainerRuntimeLookup(ctx)
	if err != nil {
		return
	}
	defer lookup.Close()
	recordRepo := repo.NewPipelineRecord(global.DB)
	for i := range pipelines {
		containerID, _ := recordRepo.LatestRunnerContainerID(pipelines[i].ID)
		meta := lookup.metaForContainerID(containerID)
		pipelines[i].RuntimeHost = meta.RuntimeHost
		pipelines[i].RuntimeKind = meta.RuntimeKind
		pipelines[i].RuntimeMode = meta.RuntimeMode
		pipelines[i].RunUser = meta.RunUser
	}
}

func FillPipelineRecordRuntimeMeta(ctx context.Context, records []model.PipelineRecord) {
	if len(records) == 0 {
		return
	}
	lookup, err := newContainerRuntimeLookup(ctx)
	if err != nil {
		return
	}
	defer lookup.Close()
	for i := range records {
		meta := lookup.metaForContainerID(records[i].RunnerContainerID)
		records[i].RuntimeHost = meta.RuntimeHost
		records[i].RuntimeKind = meta.RuntimeKind
		records[i].RuntimeMode = meta.RuntimeMode
		records[i].RunUser = meta.RunUser
	}
}

func FillWebsiteRuntimeMeta(ctx context.Context, websites []*response.WebsiteRes) {
	if len(websites) == 0 {
		return
	}
	lookup, err := newContainerRuntimeLookup(ctx)
	if err != nil {
		return
	}
	defer lookup.Close()

	recordRepo := repo.NewPipelineRecord(global.DB)
	for i := range websites {
		if websites[i] == nil {
			continue
		}
		meta := lookup.defaultMeta
		switch {
		case websites[i].AppInstallID > 0:
			appInstall, err := appInstallRepo.GetFirst(commonRepo.WithByID(websites[i].AppInstallID))
			if err == nil {
				meta = lookup.metaForContainerNames(appInstall.ContainerName)
			}
		case websites[i].PipelineID > 0:
			containerID, _ := recordRepo.LatestRunnerContainerID(websites[i].PipelineID)
			meta = lookup.metaForContainerID(containerID)
		}
		websites[i].RuntimeHost = meta.RuntimeHost
		websites[i].RuntimeKind = meta.RuntimeKind
		websites[i].RuntimeMode = meta.RuntimeMode
		websites[i].RunUser = meta.RunUser
	}
}

func newContainerRuntimeLookup(ctx context.Context) (*containerRuntimeLookup, error) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	resolved := docker.ResolveRuntime(baseCtx)
	lookup := &containerRuntimeLookup{
		defaultMeta: ContainerRuntimeMeta{
			RuntimeHost: strings.TrimSpace(resolved.Host),
			RuntimeKind: string(resolved.Kind),
		},
		containersByID: make(map[string]types.Container),
		containersByName: make(map[string]types.Container),
		sourceByID: make(map[string]string),
	}
	lookup.defaultMeta.RuntimeMode = inferContainerRuntimeMode(lookup.defaultMeta.RuntimeKind, lookup.defaultMeta.RuntimeHost)

	if resolved.Kind == docker.RuntimePodman {
		items, sourceByID, err := docker.ListContainersMergedWithSource(baseCtx, container.ListOptions{All: true})
		if err != nil {
			return nil, err
		}
		lookup.indexContainers(items, sourceByID)
		return lookup, nil
	}

	cli, err := docker.NewRuntimeAPIClient()
	if err != nil {
		return nil, err
	}
	items, err := cli.ContainerList(baseCtx, container.ListOptions{All: true})
	if err != nil {
		_ = cli.Close()
		return nil, err
	}
	lookup.dockerClient = cli
	lookup.indexContainers(items, nil)
	return lookup, nil
}

func (l *containerRuntimeLookup) Close() {
	if l == nil || l.dockerClient == nil {
		return
	}
	_ = l.dockerClient.Close()
}

func (l *containerRuntimeLookup) indexContainers(items []types.Container, sourceByID map[string]string) {
	for _, item := range items {
		if id := strings.TrimSpace(item.ID); id != "" {
			l.containersByID[id] = item
			if sourceByID != nil {
				if host := strings.TrimSpace(sourceByID[id]); host != "" {
					l.sourceByID[id] = host
				}
			}
		}
		for _, rawName := range item.Names {
			name := normalizeContainerName(rawName)
			if name == "" {
				continue
			}
			l.containersByName[name] = item
		}
	}
}

func (l *containerRuntimeLookup) metaForContainerNames(rawNames string) ContainerRuntimeMeta {
	if l == nil {
		return ContainerRuntimeMeta{}
	}
	for _, name := range splitContainerNames(rawNames) {
		if item, ok := l.containersByName[name]; ok {
			return l.metaForContainerID(item.ID)
		}
	}
	return l.defaultMeta
}

func (l *containerRuntimeLookup) metaForContainerID(containerID string) ContainerRuntimeMeta {
	if l == nil {
		return ContainerRuntimeMeta{}
	}
	meta := l.defaultMeta
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return meta
	}
	item, ok := l.containersByID[containerID]
	if !ok {
		return meta
	}
	if host := strings.TrimSpace(l.sourceByID[item.ID]); host != "" {
		meta.RuntimeHost = host
		meta.RuntimeMode = inferContainerRuntimeMode(meta.RuntimeKind, host)
	}
	meta.RunUser = inspectContainerRunUserForList(context.Background(), l.dockerClient, item.ID, meta.RuntimeHost)
	return meta
}
