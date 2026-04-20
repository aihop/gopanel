package service

import (
	"os/exec"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

func composeAvailable() bool {
	if _, err := exec.LookPath("podman"); err == nil {
		return true
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return true
	}
	if _, err := exec.LookPath("podman-compose"); err == nil {
		return true
	}
	return false
}

func hasDockerSockPathSetting() bool {
	var settingItem model.Setting
	if err := global.DB.Where("key = ?", "DockerSockPath").First(&settingItem).Error; err != nil {
		return false
	}
	return strings.TrimSpace(settingItem.Value) != ""
}

func isPodmanInstalled() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}
