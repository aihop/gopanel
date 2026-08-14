package service

import (
	"context"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestBuildEngineContainerDiagErrorReportsOOMKill(t *testing.T) {
	err := buildEngineContainerDiagError(context.Background(), nil, "runner", "3000", container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			State: &container.State{Status: "exited", ExitCode: 137, OOMKilled: true},
		},
	})
	for _, expected := range []string{"containerPort=3000", "exitCode=137", "oomKilled=true", "内存不足"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected diagnostic to contain %q: %v", expected, err)
		}
	}
}

func TestBuildRunnerNotReadyDiagErrorUsesContainerPortNotHostPort(t *testing.T) {
	err := buildRunnerNotReadyDiagError(context.Background(), nil, "runner", "3000", 41777,
		"Runner 在构建/启动完成前被重启", container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				RestartCount: 1,
				State:        &container.State{Status: "exited", ExitCode: 1},
			},
		})
	message := err.Error()
	for _, expected := range []string{"Runner 在构建/启动完成前被重启", "containerPort=3000", "hostPort=41777", "restartCount=1", "exitCode=1"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected diagnostic to contain %q: %v", expected, err)
		}
	}
	if strings.Contains(message, "containerPort=41777") {
		t.Fatalf("host port must not be reported as container port: %v", err)
	}
	if strings.Contains(message, "could not find bound port") {
		t.Fatalf("readiness failure must not claim the port was never bound: %v", err)
	}
}

// 真实场景：被 OOM 杀掉的是 nuxt 子进程，PID 1 的 shell 只是 set -e 退出 1，
// 所以 OOMKilled=false、ExitCode=1，唯一线索在日志里。
func TestDetectEngineContainerKillHintFindsSignalInLogs(t *testing.T) {
	logs := strings.Join([]string{
		"WARN  [plugin @tailwindcss/vite:generate:build] Sourcemap is likely to be incorrect",
		"[build] Nuxt 构建被信号 SIGKILL 中止",
		"[RUNNER] syncing source into working dir",
	}, "\n")
	hint := detectEngineContainerKillHint(&container.State{Status: "exited", ExitCode: 1}, logs)
	if !strings.Contains(hint, "possibleOOM=true") || !strings.Contains(hint, "SIGKILL") {
		t.Fatalf("expected SIGKILL log line to raise an OOM hint, got %q", hint)
	}
}

func TestDetectEngineContainerKillHintIgnoresNormalLogs(t *testing.T) {
	logs := "[RUNNER] syncing source into working dir\nadded 1151 packages"
	if hint := detectEngineContainerKillHint(&container.State{Status: "exited", ExitCode: 1}, logs); hint != "" {
		t.Fatalf("expected no kill hint for a clean build log, got %q", hint)
	}
}

func TestFormatEngineContainerLogsKeepsTailAndRuneBoundary(t *testing.T) {
	raw := strings.Repeat("构建噪音警告\n", 400) + "[build] Nuxt 构建被信号 SIGKILL 中止"
	logs := formatEngineContainerLogs(raw)
	if !strings.HasSuffix(logs, "[build] Nuxt 构建被信号 SIGKILL 中止") {
		t.Fatalf("expected the tail of the log to survive truncation, got %q", logs)
	}
	if !strings.HasPrefix(logs, "...") {
		t.Fatalf("expected truncated logs to be marked with a leading ellipsis, got %q", logs)
	}
	if strings.ContainsRune(logs, '�') {
		t.Fatalf("truncation split a multi-byte rune: %q", logs)
	}
}
