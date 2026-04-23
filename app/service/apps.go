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
	"regexp"
	"runtime"
	"sort"
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
	"github.com/aihop/gopanel/utils/env"
	"github.com/aihop/gopanel/utils/files"
	httpUtil "github.com/aihop/gopanel/utils/http"
	"github.com/aihop/gopanel/utils/random"
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

func upApp(appInstall *model.AppInstall, pullImages bool, logger *AppInstallLogger) error {
	upProject := func(appInstall *model.AppInstall) (err error) {
		var (
			out    string
			errMsg string
		)
		envContent, err := files.NewFileOp().GetContent(appInstall.GetEnvPath())
		if err != nil {
			logger.Error("Failed to read .env file: %v", err)
			return err
		}
		if fixedEnv, changedEnv := qualifyEnvImageVars(string(envContent)); changedEnv {
			envContent = []byte(fixedEnv)
			_ = files.NewFileOp().WriteFile(appInstall.GetEnvPath(), strings.NewReader(fixedEnv), 0644)
			logger.Info("Qualified image variables in .env")
		}
		composeContent, err := files.NewFileOp().GetContent(appInstall.GetComposePath())
		if err != nil {
			logger.Error("Failed to read docker-compose.yml: %v", err)
			return err
		}
		if fixed, changed, ferr := qualifyComposeImagesYAML(string(composeContent)); ferr == nil && changed {
			composeContent = []byte(fixed)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed), 0644)
			logger.Info("Qualified image names in docker-compose.yml")
		} else if fixed2, changed2 := qualifyComposeImagesText(string(composeContent)); changed2 {
			composeContent = []byte(fixed2)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed2), 0644)
			logger.Info("Qualified image names in docker-compose.yml")
		}
		if validateErr := validateComposeEnvForPortsVolumes(string(composeContent), string(envContent)); validateErr != nil {
			logger.Error("Compose params invalid: %s", validateErr.Error())
			appInstall.Message = validateErr.Error()
			return validateErr
		}

		if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
			_ = docker.PodmanEnsureReady(context.Background())
		}
		if nerr := ensureExternalNetworks(string(composeContent)); nerr != nil {
			logger.Error("Ensure external networks failed: %s", nerr.Error())
			appInstall.Message = nerr.Error()
			return nerr
		}

		if pullImages {
			logger.Info("Executing compose pull...")
			pullCtx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
			defer cancel()
			if skip, _ := shouldSkipComposePull(pullCtx, string(composeContent), string(envContent)); skip {
				logger.Info("All images already exist locally, skip compose pull.")
				out = ""
				err = nil
			} else {
				out, err = compose.ExecStream(pullCtx, func(line string) {
					logger.Info("%s", line)
				}, "-f", appInstall.GetComposePath(), "pull")
			}
			msg := strings.ToLower(out)
			if err == nil {
				if strings.Contains(msg, "exit code:") && !strings.Contains(msg, "exit code: 0") {
					err = errors.New("compose pull exit code not zero")
				} else if strings.Contains(msg, "short-name") && strings.Contains(msg, "did not resolve") {
					err = errors.New("podman short-name not resolved")
				} else if strings.Contains(msg, "invalid status code") && strings.Contains(msg, "404") && strings.Contains(msg, "error fetching blob") {
					err = errors.New("compose pull registry 404")
				} else if strings.Contains(msg, "unable to copy from source") && strings.Contains(msg, "parsing image configuration") && strings.Contains(msg, ": eof") {
					err = errors.New("compose pull image configuration eof")
				} else if strings.Contains(msg, "unable to copy from source") && strings.Contains(msg, ": eof") {
					err = errors.New("compose pull stream eof")
				}
			}
			if err != nil {
				msg = strings.ToLower(out + " " + err.Error())
				if strings.Contains(msg, "no such host") {
					errMsg = i18n.GetMsgByKey("ErrNoSuchHost") + ":"
				}
				if strings.Contains(msg, "timeout") {
					errMsg = i18n.GetMsgByKey("ErrImagePullTimeOut") + ":"
				}
				if errMsg == "" {
					errMsg = i18n.GetMsgWithMap("ErrImagePull", map[string]interface{}{"err": err.Error()})
				}
				if strings.Contains(msg, "invalid status code") && strings.Contains(msg, "404") && strings.Contains(msg, "error fetching blob") {
					nomirrorPath := filepath.Join(appInstall.GetPath(), ".registries.nomirror.conf")
					nomirrorConf := `unqualified-search-registries = ["docker.io"]

[[registry]]
prefix = "docker.io"
location = "docker.io"
`
					_ = files.NewFileOp().WriteFile(nomirrorPath, strings.NewReader(nomirrorConf), 0644)
					logger.Info("Compose pull failed with registry 404, retry without mirrors...")
					out2, err2 := compose.ExecStreamWithEnv(pullCtx, func(line string) {
						logger.Info("%s", line)
					}, []string{"CONTAINERS_REGISTRIES_CONF=" + nomirrorPath}, "-f", appInstall.GetComposePath(), "pull")
					out = out + "\n" + out2
					if err2 == nil {
						msg2 := strings.ToLower(out2)
						if strings.Contains(msg2, "exit code:") && !strings.Contains(msg2, "exit code: 0") {
							err2 = errors.New("compose pull exit code not zero")
						} else if strings.Contains(msg2, "short-name") && strings.Contains(msg2, "did not resolve") {
							err2 = errors.New("podman short-name not resolved")
						} else if strings.Contains(msg2, "unable to copy from source") && strings.Contains(msg2, "parsing image configuration") && strings.Contains(msg2, ": eof") {
							err2 = errors.New("compose pull image configuration eof")
						} else if strings.Contains(msg2, "unable to copy from source") && strings.Contains(msg2, ": eof") {
							err2 = errors.New("compose pull stream eof")
						}
					}
					if err2 == nil {
						err = nil
					} else {
						err = err2
					}
				}
				if err != nil {
					appInstall.Message = errMsg + out
					logger.Error("Image pull failed: %s", appInstall.Message)
					return err
				}
			}
			logger.Info("Compose pull completed.")
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
	runErr := upProject(appInstall)
	if runErr == nil {
		// 先退出 Installing，避免安装后首次同步被短路跳过。
		appInstall.Status = constant.Running
		appInstall.Message = ""
	} else {
		appInstall.Status = constant.UpErr
	}
	exist, _ := appInstallRepo.GetFirst(commonRepo.WithByID(appInstall.ID))
	if exist.ID > 0 {
		if runErr == nil {
			if confirmErr := waitForInstalledContainers(appInstall, logger); confirmErr != nil {
				appInstall.Status = constant.UpErr
				appInstall.Message = confirmErr.Error()
				_ = appInstallRepo.Save(context.Background(), appInstall)
				return confirmErr
			}
		} else {
			containerNames, err := getContainerNames(*appInstall)
			if err != nil {
				return runErr
			}
			if len(containerNames) > 0 {
				appInstall.ContainerName = strings.Join(containerNames, ",")
			}
		}
		_ = appInstallRepo.Save(context.Background(), appInstall)
	}
	return runErr
}

func waitForInstalledContainers(appInstall *model.AppInstall, logger *AppInstallLogger) error {
	deadline := time.Now().Add(15 * time.Second)
	var lastErr error
	for attempt := 1; ; attempt++ {
		containerNames, err := getContainerNames(*appInstall)
		if err != nil {
			lastErr = err
		} else if len(containerNames) > 0 {
			appInstall.ContainerName = strings.Join(containerNames, ",")
		}

		appInstall.Status = constant.Running
		appInstall.Message = ""
		if err := syncAppInstallStatus(appInstall, true); err != nil {
			lastErr = err
		} else {
			stateDesc, descErr := describeInstallContainerState(appInstall)
			if descErr != nil {
				lastErr = descErr
			}
			diagDesc := ""
			if d, diagErr := describeInstallContainerDiagnostics(appInstall); diagErr == nil {
				diagDesc = strings.TrimSpace(d)
			}
			switch appInstall.Status {
			case constant.Running:
				return nil
			case constant.Error, constant.UnHealthy, constant.Stopped:
				if stateDesc != "" {
					if diagDesc != "" {
						lastErr = errors.New(stateDesc + "; " + diagDesc)
					} else {
						lastErr = errors.New(stateDesc)
					}
				} else if strings.TrimSpace(appInstall.Message) != "" {
					lastErr = errors.New(appInstall.Message)
				} else {
					lastErr = fmt.Errorf("container status is %s", appInstall.Status)
				}
			default:
				if stateDesc != "" || diagDesc != "" {
					if stateDesc != "" && diagDesc != "" {
						lastErr = errors.New(stateDesc + "; " + diagDesc)
					} else if stateDesc != "" {
						lastErr = errors.New(stateDesc)
					} else {
						lastErr = errors.New(diagDesc)
					}
				}
			}
		}

		if time.Now().After(deadline) {
			break
		}
		if logger != nil && (attempt == 1 || attempt%3 == 0) {
			logger.Info("Waiting for container(s) to enter running state...")
			if stateDesc, err := describeInstallContainerState(appInstall); err == nil && strings.TrimSpace(stateDesc) != "" {
				logger.Info("%s", stateDesc)
			}
			if diagDesc, err := describeInstallContainerDiagnostics(appInstall); err == nil && strings.TrimSpace(diagDesc) != "" {
				logger.Info("%s", diagDesc)
			}
		}
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		return lastErr
	}
	if strings.TrimSpace(appInstall.Message) != "" {
		return errors.New(appInstall.Message)
	}
	return errors.New("container did not enter running state after compose up")
}

var composeVarRe = regexp.MustCompile(`\\$\\{([^}]+)\\}`)

func ensureExternalNetworks(composeYml string) error {
	composeMap := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(composeYml), &composeMap); err != nil {
		return err
	}
	netsAny, ok := composeMap["networks"]
	if !ok || netsAny == nil {
		return nil
	}
	nets, ok := netsAny.(map[string]interface{})
	if !ok {
		return nil
	}
	for k, v := range nets {
		name := strings.TrimSpace(k)
		external := false
		if m, ok := v.(map[string]interface{}); ok {
			if ex, ok := m["external"]; ok {
				switch x := ex.(type) {
				case bool:
					external = x
				case map[string]interface{}:
					external = true
					if n, ok := x["name"]; ok {
						if s := strings.TrimSpace(fmt.Sprint(n)); s != "" {
							name = s
						}
					}
				default:
					if strings.EqualFold(strings.TrimSpace(fmt.Sprint(ex)), "true") {
						external = true
					}
				}
			}
			if external {
				if exName, ok := m["name"]; ok {
					if s := strings.TrimSpace(fmt.Sprint(exName)); s != "" {
						name = s
					}
				}
			}
		}
		if !external {
			continue
		}
		if err := docker.EnsureNetwork(name); err != nil {
			return fmt.Errorf("external network %s not available: %w", name, err)
		}
	}
	return nil
}

func validateComposeEnvForPortsVolumes(composeYml string, envText string) error {
	envMap := parseDotEnv(envText)
	required := extractRequiredVarsFromComposePortsVolumes(composeYml)
	if len(required) == 0 {
		return nil
	}
	var missing []string
	for _, k := range required {
		v := strings.TrimSpace(envMap[k])
		v = strings.Trim(v, `"`)
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("missing required compose variables (ports/volumes): %s", strings.Join(missing, ", "))
}

func parseDotEnv(envText string) map[string]string {
	out := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(envText, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		out[k] = v
	}
	return out
}

func extractRequiredVarsFromComposePortsVolumes(composeYml string) []string {
	lines := strings.Split(strings.ReplaceAll(composeYml, "\r\n", "\n"), "\n")
	var (
		inPorts   bool
		inVolumes bool
		portsInd  int
		volInd    int
	)
	requiredSet := make(map[string]struct{})

	for _, line := range lines {
		raw := line
		line = strings.TrimRight(line, " \t")
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		ind := len(raw) - len(strings.TrimLeft(raw, " \t"))

		if inPorts && ind <= portsInd {
			inPorts = false
		}
		if inVolumes && ind <= volInd {
			inVolumes = false
		}

		if strings.HasSuffix(trim, "ports:") {
			inPorts = true
			portsInd = ind
			continue
		}
		if strings.HasSuffix(trim, "volumes:") {
			inVolumes = true
			volInd = ind
			continue
		}
		if !(inPorts || inVolumes) {
			continue
		}

		for _, m := range composeVarRe.FindAllStringSubmatch(line, -1) {
			if len(m) < 2 {
				continue
			}
			name, required := parseComposeVarExpr(m[1])
			if !required || name == "" {
				continue
			}
			requiredSet[name] = struct{}{}
		}
	}

	var required []string
	for k := range requiredSet {
		required = append(required, k)
	}
	sort.Strings(required)
	return required
}

func parseComposeVarExpr(expr string) (string, bool) {
	s := strings.TrimSpace(expr)
	if s == "" {
		return "", false
	}
	seps := []string{":-", ":-", ":+", ":?", "-", "+", "?"}
	name := s
	op := ""
	for _, sep := range seps {
		if i := strings.Index(s, sep); i > 0 {
			name = strings.TrimSpace(s[:i])
			op = sep
			break
		}
	}
	if name == "" {
		return "", false
	}
	required := op == "" || op == ":?" || op == "?"
	if strings.Contains(name, " ") || strings.Contains(name, "\t") {
		return "", false
	}
	return name, required
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
