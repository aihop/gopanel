//go:build linux

package helper

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func (s *Server) actionComposeInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = params
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if _, err := exec.LookPath("podman-compose"); err == nil {
		return "podman-compose already installed", nil
	}

	var out strings.Builder
	run := func(env []string, args ...string) error {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		if len(env) > 0 {
			cmd.Env = append(os.Environ(), env...)
		}
		b, err := cmd.CombinedOutput()
		if len(b) > 0 {
			out.Write(b)
			if b[len(b)-1] != '\n' {
				out.WriteByte('\n')
			}
		}
		return err
	}

	if _, err := exec.LookPath("apt-get"); err == nil {
		_ = run([]string{"DEBIAN_FRONTEND=noninteractive"}, "apt-get", "update", "-y")
		if err := run([]string{"DEBIAN_FRONTEND=noninteractive"}, "apt-get", "install", "-y", "podman-compose"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		if err := run(nil, "dnf", "install", "-y", "podman-compose"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("yum"); err == nil {
		if err := run(nil, "yum", "install", "-y", "podman-compose"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("apk"); err == nil {
		if err := run(nil, "apk", "add", "--no-cache", "podman-compose"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("pacman"); err == nil {
		if err := run(nil, "pacman", "-Sy", "--noconfirm", "podman-compose"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("zypper"); err == nil {
		if err := run(nil, "zypper", "--non-interactive", "in", "-y", "podman-compose"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else {
		return "", errors.New("no supported package manager found (apt/dnf/yum/apk/pacman/zypper)")
	}

	if _, err := exec.LookPath("podman-compose"); err != nil {
		return strings.TrimSpace(out.String()), errors.New("podman-compose still not found after installation")
	}
	_ = run(nil, "podman-compose", "version")
	return strings.TrimSpace(out.String()), nil
}

