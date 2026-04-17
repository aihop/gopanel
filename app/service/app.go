package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gopkg.in/yaml.v3"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/i18n"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	composeV2 "github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/env"
	"github.com/aihop/gopanel/utils/files"
	httpUtil "github.com/aihop/gopanel/utils/http"
)

type AppService struct {
}

type IAppService interface {
	PageApp(ctx fiber.Ctx, req request.AppSearch) (interface{}, error)
	GetApp(ctx fiber.Ctx, key string) (*response.AppDTO, error)
	GetAppDetail(ctx fiber.Ctx, id uint, version string) (*response.AppDetailDTO, error)
	SyncAppList() error
}

func NewIAppService() IAppService {
	return &AppService{}
}
func NewAppService() *AppService {
	return &AppService{}
}

func (a AppService) PageApp(ctx fiber.Ctx, req request.AppSearch) (interface{}, error) {
	var opts []repo.DBOption
	opts = append(opts, appRepo.OrderByRecommend())
	if req.Name != "" {
		opts = append(opts, appRepo.WithLikeName(req.Name))
	}
	if req.Type != "" {
		opts = append(opts, appRepo.WithType(req.Type))
	}
	if req.Recommend {
		opts = append(opts, appRepo.GetRecommend())
	}
	if req.Resource != "" && req.Resource != "all" {
		opts = append(opts, appRepo.WithResource(req.Resource))
	}
	var res response.AppRes
	total, apps, err := appRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return nil, err
	}
	var appDTOs []*response.AppItem
	// lang := strings.ToLower(common.GetLang(ctx))
	for _, ap := range apps {
		appDTO := &response.AppItem{
			ID:          ap.ID,
			Name:        ap.Name,
			Key:         ap.Key,
			ShortDescZh: ap.ShortDescZh,
			ShortDescEn: ap.ShortDescEn,
			Type:        ap.Type,
			Icon:        ap.Icon,
			Resource:    ap.Resource,
			Limit:       ap.Limit,
			GpuSupport:  ap.GpuSupport,
		}
		appDTO.Description = ap.GetDescription(ctx)

		details, err := appDetailRepo.GetBy(appDetailRepo.WithAppId(ap.ID))
		if err == nil {
			var versionsRaw []string
			for _, detail := range details {
				versionsRaw = append(versionsRaw, detail.Version)
			}
			appDTO.Versions = common.GetSortedVersions(versionsRaw)
		}

		appDTOs = append(appDTOs, appDTO)
		// tags, err := getAppTags(ap.ID, lang)
		// if err != nil {
		// 	return nil, err
		// }
		// appDTO.Tags = tags
		installs, _ := appInstallRepo.ListBy(appInstallRepo.WithAppId(ap.ID))
		appDTO.Installed = len(installs) > 0
	}
	res.Items = appDTOs
	res.Total = total

	return res, nil
}

func (a AppService) GetApp(ctx fiber.Ctx, key string) (*response.AppDTO, error) {
	var appDTO response.AppDTO
	if key == "postgres" {
		key = "postgresql"
	}
	app, err := appRepo.GetFirst(appRepo.WithKey(key))
	if err != nil {
		return nil, err
	}
	appDTO.App = app
	appDTO.App.Description = app.GetDescription(ctx)
	details, err := appDetailRepo.GetBy(appDetailRepo.WithAppId(app.ID))
	if err != nil {
		return nil, err
	}
	var versionsRaw []string
	for _, detail := range details {
		versionsRaw = append(versionsRaw, detail.Version)
	}
	appDTO.Versions = common.GetSortedVersions(versionsRaw)
	// tags, err := getAppTags(app.ID, strings.ToLower(common.GetLang(ctx)))
	// if err != nil {
	// 	return nil, err
	// }
	appDTO.GpuSupport = app.GpuSupport
	// appDTO.Tags = tags
	return &appDTO, nil
}

func (a AppService) GetAppDetail(ctx fiber.Ctx, id uint, version string) (*response.AppDetailDTO, error) {
	res := &response.AppDetailDTO{}

	// Default to getting the latest version if no version is provided
	var appDetail model.AppDetail
	var err error

	if version != "" {
		err = global.DB.Where("app_id = ? AND version = ?", id, version).First(&appDetail).Error
	} else {
		err = global.DB.Where("app_id = ?", id).Order("id DESC").First(&appDetail).Error
	}

	if err != nil {
		return nil, err
	}

	res.AppDetail = appDetail

	// Get the app to check its type or to download it
	app, err := appRepo.GetFirst(appRepo.WithID(appDetail.AppId))
	if err != nil {
		return nil, err
	}

	// Always pull the remote package if docker-compose is missing or if it's a runtime/AI app
	if appDetail.DockerCompose == "" || app.Type == "runtime" || app.Type == "openclaw" {
		fileOp := files.NewFileOp()
		versionPath := filepath.Join(app.GetAppResourcePath(), appDetail.Version)

		if !fileOp.Stat(versionPath) || appDetail.Update {
			if err = downloadApp(app, appDetail, nil); err != nil && !fileOp.Stat(versionPath) {
				return nil, err
			}
		}

		// Read data.yml for params
		paramsPath := filepath.Join(versionPath, "data.yml")
		if fileOp.Stat(paramsPath) {
			paramContent, err := fileOp.GetContent(paramsPath)
			if err == nil {
				paramMap := make(map[string]interface{})
				if err = yaml.Unmarshal(paramContent, &paramMap); err == nil {
					if additionalProps, ok := paramMap["additionalProperties"]; ok {
						if propsBytes, err := json.Marshal(additionalProps); err == nil {
							appDetail.Params = string(propsBytes)
							res.AppDetail.Params = appDetail.Params
						}
					}
				}
			}
		}

		// Read docker-compose.yml
		composePath := filepath.Join(versionPath, "docker-compose.yml")
		if fileOp.Stat(composePath) {
			composeContent, err := fileOp.GetContent(composePath)
			if err == nil {
				appDetail.DockerCompose = string(composeContent)
				res.DockerCompose = appDetail.DockerCompose
				res.AppDetail.DockerCompose = appDetail.DockerCompose
			}
		}

		global.DB.Save(&appDetail)
	}

	paramMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(appDetail.Params), &paramMap); err == nil {
		res.Params = paramMap
	}

	res.HostMode = strings.Contains(appDetail.DockerCompose, "network_mode: host")
	return res, nil
}

func getAppFromRepo(downloadPath string) error {
	downloadUrl := downloadPath
	global.LOG.Infof("[AppStore] download file from %s", downloadUrl)
	fileOp := files.NewFileOp()
	packagePath := filepath.Join(constant.ResourceDir, filepath.Base(downloadUrl))
	if err := fileOp.DownloadFileWithProxy(downloadUrl, packagePath); err != nil {
		return err
	}
	if err := fileOp.Decompress(packagePath, constant.ResourceDir, files.SdkZip, ""); err != nil {
		return err
	}
	defer func() {
		_ = fileOp.DeleteFile(packagePath)
	}()
	return nil
}

func (a AppService) SyncAppList() error {
	global.LOG.Infof("[AppStore] Start syncing remote apps...")
	list, err := getAppList()
	if err != nil {
		global.LOG.Errorf("[AppStore] Failed to get app list: %v", err)
		return err
	}

	for _, appDef := range list.Apps {
		appProperty := appDef.AppProperty

		app := model.App{
			Name:               strings.ReplaceAll(appProperty.Name, "1Panel", "GoPanel"),
			Key:                appProperty.Key,
			ShortDescZh:        strings.ReplaceAll(strings.ReplaceAll(appProperty.ShortDescZh, "1Panel", "GoPanel"), "1panel", "gopanel"),
			ShortDescEn:        strings.ReplaceAll(strings.ReplaceAll(appProperty.ShortDescEn, "1Panel", "GoPanel"), "1panel", "gopanel"),
			Description:        "",
			Icon:               appDef.Icon,
			Type:               appProperty.Type,
			Status:             "published",
			Required:           strings.Join(appProperty.Required, ","),
			GpuSupport:         appProperty.GpuSupport,
			CrossVersionUpdate: appProperty.CrossVersionUpdate,
			Limit:              appProperty.Limit,
			Website:            appProperty.Website,
			Github:             appProperty.Github,
			Document:           appProperty.Document,
			Recommend:          appProperty.Recommend,
			Resource:           constant.AppResourceRemote,
			ReadMe:             strings.ReplaceAll(strings.ReplaceAll(appDef.ReadMe, "1Panel", "GoPanel"), "1panel", "gopanel"),
			LastModified:       appDef.LastModified,
		}
		descBytes, _ := json.Marshal(appProperty.Description)
		descStr := strings.ReplaceAll(string(descBytes), "1Panel", "GoPanel")
		descStr = strings.ReplaceAll(descStr, "1panel", "gopanel")
		app.Description = descStr

		// Check if exists
		var existApp model.App
		if err := global.DB.Where("key = ?", app.Key).First(&existApp).Error; err == nil && existApp.ID > 0 {
			app.ID = existApp.ID
		}
		global.DB.Save(&app)

		for _, v := range appDef.Versions {
			detail := model.AppDetail{
				AppId:               app.ID,
				Version:             v.Name,
				DownloadUrl:         v.DownloadUrl,
				DownloadCallBackUrl: v.DownloadCallBackUrl,
			}
			formBytes, _ := json.Marshal(v.AppForm)
			detail.Params = string(formBytes)

			// Replace any 1panel specific identifiers in AppForm JSON if needed
			detail.Params = strings.ReplaceAll(detail.Params, "1panel-network", "gopanel-network")
			detail.Params = strings.ReplaceAll(detail.Params, "/opt/1panel", global.CONF.System.BaseDir)
			detail.Params = strings.ReplaceAll(detail.Params, "1panel", "gopanel")
			detail.Params = strings.ReplaceAll(detail.Params, "1Panel", "GoPanel")

			var existDetail model.AppDetail
			if err := global.DB.Where("app_id = ? AND version = ?", app.ID, v.Name).First(&existDetail).Error; err == nil && existDetail.ID > 0 {
				detail.ID = existDetail.ID
			}
			global.DB.Save(&detail)
		}
	}
	global.LOG.Infof("[AppStore] App sync completed.")
	return nil
}
func getAppList() (*dto.AppList, error) {
	list := &dto.AppList{}

	repoUrl := global.CONF.System.AppRepo
	if repoUrl == "" {
		repoUrl = "https://apps-assets.fit2cloud.com"
	}
	mode := global.CONF.System.Mode
	if mode == "" {
		mode = "dev"
	}
	downloadUrl := fmt.Sprintf("%s/%s/1panel.json.zip", repoUrl, mode)

	if err := getAppFromRepo(downloadUrl); err != nil {
		return nil, err
	}
	listFile := filepath.Join(constant.ResourceDir, "1panel.json")
	content, err := os.ReadFile(listFile)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(content, list); err != nil {
		return nil, err
	}
	return list, nil
}

var InitTypes = map[string]struct{}{
	"runtime": {},
	"php":     {},
	"node":    {},
}

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

	// 从 req.Params 中提取端口信息
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

	appInstall = &model.AppInstall{
		Name:        req.Name,
		AppId:       appDetail.AppId,
		AppDetailId: appDetail.ID,
		Version:     appDetail.Version,
		Status:      constant.Installing,
		HttpPort:    httpPort,
		HttpsPort:   httpsPort,
		App:         app,
	}
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
	containerName := constant.ContainerPrefix + app.Key + "-" + common.RandStr(4)
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
		// 这里处理安装失败的清理逻辑
		if err != nil && appInstall.ID > 0 {
			_ = appInstallRepo.DeleteBy(commonRepo.WithByID(appInstall.ID))
			files.NewFileOp().DeleteDir(appInstall.GetPath())
		}
	}()

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
		defer func() {
			if err != nil {
				appInstall.Status = constant.UpErr
				appInstall.Message = err.Error()
				logger.Error("Installation failed: %s", err.Error())
				_ = appInstallRepo.Save(context.Background(), appInstall)
			} else {
				logger.Info("Installation completed successfully")
			}
			logger.Info("EOF")
		}()
		logger.Info("Copying app data...")
		if err = copyData(app, appDetail, appInstall, req); err != nil {
			logger.Error("Copy data failed: %s", err.Error())
			return
		}
		logger.Info("Running init scripts...")
		if err = runScript(appInstall, "init"); err != nil {
			logger.Error("Init script failed: %s", err.Error())
			return
		}
		logger.Info("Starting container(s)...")
		upApp(appInstall, req.PullImage, logger)
	}()
	go updateToolApp(appInstall)
	return
}
func upApp(appInstall *model.AppInstall, pullImages bool, logger *AppInstallLogger) {
	upProject := func(appInstall *model.AppInstall) (err error) {
		var (
			out    string
			errMsg string
		)
		if pullImages && appInstall.App.Type != "php" {
			projectName := strings.ToLower(appInstall.Name)
			envByte, err := files.NewFileOp().GetContent(appInstall.GetEnvPath())
			if err != nil {
				logger.Error("Failed to read .env file: %v", err)
				return err
			}
			images, err := composeV2.GetDockerComposeImages(projectName, envByte, []byte(appInstall.DockerCompose))
			if err != nil {
				logger.Error("Failed to get images from docker-compose: %v", err)
				return err
			}
			for _, image := range images {
				logger.Info("Pulling image: %s", image)
				if out, err = cmd.ExecWithTimeOut("docker pull "+image, 60*time.Minute); err != nil {
					if out != "" {
						if strings.Contains(out, "no such host") {
							errMsg = i18n.GetMsgByKey("ErrNoSuchHost") + ":"
						}
						if strings.Contains(out, "timeout") {
							errMsg = i18n.GetMsgByKey("ErrImagePullTimeOut") + ":"
						}
					} else {
						if err.Error() == buserr.New(constant.ErrCmdTimeout).Error() {
							errMsg = i18n.GetMsgByKey("ErrImagePullTimeOut")
						} else {
							errMsg = i18n.GetMsgWithMap("ErrImagePull", map[string]interface{}{"err": err.Error()})
						}
					}
					appInstall.Message = errMsg + out
					logger.Error("Image pull failed: %s", appInstall.Message)
					return err
				}
				logger.Info("Image %s pulled successfully.", image)
			}
		}

		logger.Info("Executing docker-compose up -d...")
		out, err = compose.Up(appInstall.GetComposePath())
		if err != nil {
			if out != "" {
				appInstall.Message = errMsg + out
			}
			logger.Error("docker-compose up failed: %v, out: %s", err, out)
			return err
		}
		logger.Info("Container(s) started successfully. Output: %s", out)
		return
	}
	if err := upProject(appInstall); err != nil {
		appInstall.Status = constant.UpErr
	} else {
		appInstall.Status = constant.Running
	}
	exist, _ := appInstallRepo.GetFirst(commonRepo.WithByID(appInstall.ID))
	if exist.ID > 0 {
		containerNames, err := getContainerNames(*appInstall)
		if err != nil {
			return
		}
		if len(containerNames) > 0 {
			appInstall.ContainerName = strings.Join(containerNames, ",")
		}
		_ = appInstallRepo.Save(context.Background(), appInstall)
	}
}
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
	if err = env.Write(envParams, envPath); err != nil {
		return
	}
	if err := fileOp.WriteFile(appInstall.GetComposePath(), strings.NewReader(appInstall.DockerCompose), 0755); err != nil {
		return err
	}
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

func addDockerComposeCommonParam(composeMap map[string]interface{}, serviceName string, req request.AppContainerConfig, params map[string]interface{}) error {
	services, serviceValid := composeMap["services"].(map[string]interface{})
	if !serviceValid {
		return buserr.New(constant.ErrFileParse)
	}
	service, serviceExist := services[serviceName]
	if !serviceExist {
		return buserr.New(constant.ErrFileParse)
	}
	serviceValue := service.(map[string]interface{})

	deploy := map[string]interface{}{}
	if de, ok := serviceValue["deploy"]; ok {
		deploy = de.(map[string]interface{})
	}
	resource := map[string]interface{}{}
	if res, ok := deploy["resources"]; ok {
		resource = res.(map[string]interface{})
	}
	resource["limits"] = map[string]interface{}{
		"cpus":   "${CPUS}",
		"memory": "${MEMORY_LIMIT}",
	}
	deploy["resources"] = resource
	serviceValue["deploy"] = deploy

	if req.GpuConfig {
		resource["reservations"] = map[string]interface{}{
			"devices": []map[string]interface{}{
				{
					"driver":       "nvidia",
					"count":        "all",
					"capabilities": []string{"gpu"},
				},
			},
		}
	} else {
		delete(resource, "reservations")
	}

	ports, ok := serviceValue["ports"].([]interface{})
	if ok {
		for i, port := range ports {
			portStr, portOK := port.(string)
			if !portOK {
				continue
			}
			portArray := strings.Split(portStr, ":")
			if len(portArray) == 2 {
				portArray = append([]string{"${HOST_IP}"}, portArray...)
			}
			ports[i] = strings.Join(portArray, ":")
		}
		serviceValue["ports"] = ports
	}

	params[constant.CPUS] = "0"
	params[constant.MemoryLimit] = "0"
	if req.Advanced {
		if req.CpuQuota > 0 {
			params[constant.CPUS] = req.CpuQuota
		}
		if req.MemoryLimit > 0 {
			params[constant.MemoryLimit] = strconv.FormatFloat(req.MemoryLimit, 'f', -1, 32) + req.MemoryUnit
		}
	}
	_, portExist := serviceValue["ports"].([]interface{})
	if portExist {
		allowHost := "127.0.0.1"
		if req.Advanced && req.AllowPort {
			allowHost = ""
		}
		params[constant.HostIP] = allowHost
	}
	services[serviceName] = serviceValue
	return nil
}
