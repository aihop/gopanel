package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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

// 端口绑定阶段的诊断：容器压根没把端口绑上来。
func buildEngineContainerDiagError(ctx context.Context, cli *dockerclient.Client, containerID, containerPort string, inspect container.InspectResponse) error {
	parts := append([]string{fmt.Sprintf("containerPort=%s", containerPort)},
		collectEngineContainerDiagParts(ctx, cli, containerID, inspect)...)
	return fmt.Errorf("could not find bound port for engine container (%s)", strings.Join(parts, ", "))
}

// 就绪等待阶段的诊断：端口早已绑好，是 Runner 在构建/启动完成前自己退出或被重启了，
// 复用上面那句「找不到绑定端口」会把人往完全错误的方向带。
func buildRunnerNotReadyDiagError(ctx context.Context, cli *dockerclient.Client, containerID, containerPort string, hostPort int, reason string, inspect container.InspectResponse) error {
	parts := append([]string{
		fmt.Sprintf("containerPort=%s", containerPort),
		fmt.Sprintf("hostPort=%d", hostPort),
	}, collectEngineContainerDiagParts(ctx, cli, containerID, inspect)...)
	return fmt.Errorf("%s (%s)", reason, strings.Join(parts, ", "))
}

func collectEngineContainerDiagParts(ctx context.Context, cli *dockerclient.Client, containerID string, inspect container.InspectResponse) []string {
	var parts []string
	var state *container.State
	if inspect.ContainerJSONBase != nil {
		state = inspect.ContainerJSONBase.State
		if inspect.ContainerJSONBase.RestartCount > 0 {
			parts = append(parts, fmt.Sprintf("restartCount=%d", inspect.ContainerJSONBase.RestartCount))
		}
	}
	if state != nil {
		parts = append(parts, fmt.Sprintf("state=%s", state.Status))
		if state.ExitCode != 0 {
			parts = append(parts, fmt.Sprintf("exitCode=%d", state.ExitCode))
		}
		if state.Error != "" {
			parts = append(parts, fmt.Sprintf("dockerError=%s", state.Error))
		}
	}
	rawLogs := readEngineContainerLogTail(ctx, cli, containerID)
	if hint := detectEngineContainerKillHint(state, rawLogs); hint != "" {
		parts = append(parts, hint)
	}
	if logs := formatEngineContainerLogs(rawLogs); logs != "" {
		parts = append(parts, fmt.Sprintf("logs=%s", logs))
	}
	return parts
}

// detectEngineContainerKillHint 补上 state 判断不了的场景：被 OOM 杀掉的往往是构建子进程，
// PID 1 的 shell 只是 set -e 正常退出 1，此时 OOMKilled=false、ExitCode≠137，
// 唯一的线索留在日志里（SIGKILL / Killed / heap out of memory）。
func detectEngineContainerKillHint(state *container.State, rawLogs string) string {
	if state != nil && state.OOMKilled {
		return "oomKilled=true（容器内存不足，构建进程被系统终止）"
	}
	if state != nil && state.ExitCode == 137 {
		return "exitSignal=SIGKILL（进程被强制终止，通常是内存不足）"
	}
	if line := findEngineContainerKillLine(rawLogs); line != "" {
		return fmt.Sprintf("possibleOOM=true（日志出现内存不足/强制终止特征，请确认容器运行时可用内存: %s）", line)
	}
	return ""
}

func findEngineContainerKillLine(rawLogs string) string {
	if strings.TrimSpace(rawLogs) == "" {
		return ""
	}
	caseSensitive := []string{"SIGKILL", "Killed", "OOMKilled"}
	caseInsensitive := []string{
		"out of memory",
		"heap limit",
		"cannot allocate memory",
		"exit code 137",
		"signal 9",
	}
	for _, line := range strings.Split(rawLogs, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		matched := false
		for _, keyword := range caseSensitive {
			if strings.Contains(trimmed, keyword) {
				matched = true
				break
			}
		}
		if !matched {
			for _, keyword := range caseInsensitive {
				if strings.Contains(lower, keyword) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}
		if len(trimmed) > engineDiagKillLineLimit {
			trimmed = trimmed[:runeSafeCut(trimmed, engineDiagKillLineLimit)] + "..."
		}
		return trimmed
	}
	return ""
}

const (
	// 构建日志很容易被上百行 warning 刷屏，tail 太短会把真正的死因冲掉。
	engineDiagLogTailLines  = "200"
	engineDiagLogLimit      = 1200
	engineDiagKillLineLimit = 200
)

func readEngineContainerLogTail(ctx context.Context, cli *dockerclient.Client, containerID string) string {
	if cli == nil {
		return ""
	}
	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       engineDiagLogTailLines,
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
	return strings.TrimSpace(logs)
}

func formatEngineContainerLogs(rawLogs string) string {
	logs := strings.TrimSpace(rawLogs)
	if logs == "" {
		return ""
	}
	logs = strings.ReplaceAll(logs, "\n", " | ")
	// 保留末尾而不是开头：失败原因总在日志最后，截断开头才有意义。
	if len(logs) > engineDiagLogLimit {
		logs = "..." + logs[runeSafeCut(logs, len(logs)-engineDiagLogLimit):]
	}
	return logs
}

// 日志里中文很常见，按字节切会切出半个字符，这里对齐到最近的 rune 边界。
func runeSafeCut(s string, offset int) int {
	if offset <= 0 {
		return 0
	}
	if offset >= len(s) {
		return len(s)
	}
	for offset < len(s) && !utf8.RuneStart(s[offset]) {
		offset++
	}
	return offset
}

func startEngineContainerLogStreaming(ctx context.Context, cli *dockerclient.Client, containerID string, progress func(format string, a ...interface{})) context.CancelFunc {
	if progress == nil {
		return func() {}
	}
	streamCtx, cancel := context.WithCancel(ctx)
	go func() {
		logEngineProgress(progress, "已桥接 Runner 容器日志（最近 50 行 + 实时跟随）")
		reader, err := cli.ContainerLogs(streamCtx, containerID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Tail:       "50",
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(streamCtx.Err(), context.Canceled) || strings.Contains(err.Error(), "context canceled") {
				return
			}
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
