package app

import (
	"path/filepath"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
)

func Init() {
	constant.ResourceDir = filepath.Join(global.CONF.System.BaseDir, "resource")
	constant.AppResourceDir = filepath.Join(constant.ResourceDir, "apps")
	constant.AppInstallDir = filepath.Join(global.CONF.System.BaseDir, "apps")
	constant.RuntimeDir = filepath.Join(global.CONF.System.BaseDir, "runtime")

	constant.LocalAppResourceDir = filepath.Join(constant.AppResourceDir, "local")
	constant.LocalAppInstallDir = filepath.Join(constant.AppInstallDir, "local")
	constant.RemoteAppResourceDir = filepath.Join(constant.AppResourceDir, "remote")

	constant.SSLLogDir = filepath.Join(global.CONF.System.BaseDir, "log", "ssl")

	constant.McpDir = filepath.Join(global.CONF.System.BaseDir, "mcp")

	dirs := []string{constant.ResourceDir, constant.AppResourceDir, constant.AppInstallDir,
		global.CONF.System.Backup, constant.RuntimeDir, constant.LocalAppResourceDir, constant.RemoteAppResourceDir,
		constant.SSLLogDir, constant.McpDir}

	fileOp := files.NewFileOp()
	for _, dir := range dirs {
		createDir(fileOp, dir)
	}
}

func createDir(fileOp files.FileOp, dirPath string) {
	if !fileOp.Stat(dirPath) {
		_ = fileOp.CreateDir(dirPath, 0755)
	}
}
