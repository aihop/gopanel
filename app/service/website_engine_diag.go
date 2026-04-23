package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

func waitForEnginePortBinding(ctx context.Context, cli *dockerclient.Client, containerID, containerPort string) ([]nat.PortBinding, error) {
	targetPort := nat.Port(containerPort + "/tcp")
	var lastInspectErr error

	for i := 0; i < 10; i++ {
		inspect, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			lastInspectErr = err
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if bindings, ok := inspect.NetworkSettings.Ports[targetPort]; ok && len(bindings) > 0 {
			return bindings, nil
		}
		if inspect.ContainerJSONBase != nil && inspect.ContainerJSONBase.State != nil && !inspect.ContainerJSONBase.State.Running {
			return nil, buildEngineContainerDiagError(ctx, cli, containerID, containerPort, inspect)
		}
		time.Sleep(300 * time.Millisecond)
	}

	if lastInspectErr != nil {
		return nil, fmt.Errorf("failed to inspect engine container: %w", lastInspectErr)
	}

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect engine container: %w", err)
	}
	return nil, buildEngineContainerDiagError(ctx, cli, containerID, containerPort, inspect)
}

func buildEngineContainerDiagError(ctx context.Context, cli *dockerclient.Client, containerID, containerPort string, inspect container.InspectResponse) error {
	var parts []string
	if inspect.ContainerJSONBase != nil && inspect.ContainerJSONBase.State != nil {
		state := inspect.ContainerJSONBase.State
		parts = append(parts, fmt.Sprintf("state=%s", state.Status))
		if state.ExitCode != 0 {
			parts = append(parts, fmt.Sprintf("exitCode=%d", state.ExitCode))
		}
		if state.Error != "" {
			parts = append(parts, fmt.Sprintf("dockerError=%s", state.Error))
		}
	}
	if logs := readEngineContainerLogs(ctx, cli, containerID); logs != "" {
		parts = append(parts, fmt.Sprintf("logs=%s", logs))
	}
	return fmt.Errorf("could not find bound port for engine container (containerPort=%s, %s)", containerPort, strings.Join(parts, ", "))
}

func readEngineContainerLogs(ctx context.Context, cli *dockerclient.Client, containerID string) string {
	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "50",
	})
	if err != nil {
		return ""
	}
	defer reader.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader); err != nil {
		return ""
	}

	logs := strings.TrimSpace(stdoutBuf.String())
	if errLogs := strings.TrimSpace(stderrBuf.String()); errLogs != "" {
		if logs != "" {
			logs += "\n"
		}
		logs += errLogs
	}
	logs = strings.TrimSpace(logs)
	if logs == "" {
		return ""
	}
	logs = strings.ReplaceAll(logs, "\n", " | ")
	if len(logs) > 600 {
		logs = logs[:600] + "..."
	}
	return logs
}

func startEngineContainerLogStreaming(ctx context.Context, cli *dockerclient.Client, containerID string, progress func(format string, a ...interface{})) context.CancelFunc {
	if progress == nil {
		return func() {}
	}
	streamCtx, cancel := context.WithCancel(ctx)
	go func() {
		reader, err := cli.ContainerLogs(streamCtx, containerID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Tail:       "0",
		})
		if err != nil {
			logEngineProgress(progress, "附加 Runner 容器日志失败: %v", err)
			return
		}
		defer reader.Close()

		writer := &progressLineWriter{
			progress: progress,
			prefix:   "[Runner Log] ",
		}
		defer writer.Flush()
		_, _ = stdcopy.StdCopy(writer, writer, reader)
	}()
	return cancel
}

type progressLineWriter struct {
	progress func(format string, a ...interface{})
	prefix   string
	buffer   strings.Builder
}

func (w *progressLineWriter) Write(p []byte) (int, error) {
	s := string(p)
	for len(s) > 0 {
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			w.buffer.WriteString(s)
			break
		}
		w.buffer.WriteString(s[:idx])
		w.flushLine()
		s = s[idx+1:]
	}
	return len(p), nil
}

func (w *progressLineWriter) Flush() {
	w.flushLine()
}

func (w *progressLineWriter) flushLine() {
	line := strings.TrimSpace(w.buffer.String())
	w.buffer.Reset()
	if line == "" {
		return
	}
	logEngineProgress(w.progress, "%s%s", w.prefix, line)
}

func RemoveEngineContainer(ctx context.Context, cli *dockerclient.Client, containerID string) error {
	err := cli.ContainerStop(ctx, containerID, container.StopOptions{})
	if err != nil {
		global.LOG.Errorf("Failed to stop container %s: %v", containerID, err)
	}
	return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}
