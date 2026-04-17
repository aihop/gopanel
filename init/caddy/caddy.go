package caddy

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils"
	"github.com/aihop/gopanel/utils/files"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"

	_ "github.com/caddy-dns/cloudflare"
	_ "github.com/caddyserver/cache-handler"
	_ "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/fileserver"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	_ "github.com/caddyserver/caddy/v2/modules/logging"
	_ "github.com/darkweak/storages/otter/caddy"
	_ "github.com/darkweak/storages/redis/caddy"
)

type CaddyServer struct {
	Status        bool   // Caddy 服务状态
	CaddyFilePath string // Caddyfile 路径
}

var Server CaddyServer = CaddyServer{
	Status:        false,
	CaddyFilePath: "",
}

func Init() {
	path := CaddyFilePath()
	fileService := files.NewFileOp()
	if !fileService.Stat(path) { // 如果文件不存在，就创建一个文件
		fileService.CreateFileWithMode(path, 0644)
	}
	content, err := fileService.GetContent(path)
	if err != nil {
		return
	}
	err = StartCaddyServer(content)
	if err != nil {
		global.LOG.Error(err.Error())
		return
	}
	global.LOG.Info("Caddy start successfully")
}

func CaddyFilePath() string {
	if Server.CaddyFilePath == "" {
		Server.CaddyFilePath = utils.GetConfigPath(global.CONF.System.BaseDir, "Caddyfile")
	}
	return Server.CaddyFilePath
}

// 接收 CaddyFile 启动caddy服务
func StartCaddyServer(content []byte) error {
	jsonConfig, err := CaddyFileToJson(content)
	if err != nil && len(content) > 0 {
		return errors.New(err.Error())
	}
	if err := caddy.Load(jsonConfig, true); err != nil {
		Server.Status = false
		errInfo := err.Error()
		if strings.Contains(errInfo, "address already in use") || strings.Contains(errInfo, "bind: address already in use") {
			return errors.New("HTTP服务端口被占用，错误: " + errInfo)
		}
		if strings.Contains(strings.ToLower(errInfo), "permission denied") || strings.Contains(errInfo, "eacces") {
			return errors.New("HTTP服务，权限不足，无法绑定受限端口(80/443): " + errInfo)
		}
		return errors.New("HTTP服务启动失败: " + errInfo)
	}
	Server.Status = true
	return nil
}

func StopCaddyServer() error {
	if err := caddy.Stop(); err != nil {
		Server.Status = false
		return errors.New("停止HTTP服务失败: " + err.Error())
	}
	Server.Status = false
	return nil
}

func CaddyFileToJson(content []byte) ([]byte, error) {
	adapter := caddyconfig.GetAdapter("caddyfile")
	if adapter == nil {
		return nil, errors.New("未找到配置文件")
	}
	jsonConfig, warnings, err := adapter.Adapt([]byte(content), nil)
	if err != nil {
		return nil, err
	}
	for _, warning := range warnings {
		global.LOG.Warn(warning.String())
	}
	return jsonConfig, nil
}
