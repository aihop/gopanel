//go:build windows

package api

import (
	"os/exec"
	"strings"

	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

func buildHostTerminalCommand(shellName, workDir string) (*exec.Cmd, string, error) {
	shellName = strings.ToLower(strings.TrimSpace(shellName))
	if shellName == "" || shellName == "default" {
		shellName = "powershell"
	}
	var command *exec.Cmd
	switch shellName {
	case "powershell":
		path, err := exec.LookPath("powershell.exe")
		if err != nil {
			return nil, "", buserr.New(constant.ErrHostTerminalPowerShellMissing)
		}
		command = exec.Command(path, "-NoLogo")
	case "cmd":
		path, err := exec.LookPath("cmd.exe")
		if err != nil {
			return nil, "", buserr.New(constant.ErrHostTerminalCmdUnavailable)
		}
		command = exec.Command(path)
	default:
		return nil, "", buserr.New(constant.ErrHostTerminalShellUnsupported)
	}
	command.Dir = workDir
	return command, shellName, nil
}

func getHostTerminalCapabilities() hostTerminalCapabilities {
	capabilities := hostTerminalCapabilities{DefaultShell: "cmd", Shells: []string{"cmd"}}
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		capabilities.DefaultShell = "powershell"
		capabilities.Shells = append([]string{"powershell"}, capabilities.Shells...)
	}
	return capabilities
}

func stopHostTerminalProcess(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return buserr.New(constant.ErrHostTerminalProcessMissing)
	}
	return command.Process.Kill()
}
