package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/image"
)

type websiteImageDeployRequest struct {
	Alias               string
	Image               string
	PreviousContainerID string
}

func deployWebsiteImage(ctx context.Context, request websiteImageDeployRequest, progress func(format string, a ...interface{})) (int, string, error) {
	imageName := strings.TrimSpace(request.Image)
	if imageName == "" {
		imageName = "node:20-alpine"
	}
	result, err := deployRuntimeContainer(ctx, imageName, func(imageInspect image.InspectResponse) (runtimeContainerSpec, error) {
		spec := buildWebsiteImageRuntimeSpec(request, imageInspect)
		logEngineProgress(progress, "镜像运行配置: workingDir=%s, containerPort=%s", spec.WorkingDir, spec.ContainerPort)
		return spec, nil
	}, progress)
	if err != nil {
		return 0, "", err
	}
	return result.HostPort, result.ContainerID, nil
}

func buildWebsiteImageRuntimeSpec(request websiteImageDeployRequest, imageInspect image.InspectResponse) runtimeContainerSpec {
	var envs []string
	if imageInspect.Config != nil {
		envs = append(envs, imageInspect.Config.Env...)
	}
	containerPort := detectEngineContainerPort(imageInspect)
	workingDir := detectEngineWorkingDir(imageInspect)
	return runtimeContainerSpec{
		Alias:               strings.TrimSpace(request.Alias),
		ContainerPort:       containerPort,
		PublishedHostPort:   "0",
		WorkingDir:          workingDir,
		Env:                 envs,
		PreviousContainerID: strings.TrimSpace(request.PreviousContainerID),
	}
}

func validateWebsiteImageSource(codeSource string) error {
	if strings.TrimSpace(codeSource) != "git" {
		return fmt.Errorf("unsupported container deployment source: %s", codeSource)
	}
	return nil
}
