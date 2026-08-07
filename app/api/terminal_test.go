package api

import (
	"context"
	"errors"
	"testing"
)

func TestEnsureContainerTerminalReady(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		inspectErr error
		wantErr    string
	}{
		{name: "running", output: "true\n"},
		{name: "stopped", output: "false\n", wantErr: "容器当前未运行，请启动容器后重试"},
		{name: "missing", output: "Error: no such container", inspectErr: errors.New("exit status 125"), wantErr: "容器不存在或运行时已变化，请刷新列表后重试"},
		{name: "improper state", output: "Error: can only create exec sessions on running containers: container state improper", inspectErr: errors.New("exit status 125"), wantErr: "容器当前未运行，请启动容器后重试"},
		{name: "runtime unavailable", output: "connection refused", inspectErr: errors.New("exit status 125"), wantErr: "无法读取容器状态，请确认容器运行时可用后重试"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inspect := func(context.Context, string, string) ([]byte, error) {
				return []byte(test.output), test.inspectErr
			}
			err := ensureContainerTerminalReadyWith(context.Background(), "container-id", "runtime-host", inspect)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("running container rejected: %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErr {
				t.Fatalf("unexpected error: %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestNormalizeContainerTerminalStartError(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "container state improper", want: "容器当前未运行，请启动容器后重试"},
		{input: "no such container", want: "容器不存在或运行时已变化，请刷新列表后重试"},
		{input: "permission denied", want: "启动容器终端失败，请确认容器和运行时状态后重试"},
	}
	for _, test := range tests {
		if got := normalizeContainerTerminalStartError(errors.New(test.input)); got == nil || got.Error() != test.want {
			t.Fatalf("normalize %q: got %v, want %q", test.input, got, test.want)
		}
	}
}

func TestEnsureContainerTerminalReadyTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	inspect := func(ctx context.Context, _, _ string) ([]byte, error) {
		return nil, ctx.Err()
	}
	err := ensureContainerTerminalReadyWith(ctx, "container-id", "runtime-host", inspect)
	if err == nil || err.Error() != "连接容器运行时超时，请稍后重试" {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}
