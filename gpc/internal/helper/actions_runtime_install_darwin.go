//go:build darwin

package helper

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

func (s *Server) actionContainerRuntimeInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	runtimeKind, err := validateRuntimeInstallKind(params)
	if err != nil {
		return "", err
	}
	uid, ok := getInt(params, "uid")
	if !ok || uid <= 0 {
		return "", errors.New("invalid params: a non-root runtime user is required on macOS")
	}
	username := strings.TrimSpace(getString(params, "username"))
	account, err := user.LookupId(strconv.Itoa(uid))
	if err != nil || account.Username != username {
		return "", errors.New("invalid params: runtime user does not match uid")
	}
	brew, err := findBrewBinary()
	if err != nil {
		return "", err
	}

	var output strings.Builder
	brewArgs := []string{"/usr/bin/env", "HOME=" + account.HomeDir, "PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin", brew, "install"}
	if runtimeKind == "docker" {
		brewArgs = append(brewArgs, "--cask", "docker")
	} else {
		brewArgs = append(brewArgs, "podman", "podman-compose")
	}
	if err := runInstallCommand(ctx, &output, nil, append([]string{"/usr/bin/sudo", "-H", "-u", username}, brewArgs...)...); err != nil {
		return strings.TrimSpace(output.String()), err
	}
	if runtimeKind == "docker" {
		return encodeRuntimeInstallResult(runtimeInstallResult{
			Runtime: runtimeKind, Message: "Docker Desktop installed", NeedsAction: "openDockerDesktop", Output: strings.TrimSpace(output.String()),
		})
	}

	userCommand := func(args ...string) error {
		base := []string{"/usr/bin/sudo", "-H", "-u", username, "/usr/bin/env", "HOME=" + account.HomeDir, "PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"}
		return runInstallCommand(ctx, &output, nil, append(base, args...)...)
	}
	if err := userCommand("podman", "info"); err != nil {
		if inspectErr := userCommand("podman", "machine", "inspect"); inspectErr != nil {
			if initErr := userCommand("podman", "machine", "init"); initErr != nil {
				return strings.TrimSpace(output.String()), fmt.Errorf("podman machine init failed: %w", initErr)
			}
		}
		if startErr := userCommand("podman", "machine", "start"); startErr != nil {
			return strings.TrimSpace(output.String()), fmt.Errorf("podman machine start failed: %w", startErr)
		}
	}
	return encodeRuntimeInstallResult(runtimeInstallResult{Runtime: runtimeKind, Message: "Podman installed and machine started", Output: strings.TrimSpace(output.String())})
}

func findBrewBinary() (string, error) {
	for _, candidate := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("Homebrew is required to install a container runtime on macOS")
}
