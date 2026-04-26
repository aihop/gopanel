//go:build linux

package helper

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func (s *Server) actionMysqlClientInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = params
	if hasMysqlClientCommands() {
		return "mysql client already installed", nil
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
		if err := run([]string{"DEBIAN_FRONTEND=noninteractive"}, "apt-get", "install", "-y", "default-mysql-client"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		if err := run(nil, "dnf", "install", "-y", "mysql"); err != nil {
			_ = run(nil, "dnf", "install", "-y", "mariadb")
		}
	} else if _, err := exec.LookPath("yum"); err == nil {
		if err := run(nil, "yum", "install", "-y", "mysql"); err != nil {
			_ = run(nil, "yum", "install", "-y", "mariadb")
		}
	} else if _, err := exec.LookPath("apk"); err == nil {
		if err := run(nil, "apk", "add", "--no-cache", "mysql-client"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("pacman"); err == nil {
		if err := run(nil, "pacman", "-Sy", "--noconfirm", "mariadb-clients"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else if _, err := exec.LookPath("zypper"); err == nil {
		if err := run(nil, "zypper", "--non-interactive", "in", "-y", "mariadb-client"); err != nil {
			return strings.TrimSpace(out.String()), err
		}
	} else {
		return "", errors.New("no supported package manager found for mysql client install")
	}

	if !hasMysqlClientCommands() {
		return strings.TrimSpace(out.String()), errors.New("mysql client still not found after installation")
	}
	return strings.TrimSpace(out.String()), nil
}

func hasMysqlClientCommands() bool {
	if _, err := exec.LookPath("mysql"); err == nil {
		if _, err := exec.LookPath("mysqldump"); err == nil {
			return true
		}
	}
	if _, err := exec.LookPath("mariadb"); err == nil {
		if _, err := exec.LookPath("mariadb-dump"); err == nil {
			return true
		}
	}
	return false
}

