package constant

import (
	"path/filepath"

	"github.com/aihop/gopanel/global"
)

var (
	ResourceDir          = filepath.Join(global.CONF.System.BaseDir, "resource")
	AppResourceDir       = filepath.Join(ResourceDir, "apps")
	AppInstallDir        = filepath.Join(global.CONF.System.BaseDir, "apps")
	LocalAppResourceDir  = filepath.Join(AppResourceDir, "local")
	LocalAppInstallDir   = filepath.Join(AppInstallDir, "local")
	RemoteAppResourceDir = filepath.Join(AppResourceDir, "remote")
	RuntimeDir           = filepath.Join(global.CONF.System.BaseDir, "runtime")
	RecycleBinDir        = "/.gopanel_clash"
	SSLLogDir            = filepath.Join(global.CONF.System.BaseDir, "log", "ssl")
	McpDir               = filepath.Join(global.CONF.System.BaseDir, "mcp")
)
