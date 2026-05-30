package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/random"
	"gopkg.in/yaml.v3"
)

func (a AppService) Install(ctx context.Context, req request.AppInstallCreate) (appInstall *model.AppInstall, err error) {
	if err = docker.CreateDefaultDockerNetwork(); err != nil {
		err = errors.New(constant.ErrGoPanelNetworkFailed + err.Error())
		return
	}
	if list, _ := appInstallRepo.ListBy(commonRepo.WithByLowerName(req.Name)); len(list) > 0 {
		err = errors.New(constant.ErrAppNameExist)
		return
	}
	var (
		httpPort  int
		httpsPort int
		appDetail model.AppDetail
		app       model.App
	)
	appDetailRepo := repo.NewAppDetail()
	appDetail, err = appDetailRepo.GetFirst(commonRepo.WithByID(req.AppDetailId))
	if err != nil {
		return
	}
	app, err = appRepo.GetFirst(commonRepo.WithByID(appDetail.AppId))
	if err != nil {
		return
	}
	for key := range req.Params {
		if !strings.Contains(key, "PANEL_APP_PORT") {
			continue
		}
		var port int
		if port, err = checkPort(key, req.Params); err == nil {
			if key == "PANEL_APP_PORT_HTTP" {
				httpPort = port
			}
			if key == "PANEL_APP_PORT_HTTPS" {
				httpsPort = port
			}
		} else {
			return
		}
	}
	if err = checkRequiredAndLimit(app); err != nil {
		return
	}
	appInstall = &model.AppInstall{Name: req.Name, AppId: appDetail.AppId, AppDetailId: appDetail.ID, Version: appDetail.Version, Status: constant.Installing, HttpPort: httpPort, HttpsPort: httpsPort, App: app}
	composeMap := make(map[string]interface{})
	if req.EditCompose {
		if err = yaml.Unmarshal([]byte(req.DockerCompose), &composeMap); err != nil {
			return
		}
	} else {
		if err = yaml.Unmarshal([]byte(appDetail.DockerCompose), &composeMap); err != nil {
			return
		}
	}
	value, ok := composeMap["services"]
	if !ok || value == nil {
		err = buserr.New(constant.ErrFileParse)
		return
	}
	servicesMap := value.(map[string]interface{})
	containerName := req.Name
	if containerName == "" {
		containerName = app.Key + "-" + random.RandString(4)
	}
	if req.Advanced && req.ContainerName != "" {
		containerName = req.ContainerName
		appInstalls, _ := appInstallRepo.ListBy(appInstallRepo.WithContainerName(containerName))
		if len(appInstalls) > 0 {
			err = buserr.New(constant.ErrContainerName)
			return
		}
		containerExist := false
		containerExist, err = checkContainerNameIsExist(req.ContainerName, appInstall.GetPath())
		if err != nil {
			return
		}
		if containerExist {
			err = buserr.New(constant.ErrContainerName)
			return
		}
	}
	req.Params[constant.ContainerName] = containerName
	appInstall.ContainerName = containerName
	index := 0
	serviceName := ""
	for k := range servicesMap {
		serviceName = k
		if index > 0 {
			continue
		}
		index++
	}
	if app.Limit == 0 && appInstall.Name != serviceName && len(servicesMap) == 1 {
		servicesMap[appInstall.Name] = servicesMap[serviceName]
		delete(servicesMap, serviceName)
		serviceName = appInstall.Name
	}
	appInstall.ServiceName = serviceName
	if err = addDockerComposeCommonParam(composeMap, appInstall.ServiceName, req.AppContainerConfig, req.Params); err != nil {
		return
	}
	qualifyComposeImagesInMap(composeMap)
	var (
		composeByte []byte
		paramByte   []byte
	)
	composeByte, err = yaml.Marshal(composeMap)
	if err != nil {
		return
	}
	appInstall.DockerCompose = string(composeByte)
	defer func() {
		if err != nil && appInstall.ID > 0 {
			_ = appInstallRepo.DeleteBy(commonRepo.WithByID(appInstall.ID))
			files.NewFileOp().DeleteDir(appInstall.GetPath())
		}
	}()
	normalizeRedisPasswordParamAliases(req.Params)
	paramByte, err = json.Marshal(req.Params)
	if err != nil {
		return
	}
	appInstall.Env = string(paramByte)
	if err = appInstallRepo.Create(ctx, appInstall); err != nil {
		return
	}
	if err = createLink(ctx, app, appInstall, req.Params); err != nil {
		return
	}
	logger := GetAppInstallLogger(appInstall.Name)
	logger.Info("Starting installation for %s (App: %s, Version: %s)", appInstall.Name, app.Key, appInstall.Version)
	go func() {
		installErr := error(nil)
		defer func() {
			if installErr != nil {
				logger.Error("Installation failed: %s", installErr.Error())
				// 清理 DB 记录和文件，避免列表中残留失败项
				_ = appInstallRepo.DeleteBy(commonRepo.WithByID(appInstall.ID))
				files.NewFileOp().DeleteDir(appInstall.GetPath())
			} else {
				if appInstall.Status == constant.Running {
					logger.Info("Installation completed successfully")
				} else {
					finalMsg := strings.TrimSpace(appInstall.Message)
					if finalMsg == "" {
						finalMsg = fmt.Sprintf("installation finished but final status is %s", appInstall.Status)
					}
					logger.Error("Installation did not enter running state: %s", finalMsg)
				}
			}
			logger.Info("EOF")
		}()
		logger.Info("Copying app data...")
		if installErr = copyData(app, appDetail, appInstall, req); installErr != nil {
			logger.Error("Copy data failed: %s", installErr.Error())
			return
		}
		logger.Info("Running init scripts...")
		if installErr = runScript(appInstall, "init"); installErr != nil {
			logger.Error("Init script failed: %s", installErr.Error())
			return
		}
		logger.Info("Starting container(s)...")
		installErr = upApp(appInstall, req.PullImage, logger)
	}()
	go updateToolApp(appInstall)
	return
}
