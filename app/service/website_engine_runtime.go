package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
)

type engineRuntimeTemplate struct {
	Binds       []string
	RuntimeDir  string
	NetworkMode string
	ExtraHosts  []string
	Env         []string
	Source      string
	ContainerID string
}

func detectReusableRuntimeTemplate(ctx context.Context, cli *dockerclient.Client, imageName, workingDir, previousContainerID string) engineRuntimeTemplate {
	if template, ok := loadRuntimeTemplateFromContainer(ctx, cli, previousContainerID, workingDir, "current-container"); ok {
		return template
	}
	return loadRuntimeTemplateFromImageContainers(ctx, cli, imageName, workingDir, previousContainerID)
}

func loadRuntimeTemplateFromContainer(ctx context.Context, cli *dockerclient.Client, containerID, workingDir, source string) (engineRuntimeTemplate, bool) {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return engineRuntimeTemplate{}, false
	}
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return engineRuntimeTemplate{}, false
	}
	template := buildRuntimeTemplateFromInspect(inspect, workingDir)
	if len(template.Binds) == 0 {
		return engineRuntimeTemplate{}, false
	}
	template.Source = source
	template.ContainerID = inspect.ID
	return template, true
}

func loadRuntimeTemplateFromImageContainers(ctx context.Context, cli *dockerclient.Client, imageName, workingDir, excludeContainerID string) engineRuntimeTemplate {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return engineRuntimeTemplate{}
	}

	bestScore := -1
	bestTemplate := engineRuntimeTemplate{}
	excludeContainerID = strings.TrimSpace(excludeContainerID)
	for _, item := range containers {
		if excludeContainerID != "" && item.ID == excludeContainerID {
			continue
		}
		inspect, err := cli.ContainerInspect(ctx, item.ID)
		if err != nil || inspect.Config == nil {
			continue
		}
		if !sameRuntimeImageRepo(inspect.Config.Image, imageName) {
			continue
		}
		template := buildRuntimeTemplateFromInspect(inspect, workingDir)
		if len(template.Binds) == 0 {
			continue
		}
		score := len(template.Binds)
		if inspect.ContainerJSONBase != nil && inspect.ContainerJSONBase.State != nil && inspect.ContainerJSONBase.State.Running {
			score += 100
		}
		if sameRuntimeImageRef(inspect.Config.Image, imageName) {
			score += 20
		}
		if score > bestScore {
			bestScore = score
			template.Source = inspect.Name
			template.ContainerID = inspect.ID
			bestTemplate = template
		}
	}

	return bestTemplate
}

func buildRuntimeTemplateFromInspect(inspect container.InspectResponse, workingDir string) engineRuntimeTemplate {
	template := engineRuntimeTemplate{}
	if inspect.HostConfig != nil {
		template.NetworkMode = string(inspect.HostConfig.NetworkMode)
		template.ExtraHosts = append(template.ExtraHosts, inspect.HostConfig.ExtraHosts...)
	}

	if inspect.Config != nil && len(inspect.Config.Env) > 0 {
		template.Env = append(template.Env, inspect.Config.Env...)
	}

	seen := make(map[string]struct{})
	for _, mountPoint := range inspect.Mounts {
		bind, source := buildRuntimeBindFromMount(string(mountPoint.Type), mountPoint.Source, mountPoint.Name, mountPoint.Destination, mountPoint.Mode, mountPoint.RW)
		if bind == "" {
			continue
		}
		if _, ok := seen[bind]; ok {
			continue
		}
		seen[bind] = struct{}{}
		template.Binds = append(template.Binds, bind)
		if template.RuntimeDir == "" && matchRuntimeMountDestination(mountPoint.Destination, workingDir) {
			template.RuntimeDir = source
		}
	}

	return template
}

func buildRuntimeBindFromMount(mountType, source, name, destination, mode string, rw bool) (string, string) {
	source = strings.TrimSpace(source)
	name = strings.TrimSpace(name)
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "", ""
	}

	bindSource := source
	switch mountType {
	case "bind":
		if bindSource == "" {
			return "", ""
		}
	case "volume":
		if name != "" {
			bindSource = name
		}
		if bindSource == "" {
			return "", ""
		}
	default:
		return "", ""
	}

	if mode == "" {
		if rw {
			mode = "rw"
		} else {
			mode = "ro"
		}
	}
	return fmt.Sprintf("%s:%s:%s", bindSource, destination, mode), bindSource
}

func matchRuntimeMountDestination(destination, workingDir string) bool {
	destination = filepath.Clean(strings.TrimSpace(destination))
	workingDir = filepath.Clean(strings.TrimSpace(workingDir))
	if destination == "" || workingDir == "" {
		return false
	}
	return destination == workingDir || strings.HasPrefix(destination, workingDir+string(os.PathSeparator))
}

func sameRuntimeImageRef(currentImage, targetImage string) bool {
	return normalizeRuntimeImageRef(currentImage) == normalizeRuntimeImageRef(targetImage)
}

func sameRuntimeImageRepo(currentImage, targetImage string) bool {
	currentRepo := normalizeRuntimeImageRef(currentImage)
	targetRepo := normalizeRuntimeImageRef(targetImage)
	if currentRepo == "" || targetRepo == "" {
		return false
	}
	if idx := strings.LastIndex(currentRepo, ":"); idx > strings.LastIndex(currentRepo, "/") {
		currentRepo = currentRepo[:idx]
	}
	if idx := strings.LastIndex(targetRepo, ":"); idx > strings.LastIndex(targetRepo, "/") {
		targetRepo = targetRepo[:idx]
	}
	return currentRepo == targetRepo
}

func normalizeRuntimeImageRef(imageRef string) string {
	imageRef = strings.TrimSpace(imageRef)
	imageRef = strings.Trim(imageRef, "`\"'")
	imageRef = strings.TrimPrefix(imageRef, "docker.io/library/")
	return imageRef
}

func detectPreviousContainerMountDirs(ctx context.Context, cli *dockerclient.Client, containerID, workingDir string) []string {
	containerID = strings.TrimSpace(containerID)
	workingDir = filepath.Clean(strings.TrimSpace(workingDir))
	if containerID == "" || workingDir == "" {
		return nil
	}

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil
	}

	var exactMatches []string
	var parentMatches []string
	seen := make(map[string]struct{})
	for _, mountPoint := range inspect.Mounts {
		source := strings.TrimSpace(mountPoint.Source)
		destination := filepath.Clean(strings.TrimSpace(mountPoint.Destination))
		if source == "" || destination == "" {
			continue
		}
		if _, ok := seen[source]; ok {
			continue
		}
		switch {
		case destination == workingDir:
			exactMatches = append(exactMatches, source)
			seen[source] = struct{}{}
		case strings.HasPrefix(workingDir, destination+string(os.PathSeparator)):
			parentMatches = append(parentMatches, source)
			seen[source] = struct{}{}
		}
	}

	return append(exactMatches, parentMatches...)
}
