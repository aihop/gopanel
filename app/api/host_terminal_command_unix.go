//go:build !windows

package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

func buildHostTerminalCommand(shellName, workDir string) (*exec.Cmd, string, error) {
	allowed := map[string]string{"bash": "bash", "sh": "sh", "zsh": "zsh"}
	shellName = strings.ToLower(strings.TrimSpace(shellName))
	if shellName == "" || shellName == "default" {
		shellName = filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
		if _, ok := allowed[shellName]; !ok {
			shellName = "bash"
		}
	}
	binary, ok := allowed[shellName]
	if !ok {
		return nil, "", buserr.New(constant.ErrHostTerminalShellUnsupported)
	}
	path, err := exec.LookPath(binary)
	if err != nil && shellName == "bash" {
		shellName = "sh"
		path, err = exec.LookPath("sh")
	}
	if err != nil {
		return nil, "", buserr.New(constant.ErrHostTerminalShellNotInstalled)
	}
	command := exec.Command(path, "-l")
	command.Dir = workDir
	return command, shellName, nil
}

func getHostTerminalCapabilities() hostTerminalCapabilities {
	capabilities := hostTerminalCapabilities{DefaultShell: "sh", Shells: []string{}}
	defaultShell := filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
	for _, shellName := range []string{"bash", "zsh", "sh"} {
		if _, err := exec.LookPath(shellName); err != nil {
			continue
		}
		capabilities.Shells = append(capabilities.Shells, shellName)
		if shellName == defaultShell {
			capabilities.DefaultShell = shellName
		}
	}
	return capabilities
}

func stopHostTerminalProcess(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return buserr.New(constant.ErrHostTerminalProcessMissing)
	}
	if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return command.Process.Kill()
}
