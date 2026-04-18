package docker

import (
	"os"
	"path/filepath"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/firewall"
)

var DaemonJsonPath string

func Init() {

	go func() {
		_ = docker.CreateDefaultDockerNetwork()

		if f, err := firewall.NewFirewallClient(); err == nil {
			if err = f.EnableForward(); err != nil {
				global.LOG.Errorf("init port forward failed, err: %v", err)
			}
		}
	}()
	// 优先使用环境变量覆盖
	if p := os.Getenv("DOCKER_DAEMON_JSON_PATH"); p != "" {
		DaemonJsonPath = p
		ensureDaemonJson()
		return
	}

	DaemonJsonPath = utils.GetConfigPath(constant.DaemonJsonPath, "daemon.json")

	ensureDaemonJson()
}

func ensureDaemonJson() {
	if DaemonJsonPath == "" {
		return
	}
	dir := filepath.Dir(DaemonJsonPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		global.LOG.Error("create daemon.json dir failed", "path", dir, "err", err.Error())
		return
	}
	if _, err := os.Stat(DaemonJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(DaemonJsonPath, []byte("{}\n"), 0644); err != nil {
			global.LOG.Error("create daemon.json failed", "path", DaemonJsonPath, "err", err.Error())
			return
		}
		global.LOG.Info("created default daemon.json", "path", DaemonJsonPath)
	}
}
