package caddy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"go.uber.org/zap"

	_ "github.com/caddy-dns/cloudflare"
	_ "github.com/caddyserver/cache-handler"
	_ "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/fileserver"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	_ "github.com/caddyserver/caddy/v2/modules/logging"
)

type CaddyServer struct {
	Status        bool
	CaddyFilePath string
}

var Server = CaddyServer{
	Status:        false,
	CaddyFilePath: "",
}

func Init() error {
	path := CaddyFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(path, []byte(""), 0644); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := StartCaddyServer(content); err != nil {
		if global.LOG != nil {
			global.LOG.Error("caddy start failed", zap.String("error", err.Error()))
		}
		return err
	}
	if global.LOG != nil {
		global.LOG.Info("caddy start successfully")
	}
	return nil
}

func CaddyFilePath() string {
	if Server.CaddyFilePath != "" {
		return Server.CaddyFilePath
	}
	if global.CONF.ConfDir != "" {
		Server.CaddyFilePath = filepath.Join(global.CONF.ConfDir, "Caddyfile")
		return Server.CaddyFilePath
	}
	if global.CONF.BaseDir != "" {
		Server.CaddyFilePath = filepath.Join(global.CONF.BaseDir, "gp-agent", "conf", "Caddyfile")
		return Server.CaddyFilePath
	}
	Server.CaddyFilePath = "Caddyfile"
	return Server.CaddyFilePath
}

func StartCaddyServer(content []byte) error {
	jsonConfig, err := CaddyFileToJSON(content)
	if err != nil && len(content) > 0 {
		return err
	}

	if err := caddy.Load(jsonConfig, true); err != nil {
		Server.Status = false
		errInfo := err.Error()
		if strings.Contains(errInfo, "address already in use") || strings.Contains(errInfo, "bind: address already in use") {
			return errors.New("HTTP服务端口被占用，错误: " + errInfo)
		}
		if strings.Contains(strings.ToLower(errInfo), "permission denied") || strings.Contains(strings.ToLower(errInfo), "eacces") {
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

func CaddyFileToJSON(content []byte) ([]byte, error) {
	adapter := caddyconfig.GetAdapter("caddyfile")
	if adapter == nil {
		return nil, errors.New("未找到配置文件适配器")
	}
	jsonConfig, warnings, err := adapter.Adapt([]byte(content), nil)
	if err != nil {
		return nil, err
	}
	if global.LOG != nil {
		for _, warning := range warnings {
			global.LOG.Warn("caddy adapt warning", zap.String("warning", warning.String()))
		}
	}
	return jsonConfig, nil
}
