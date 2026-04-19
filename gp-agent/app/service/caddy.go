package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/aihop/gopanel/gp-agent/init/caddy"
	"github.com/caddyserver/caddy/v2/caddyconfig"
)

type CaddyStatus struct {
	Running   bool   `json:"running"`
	Caddyfile string `json:"caddyfile"`
}

type CaddyConfig struct {
	Caddyfile string `json:"caddyfile"`
}

func CaddyStatusJSON() (string, error) {
	out := CaddyStatus{
		Running:   caddy.Server.Status,
		Caddyfile: caddy.CaddyFilePath(),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CaddyGetConfigJSON() (string, error) {
	path := caddy.CaddyFilePath()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out := CaddyConfig{Caddyfile: string(b)}
	j, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func CaddyApply(ctx context.Context, params map[string]interface{}) (string, error) {
	content := ""
	if v, ok := params["caddyfile"]; ok && v != nil {
		if s, ok := v.(string); ok {
			content = s
		}
	}
	path := caddy.CaddyFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	prev, _ := os.ReadFile(path)
	backupDir := global.CONF.BackupDir
	_ = os.MkdirAll(backupDir, 0755)
	backupPath := filepath.Join(backupDir, "Caddyfile."+time.Now().Format("20060102-150405.000000000"))
	_ = os.WriteFile(backupPath, prev, 0644)

	if content == "" {
		_ = os.WriteFile(path, []byte(""), 0644)
		if err := caddy.StartCaddyServer([]byte("")); err != nil {
			return "", err
		}
		return "ok", nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	if err := caddy.StartCaddyServer([]byte(content)); err != nil {
		_ = os.WriteFile(path, prev, 0644)
		_ = caddy.StartCaddyServer(prev)
		return "", err
	}
	return "ok", nil
}

func CaddyStop(ctx context.Context) (string, error) {
	_ = ctx
	if !caddy.Server.Status {
		return "ok", nil
	}
	if err := caddy.StopCaddyServer(); err != nil {
		return "", err
	}
	return "ok", nil
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
