package service

import (
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	container "github.com/docker/docker/api/types/container"
	"net"
	"strconv"
	"strings"
	"time"
)

func resolvePipelineRunnerBridge(website *model.Website, pipelineRecordID uint) (int, string, string, bool, error) {
	if website == nil || website.PipelineID == 0 {
		return 0, "", "", false, nil
	}
	pipeline, err := repo.NewPipeline(global.DB).Get(website.PipelineID)
	if err != nil {
		return 0, "", "", false, fmt.Errorf("读取流水线配置失败: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(pipeline.RunnerMode), "runner") {
		return 0, "", "", false, nil
	}
	recordRepo := repo.NewPipelineRecord(global.DB)
	var record *model.PipelineRecord
	if pipelineRecordID > 0 {
		rec, err := recordRepo.Get(pipelineRecordID)
		if err == nil && rec != nil && rec.PipelineID == website.PipelineID {
			record = rec
		}
	}
	if record == nil {
		rec, err := recordRepo.LatestByPipelineID(website.PipelineID)
		if err != nil {
			return 0, "", "", false, nil
		}
		record = rec
	}
	if record == nil || record.RunnerHostPort <= 0 {
		return 0, "", "", false, nil
	}
	return record.RunnerHostPort, strings.TrimSpace(record.RunnerContainerID), strings.TrimSpace(record.RunnerReleaseDir), true, nil
}
func resolvePipelineScriptProxyTarget(website *model.Website, pipelineRecordID uint) (int, string, bool, error) {
	if website == nil || website.PipelineID == 0 {
		return 0, "", false, nil
	}
	pipeline, err := repo.NewPipeline(global.DB).Get(website.PipelineID)
	if err != nil {
		return 0, "", false, fmt.Errorf("读取流水线配置失败: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(pipeline.RunnerMode), "runner") {
		return 0, "", false, nil
	}
	containerName := strings.TrimSpace(pipeline.PipelineKey)
	if containerName == "" {
		containerName = strings.TrimSpace(website.Alias)
	}
	if containerName == "" {
		return 0, "", false, fmt.Errorf("纯脚本流水线缺少稳定容器名，无法自动识别运行端口；请为流水线设置 pipelineKey")
	}
	hostPort, containerID, err := detectScriptRuntimePortByContainerName(containerName, pipeline.ExposePort)
	if err != nil {
		return 0, "", false, fmt.Errorf("纯脚本流水线未检测到可用容器端口，请确认脚本已成功启动容器 %s: %w", containerName, err)
	}
	if pipelineRecordID > 0 {
		_ = repo.NewPipelineRecord(global.DB).UpdateRunnerResult(pipelineRecordID, "", containerID, hostPort)
	}
	return hostPort, containerID, true, nil
}
func detectScriptRuntimePortByContainerName(containerName string, preferredPort int) (int, string, error) {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return 0, "", err
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(context.Background(), containerName)
	if err != nil {
		return 0, "", err
	}
	if inspect.State == nil || !inspect.State.Running {
		return 0, "", fmt.Errorf("容器 %s 未在运行", containerName)
	}
	hostPort, err := choosePublishedHostPort(inspect, preferredPort)
	if err != nil {
		return 0, "", err
	}
	if err := verifyLocalProxyPortReachable(hostPort); err != nil {
		return 0, "", fmt.Errorf("容器 %s 已运行，但宿主机端口 %d 当前不可访问: %w", containerName, hostPort, err)
	}
	return hostPort, inspect.ID, nil
}
func choosePublishedHostPort(inspect container.InspectResponse, preferredPort int) (int, error) {
	portBindings := inspect.NetworkSettings.Ports
	if len(portBindings) == 0 && inspect.ContainerJSONBase != nil && inspect.ContainerJSONBase.HostConfig != nil {
		portBindings = inspect.ContainerJSONBase.HostConfig.PortBindings
	}
	if len(portBindings) == 0 {
		return 0, fmt.Errorf("容器没有可用的端口映射")
	}
	type portCandidate struct {
		hostPort    int
		privatePort int
	}
	seen := make(map[int]portCandidate)
	for key, bindings := range portBindings {
		privatePort, err := strconv.Atoi(key.Port())
		if err != nil {
			continue
		}
		for _, binding := range bindings {
			hostPort, err := strconv.Atoi(strings.TrimSpace(binding.HostPort))
			if err != nil || hostPort <= 0 {
				continue
			}
			hostIP := strings.TrimSpace(binding.HostIP)
			if hostIP != "" && hostIP != "127.0.0.1" && hostIP != "0.0.0.0" && hostIP != "::" && hostIP != "::1" {
				continue
			}
			if _, ok := seen[hostPort]; !ok {
				seen[hostPort] = portCandidate{hostPort: hostPort, privatePort: privatePort}
			}
		}
	}
	if len(seen) == 0 {
		return 0, fmt.Errorf("容器没有可识别的宿主机端口映射")
	}
	if preferredPort > 0 {
		for _, candidate := range seen {
			if candidate.privatePort == preferredPort || candidate.hostPort == preferredPort {
				return candidate.hostPort, nil
			}
		}
	}
	if len(seen) == 1 {
		for _, candidate := range seen {
			return candidate.hostPort, nil
		}
	}
	return 0, fmt.Errorf("容器存在多个端口映射，无法自动判断入口端口")
}
func verifyLocalProxyPortReachable(port int) error {
	target := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
func cleanupPreviousContainer(containerID string) error {
	cli, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return RemoveEngineContainer(context.Background(), cli, containerID)
}
