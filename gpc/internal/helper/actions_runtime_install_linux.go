//go:build linux

package helper

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func (s *Server) actionContainerRuntimeInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	runtimeKind, err := validateRuntimeInstallKind(params)
	if err != nil {
		return "", err
	}
	if _, err := exec.LookPath(runtimeKind); err == nil {
		return encodeRuntimeInstallResult(runtimeInstallResult{Runtime: runtimeKind, Message: runtimeKind + " already installed"})
	}

	var output strings.Builder
	if err := installLinuxRuntimePackage(ctx, &output, runtimeKind); err != nil {
		return strings.TrimSpace(output.String()), err
	}
	if _, err := exec.LookPath(runtimeKind); err != nil {
		return strings.TrimSpace(output.String()), fmt.Errorf("%s still not found after installation", runtimeKind)
	}

	needsAction := ""
	if runtimeKind == "docker" {
		if _, err := exec.LookPath("systemctl"); err == nil {
			if err := runInstallCommand(ctx, &output, nil, "systemctl", "enable", "--now", "docker.service"); err != nil {
				needsAction = "startDocker"
			}
		} else if _, err := exec.LookPath("rc-update"); err == nil {
			_ = runInstallCommand(ctx, &output, nil, "rc-update", "add", "docker", "default")
			if err := runInstallCommand(ctx, &output, nil, "service", "docker", "start"); err != nil {
				needsAction = "startDocker"
			}
		} else {
			needsAction = "startDocker"
		}
	} else {
		if err := s.prepareInstalledPodman(ctx, params, &output); err != nil {
			needsAction = "startPodmanSocket"
			output.WriteString(err.Error() + "\n")
		}
	}
	if err := s.prepareInstalledCompose(ctx, runtimeKind, &output); err != nil {
		output.WriteString(err.Error() + "\n")
		if needsAction == "" {
			needsAction = "composeMissing"
		}
	}

	return encodeRuntimeInstallResult(runtimeInstallResult{
		Runtime: runtimeKind, Message: runtimeKind + " installed", NeedsAction: needsAction, Output: strings.TrimSpace(output.String()),
	})
}

func installLinuxRuntimePackage(ctx context.Context, output *strings.Builder, runtimeKind string) error {
	type packageCommand struct {
		bin  string
		env  []string
		args []string
	}
	commands := []packageCommand{}
	switch {
	case commandExists("apt-get"):
		if err := runInstallCommand(ctx, output, []string{"DEBIAN_FRONTEND=noninteractive"}, "apt-get", "update"); err != nil {
			return err
		}
		packageName := map[string]string{"docker": "docker.io", "podman": "podman"}[runtimeKind]
		commands = append(commands, packageCommand{"apt-get", []string{"DEBIAN_FRONTEND=noninteractive"}, []string{"install", "-y", packageName}})
	case commandExists("dnf"):
		packages := map[string][]string{"docker": {"docker", "moby-engine", "docker-ce"}, "podman": {"podman"}}[runtimeKind]
		for _, packageName := range packages {
			commands = append(commands, packageCommand{"dnf", nil, []string{"install", "-y", packageName}})
		}
	case commandExists("yum"):
		packages := map[string][]string{"docker": {"docker", "moby-engine", "docker-ce"}, "podman": {"podman"}}[runtimeKind]
		for _, packageName := range packages {
			commands = append(commands, packageCommand{"yum", nil, []string{"install", "-y", packageName}})
		}
	case commandExists("apk"):
		commands = append(commands, packageCommand{"apk", nil, []string{"add", "--no-cache", runtimeKind}})
	case commandExists("pacman"):
		commands = append(commands, packageCommand{"pacman", nil, []string{"-Sy", "--noconfirm", runtimeKind}})
	case commandExists("zypper"):
		commands = append(commands, packageCommand{"zypper", nil, []string{"--non-interactive", "install", "-y", runtimeKind}})
	default:
		return errors.New("no supported package manager found (apt/dnf/yum/apk/pacman/zypper)")
	}

	var lastErr error
	for _, command := range commands {
		lastErr = runInstallCommand(ctx, output, command.env, append([]string{command.bin}, command.args...)...)
		if lastErr == nil || commandExists(runtimeKind) {
			return nil
		}
	}
	return lastErr
}

func (s *Server) prepareInstalledPodman(ctx context.Context, params map[string]interface{}, output *strings.Builder) error {
	rootless := getBool(params, "rootless")
	group := strings.TrimSpace(getString(params, "group"))
	if group == "" {
		group = "root"
	}
	repairParams := map[string]interface{}{"group": group, "rootless": rootless}
	if rootless {
		repairParams["uid"], _ = getInt(params, "uid")
		username := strings.TrimSpace(getString(params, "username"))
		repairParams["username"] = username
		if subIDOutput, err := s.actionRepairPodmanSubuid(ctx, map[string]interface{}{"username": username}); err != nil {
			output.WriteString(err.Error() + "\n")
		} else if strings.TrimSpace(subIDOutput) != "" {
			output.WriteString(subIDOutput + "\n")
		}
	}
	repairOutput, err := s.actionPodmanSocketRepair(ctx, repairParams)
	if strings.TrimSpace(repairOutput) != "" {
		output.WriteString(repairOutput + "\n")
	}
	return err
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (s *Server) prepareInstalledCompose(ctx context.Context, runtimeKind string, output *strings.Builder) error {
	if runtimeKind == "podman" {
		composeOutput, err := s.actionComposeInstall(ctx, nil)
		if strings.TrimSpace(composeOutput) != "" {
			output.WriteString(composeOutput + "\n")
		}
		return err
	}
	if dockerComposeAvailable(ctx) {
		return nil
	}

	var candidates [][]string
	switch {
	case commandExists("apt-get"):
		candidates = [][]string{{"apt-get", "install", "-y", "docker-compose-v2"}, {"apt-get", "install", "-y", "docker-compose-plugin"}, {"apt-get", "install", "-y", "docker-compose"}}
	case commandExists("dnf"):
		candidates = [][]string{{"dnf", "install", "-y", "docker-compose-plugin"}, {"dnf", "install", "-y", "docker-compose"}}
	case commandExists("yum"):
		candidates = [][]string{{"yum", "install", "-y", "docker-compose-plugin"}, {"yum", "install", "-y", "docker-compose"}}
	case commandExists("apk"):
		candidates = [][]string{{"apk", "add", "--no-cache", "docker-cli-compose"}, {"apk", "add", "--no-cache", "docker-compose"}}
	case commandExists("pacman"):
		candidates = [][]string{{"pacman", "-Sy", "--noconfirm", "docker-compose"}}
	case commandExists("zypper"):
		candidates = [][]string{{"zypper", "--non-interactive", "install", "-y", "docker-compose"}}
	}
	var lastErr error
	for _, candidate := range candidates {
		lastErr = runInstallCommand(ctx, output, nil, candidate...)
		if lastErr == nil || dockerComposeAvailable(ctx) {
			return nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no supported Docker Compose package found")
	}
	return lastErr
}

func dockerComposeAvailable(ctx context.Context) bool {
	return exec.CommandContext(ctx, "docker", "compose", "version").Run() == nil || commandExists("docker-compose")
}
