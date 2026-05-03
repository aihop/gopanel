package service

import (
	"context"
	"encoding/json"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpagent"
	"os"
)

type caddyGetConfigResp struct {
	Caddyfile string `json:"caddyfile"`
}

func CaddyFilePath() string {
	return utils.GetConfigPath(global.CONF.System.BaseDir, "Caddyfile")
}
func CaddyContent() (string, error) {
	content, err := os.ReadFile(CaddyFilePath())
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(content), nil
}
func GetCaddyFile(ctx context.Context) (string, error) {
	resp, err := gpagent.Do(ctx, "CADDY_CONFIG", nil)
	if err != nil {
		return "", err
	}
	var out caddyGetConfigResp
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return "", err
	}
	return out.Caddyfile, nil
}
func CaddyApplyCaddyFile(ctx context.Context, content string) error {
	_, err := gpagent.Do(ctx, "CADDY_APPLY", map[string]interface{}{"caddyfile": content})
	return err
}
func CaddySaveContent(ctx context.Context, content string) error {
	fileUtil := files.NewFileOp()
	filePath := CaddyFilePath()
	backup, err := CaddyContent()
	if err != nil {
		return err
	}
	if err := fileUtil.SaveFileWithByte(filePath, []byte(content), 0644); err != nil {
		return err
	}
	if err := CaddyApplyCaddyFile(ctx, content); err != nil {
		if err != nil {
			_ = fileUtil.SaveFileWithByte(filePath, []byte(backup), 0644)
			return err
		}
	}
	return nil
}
