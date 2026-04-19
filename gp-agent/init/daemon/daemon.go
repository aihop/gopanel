package daemon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/aihop/gopanel/gp-agent/init/daemon/supervisord"
	"go.uber.org/zap"
)

// 守护进程管理器，负责管理 supervisord
var Supervisor *supervisord.Supervisor

func Init() error {
	supervisord.ReapZombie()
	Supervisor = supervisord.NewSupervisor(GetConfigFilePath())
	added, changed, removed, err := Supervisor.Reload(true) // 加载配置并启动
	if err != nil {
		if global.LOG != nil {
			global.LOG.Error(fmt.Sprintf("重载 Daemon 配置失败: %v", err))
		}
		return err
	} else {
		if global.LOG != nil {
			global.LOG.Info("Daemon init successfully",
				zap.Strings("added", added),
				zap.Strings("changed", changed),
				zap.Strings("removed", removed),
			)
		}
	}
	return nil
}

func GetConfigFilePath() string {
	confDir := global.CONF.ConfDir
	if confDir == "" && global.CONF.BaseDir != "" {
		confDir = filepath.Join(global.CONF.BaseDir, "gp-agent", "conf")
	}
	if confDir == "" {
		confDir = "."
	}
	_ = os.MkdirAll(confDir, 0755)
	path := filepath.Join(confDir, "supervisord.ini")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			_ = os.WriteFile(path, []byte(""), 0644)
		}
	}
	return path
}
