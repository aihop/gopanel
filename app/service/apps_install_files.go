package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/env"
	"github.com/aihop/gopanel/utils/files"
	httpUtil "github.com/aihop/gopanel/utils/http"
	"net/http"
	"path"
	"strings"
)

func RequestDownloadCallBack(downloadCallBackUrl string) {
	if downloadCallBackUrl == "" {
		return
	}
	_, _, _ = httpUtil.HandleGet(downloadCallBackUrl, http.MethodGet, constant.TimeOut5s)
}
func copyData(app model.App, appDetail model.AppDetail, appInstall *model.AppInstall, req request.AppInstallCreate) (err error) {
	fileOp := files.NewFileOp()
	appResourceDir := path.Join(constant.AppResourceDir, app.Resource)
	if app.Resource == constant.AppResourceRemote {
		err = downloadApp(app, appDetail, appInstall)
		if err != nil {
			return
		}
		go func() {
			RequestDownloadCallBack(appDetail.DownloadCallBackUrl)
		}()
	}
	appKey := app.Key
	installAppDir := path.Join(constant.AppInstallDir, app.Key)
	if app.Resource == constant.AppResourceLocal {
		appResourceDir = constant.LocalAppResourceDir
		appKey = strings.TrimPrefix(app.Key, "local")
		installAppDir = path.Join(constant.LocalAppInstallDir, appKey)
	}
	resourceDir := path.Join(appResourceDir, appKey, appDetail.Version)
	if !fileOp.Stat(installAppDir) {
		if err = fileOp.CreateDir(installAppDir, 0755); err != nil {
			return
		}
	}
	appDir := path.Join(installAppDir, req.Name)
	if fileOp.Stat(appDir) {
		if err = fileOp.DeleteDir(appDir); err != nil {
			return
		}
	}
	if err = fileOp.Copy(resourceDir, installAppDir); err != nil {
		return
	}
	versionDir := path.Join(installAppDir, appDetail.Version)
	if err = fileOp.Rename(versionDir, appDir); err != nil {
		return
	}
	envPath := path.Join(appDir, ".env")
	envParams := make(map[string]string, len(req.Params))
	handleMap(req.Params, envParams)
	normalizeRedisPasswordEnvAliases(envParams)
	if err = env.Write(envParams, envPath); err != nil {
		return
	}
	if err := fileOp.WriteFile(appInstall.GetComposePath(), strings.NewReader(appInstall.DockerCompose), 0755); err != nil {
		return err
	}
	ensureComposeRecord(appInstall.Name, appInstall.GetComposePath())
	return
}
func runScript(appInstall *model.AppInstall, operate string) error {
	workDir := appInstall.GetPath()
	scriptPath := ""
	switch operate {
	case "init":
		scriptPath = path.Join(workDir, "scripts", "init.sh")
	case "upgrade":
		scriptPath = path.Join(workDir, "scripts", "upgrade.sh")
	case "uninstall":
		scriptPath = path.Join(workDir, "scripts", "uninstall.sh")
	}
	if !files.NewFileOp().Stat(scriptPath) {
		return nil
	}
	out, err := cmd.ExecScript(scriptPath, workDir)
	if err != nil {
		if out != "" {
			errMsg := fmt.Sprintf("run script %s error %s", scriptPath, out)
			global.LOG.Error(errMsg)
			return errors.New(errMsg)
		}
		return err
	}
	return nil
}
func updateToolApp(installed *model.AppInstall) {
	tooKey, ok := dto.AppToolMap[installed.App.Key]
	if !ok {
		return
	}
	toolInstall, _ := getAppInstallByKey(tooKey)
	if toolInstall.ID == 0 {
		return
	}
	paramMap := make(map[string]string)
	_ = json.Unmarshal([]byte(installed.Param), &paramMap)
	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(toolInstall.Env), &envMap)
	if password, ok := paramMap["PANEL_DB_ROOT_PASSWORD"]; ok {
		envMap["PANEL_DB_ROOT_PASSWORD"] = password
	}
	if _, ok := envMap["PANEL_REDIS_HOST"]; ok {
		envMap["PANEL_REDIS_HOST"] = installed.ServiceName
	}
	if _, ok := envMap["PANEL_DB_HOST"]; ok {
		envMap["PANEL_DB_HOST"] = installed.ServiceName
	}
	envPath := path.Join(toolInstall.GetPath(), ".env")
	contentByte, err := json.Marshal(envMap)
	if err != nil {
		global.LOG.Errorf("update tool app [%s] error : %s", toolInstall.Name, err.Error())
		return
	}
	envFileMap := make(map[string]string)
	handleMap(envMap, envFileMap)
	if err = env.Write(envFileMap, envPath); err != nil {
		global.LOG.Errorf("update tool app [%s] error : %s", toolInstall.Name, err.Error())
		return
	}
	toolInstall.Env = string(contentByte)
	if err := appInstallRepo.Save(context.Background(), &toolInstall); err != nil {
		global.LOG.Errorf("update tool app [%s] error : %s", toolInstall.Name, err.Error())
		return
	}
	if out, err := compose.Down(toolInstall.GetComposePath()); err != nil {
		global.LOG.Errorf("update tool app [%s] error : %s", toolInstall.Name, out)
		return
	}
	if out, err := compose.Up(toolInstall.GetComposePath()); err != nil {
		global.LOG.Errorf("update tool app [%s] error : %s", toolInstall.Name, out)
		return
	}
}
