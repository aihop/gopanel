package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

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
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/random"
)

var composeShellConditionalExprPattern = regexp.MustCompile(`\$\{[A-Za-z_][A-Za-z0-9_]*:\+[\s\S]*?\}`)

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
	total, apps, err := appRepo.Page(req.Page, req.Limit, opts...)
	if err != nil {
		return nil, err
	}
	var appDTOs []*response.AppItem
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
	// Check if the app is installed
	installs, _ := appInstallRepo.ListBy(appInstallRepo.WithAppId(app.ID))
	appDTO.Installed = len(installs) > 0
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

	var paramMap dto.AppForm
	if err := json.Unmarshal([]byte(appDetail.Params), &paramMap); err == nil {
		if len(paramMap.FormFields) > 0 {
			for i := range paramMap.FormFields {
				v := &paramMap.FormFields[i]
				switch v.Type {
				case "user":
					if str, ok := v.Default.(string); ok {
						v.Default = "gopanel_" + str
					}
				case "password":
					if _, ok := v.Default.(string); ok {
						v.Default = random.RandString(16)
					}
				}
			}
		}
		res.Params = paramMap
	}

	res.HostMode = strings.Contains(appDetail.DockerCompose, "network_mode: host")
	return res, nil
}

func (a AppService) Install(ctx context.Context, req request.AppInstallCreate) (appInstall *model.AppInstall, err error) {
	// 创建默认网络
	if err = docker.CreateDefaultDockerNetwork(); err != nil {
		err = errors.New(constant.ErrGoPanelNetworkFailed + err.Error())
		return
	}
	// 检查应用名称是否存在
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
	// 检查应用详情是否存在
	appDetailRepo := repo.NewAppDetail()
	appDetail, err = appDetailRepo.GetFirst(commonRepo.WithByID(req.AppDetailId))
	if err != nil {
		return
	}
	// 检查应用是否存在
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

	// 检查应用是否需要端口
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
		installErr := error(nil)
		defer func() {
			if installErr != nil {
				appInstall.Status = constant.UpErr
				appInstall.Message = installErr.Error()
				logger.Error("Installation failed: %s", installErr.Error())
				_ = appInstallRepo.Save(context.Background(), appInstall)
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
