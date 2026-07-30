//go:build !windows

package helper

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
)

type runtimeInstallResult struct {
	Runtime     string `json:"runtime"`
	Message     string `json:"message"`
	NeedsAction string `json:"needsAction,omitempty"`
	Output      string `json:"output,omitempty"`
}

func encodeRuntimeInstallResult(result runtimeInstallResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func validateRuntimeInstallKind(params map[string]interface{}) (string, error) {
	runtimeKind := strings.ToLower(strings.TrimSpace(getString(params, "runtime")))
	if runtimeKind != "docker" && runtimeKind != "podman" {
		return "", errors.New("invalid params: runtime must be docker or podman")
	}
	return runtimeKind, nil
}

func runInstallCommand(ctx context.Context, output *strings.Builder, env []string, args ...string) error {
	if len(args) == 0 {
		return errors.New("empty install command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	data, err := cmd.CombinedOutput()
	if len(data) > 0 {
		output.Write(data)
		if data[len(data)-1] != '\n' {
			output.WriteByte('\n')
		}
	}
	return err
}
