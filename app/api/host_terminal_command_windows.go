//go:build windows

package api

import (
	"errors"
	"os/exec"
	"strings"
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
			return nil, "", errors.New("PowerShell 未安装")
		}
		command = exec.Command(path, "-NoLogo")
	case "cmd":
		path, err := exec.LookPath("cmd.exe")
		if err != nil {
			return nil, "", errors.New("cmd.exe 不可用")
		}
		command = exec.Command(path)
	default:
		return nil, "", errors.New("不支持的终端 Shell")
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
		return errors.New("终端进程不存在")
	}
	return command.Process.Kill()
}
