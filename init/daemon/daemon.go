package daemon

import (
	"fmt"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/daemon/supervisord"
	"github.com/aihop/gopanel/utils"
	"github.com/aihop/gopanel/utils/files"
)

// 守护进程管理器，负责管理 supervisord
var Supervisor *supervisord.Supervisor

func Init() {
	supervisord.ReapZombie()
	Supervisor = supervisord.NewSupervisor(GetConfigFilePath())
	added, changed, removed, err := Supervisor.Reload(true) // 加载配置并启动
	if err != nil {
		global.LOG.Error(fmt.Sprintf("重载 Daemon 配置失败: %v", err))
	} else {
		global.LOG.Info("Daemon init successfully", "added", added, "changed", changed, "removed", removed)
	}
}

func GetConfigFilePath() string {
	// 创建初始化文件
	global.CONF.System.ConfigSupervisorFile = utils.GetConfigPath(global.CONF.System.BaseDir, "supervisord.ini")
	fileService := files.NewFileOp()
	isExist := fileService.Stat(global.CONF.System.ConfigSupervisorFile)
	if !isExist {
		err := fileService.CreateFileWithMode(global.CONF.System.ConfigSupervisorFile, 0644)
		if err != nil {
			global.LOG.Error(fmt.Sprintf("创建 Daemon 配置文件失败: %v", err))
			panic(err)
		}
	}
	return global.CONF.System.ConfigSupervisorFile
}
