package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

const (
	runnerWorkspaceMountPath = "/gopanel/workspace"
	runnerNetworkName        = "gopanel-network"
)

type runnerConfig struct {
	Mode                string
	BaseImage           string
	WorkingDir          string
	HasCustomWorkingDir bool
	ContainerPort       string
	HostPort            string
	RunnerUser          string
	StartCommand        string
	BuildCommand        string
	PreStart            string
	Env                 map[string]string
	PersistentPaths     []string
	ExtraNetworks       []string
}

type pipelineRunnerDeployRequest struct {
	Alias               string
	CodeRoot            string
	PipelineKey         string
	PipelineVersion     string
	PreviousContainerID string
	Config              map[string]interface{}
}

func deployPipelineRunner(ctx context.Context, request pipelineRunnerDeployRequest, progress func(format string, a ...interface{})) (int, string, error) {
	rc := parseRunnerConfig(request.Config)
	imageName := strings.TrimSpace(rc.BaseImage)
	if imageName == "" {
		imageName = "node:20-alpine"
	}
	result, err := deployRuntimeContainer(ctx, imageName, func(imageInspect image.InspectResponse) (runtimeContainerSpec, error) {
		if err := docker.EnsureNetwork(runnerNetworkName); err != nil {
			return runtimeContainerSpec{}, fmt.Errorf("failed to ensure runner network %s: %w", runnerNetworkName, err)
		}
		spec, err := buildPipelineRunnerRuntimeSpec(request, rc, imageInspect)
		if err != nil {
			return runtimeContainerSpec{}, err
		}
		sourceMountDir := resolveRunnerSourceMountDir(rc, spec.WorkingDir)
		persistentBindCount := len(spec.Binds) - 1
		logPipelineRunnerSpec(progress, imageName, rc, spec, request.CodeRoot, sourceMountDir, persistentBindCount)
		return spec, nil
	}, progress)
	if err != nil {
		return 0, "", err
	}
	return result.HostPort, result.ContainerID, nil
}

func buildPipelineRunnerRuntimeSpec(request pipelineRunnerDeployRequest, rc runnerConfig, imageInspect image.InspectResponse) (runtimeContainerSpec, error) {
	codeRoot := strings.TrimSpace(request.CodeRoot)
	if codeRoot == "" {
		return runtimeContainerSpec{}, fmt.Errorf("runner 代码目录为空")
	}

	var imageEnvs []string
	if imageInspect.Config != nil {
		imageEnvs = imageInspect.Config.Env
	}
	containerPort := strings.TrimSpace(rc.ContainerPort)
	if containerPort == "" {
		containerPort = detectEngineContainerPort(imageInspect)
	}
	workingDir := strings.TrimSpace(rc.WorkingDir)
	if workingDir == "" {
		workingDir = detectEngineWorkingDir(imageInspect)
	}
	sourceMountDir := resolveRunnerSourceMountDir(rc, workingDir)

	spec := runtimeContainerSpec{
		Alias:               strings.TrimSpace(request.Alias),
		ContainerPort:       containerPort,
		PublishedHostPort:   resolveRunnerPublishedHostPort(rc),
		WorkingDir:          workingDir,
		User:                strings.TrimSpace(rc.RunnerUser),
		Env:                 mergeRunnerEnvs(imageEnvs, rc, containerPort, request.PipelineVersion),
		Cmd:                 []string{"sh", "-lc", buildRunnerScript(rc, sourceMountDir)},
		NetworkMode:         container.NetworkMode(runnerNetworkName),
		PreviousContainerID: strings.TrimSpace(request.PreviousContainerID),
		WaitForReady:        true,
	}
	if rc.HasCustomWorkingDir {
		spec.Binds = append(spec.Binds, fmt.Sprintf("%s:%s", codeRoot, sourceMountDir))
	} else {
		spec.Binds = append(spec.Binds, fmt.Sprintf("%s:%s:ro", codeRoot, sourceMountDir))
	}
	for _, networkName := range rc.ExtraNetworks {
		if networkName != runnerNetworkName {
			spec.ExtraNetworks = append(spec.ExtraNetworks, networkName)
		}
	}
	persistentBinds, err := buildRunnerPersistentBinds(request.PipelineKey, rc, workingDir)
	if err != nil {
		return runtimeContainerSpec{}, fmt.Errorf("failed to prepare runner persistent dirs: %w", err)
	}
	spec.Binds = append(spec.Binds, persistentBinds...)
	spec.Mounts = buildRunnerEphemeralMounts(rc, workingDir, codeRoot)
	return spec, nil
}

func logPipelineRunnerSpec(progress func(format string, a ...interface{}), imageName string, rc runnerConfig, spec runtimeContainerSpec, codeRoot, sourceMountDir string, persistentBindCount int) {
	logEngineProgress(progress, "镜像运行配置: workingDir=%s, containerPort=%s", spec.WorkingDir, spec.ContainerPort)
	logEngineProgress(progress, "Runner 配置: baseImage=%s, mode=%s", strings.TrimSpace(imageName), strings.TrimSpace(rc.Mode))
	logEngineProgress(progress, "Runner 目录语义: sourceMountDir=%s, workingDir=%s", sourceMountDir, spec.WorkingDir)
	logEngineProgress(progress, "Runner 启动脚本已生成")
	logEngineProgress(progress, "Runner 默认接入网络: %s", runnerNetworkName)
	if len(spec.ExtraNetworks) > 0 {
		logEngineProgress(progress, "Runner 额外接入网络: %s", strings.Join(spec.ExtraNetworks, ", "))
	}
	logEngineProgress(progress, "Runner 项目类型: %s", detectRunnerProjectKind(codeRoot, rc))
	if rc.HasCustomWorkingDir {
		logEngineProgress(progress, "Runner 使用自定义 workingDir，代码源直接挂载到最终运行目录: %s -> %s", codeRoot, sourceMountDir)
	} else {
		logEngineProgress(progress, "Runner 未自定义 workingDir，代码源先挂到只读中间目录: %s -> %s (ro)", codeRoot, sourceMountDir)
	}
	if persistentBindCount > 0 {
		logEngineProgress(progress, "已追加 %d 个持久化目录映射", persistentBindCount)
	}
	if len(spec.Mounts) > 0 {
		logEngineProgress(progress, "Runner 依赖隔离: 已启用 node_modules 临时卷（Node 项目）")
	} else {
		logEngineProgress(progress, "Runner 依赖隔离: 未启用（非 Node 项目）")
	}
	if spec.User != "" {
		logEngineProgress(progress, "Runner 容器内运行用户: %s", spec.User)
	}
}
