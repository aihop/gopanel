//go:build darwin

package service

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/aihop/gopanel/utils/docker"
)

func RepairPodmanShortNameOnDarwin(ctx context.Context) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("unsupported platform")
	}
	if err := docker.PodmanEnsureReady(ctx); err != nil {
		return "", err
	}

	data, machine, err := podmanMachineReadFile(ctx, "/etc/containers/registries.conf")
	if err != nil {
		return "", err
	}

	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return "", err
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	searchList := anyToStringSlice(config["unqualified-search-registries"])
	foundDocker := false
	for _, s := range searchList {
		if s == "docker.io" {
			foundDocker = true
			break
		}
	}
	if foundDocker {
		return "已经配置了 docker.io 短名解析，无需修复", nil
	}

	searchList = append(searchList, "docker.io")
	config["unqualified-search-registries"] = searchList

	outData, err := toml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal registries.conf: %w", err)
	}
	writeMachine, err := podmanMachineWriteFile(ctx, "/etc/containers/registries.conf", outData)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(writeMachine) == "" {
		writeMachine = machine
	}
	verifyData, err := podmanMachineReadFileOn(ctx, writeMachine, "/etc/containers/registries.conf")
	if err != nil {
		return "", err
	}
	var verify map[string]interface{}
	if err := toml.Unmarshal(verifyData, &verify); err != nil {
		return "", err
	}
	verifySearchList := anyToStringSlice(verify["unqualified-search-registries"])
	found := false
	for _, s := range verifySearchList {
		if s == "docker.io" {
			found = true
			break
		}
	}
	if !found {
		raw := strings.TrimSpace(string(verifyData))
		if len(raw) > 300 {
			raw = raw[:300]
		}
		return "", fmt.Errorf("podman machine registries.conf 未生效（machine=%s，unqualified=%v，raw=%s）", writeMachine, verifySearchList, raw)
	}
	return "已成功添加 docker.io 短名解析到 podman machine 的 /etc/containers/registries.conf（machine=" + writeMachine + "）", nil
}
