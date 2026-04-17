package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/env"
	"github.com/aihop/gopanel/utils/files"
	"github.com/subosito/gotenv"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type LocalAppService struct{}

func NewLocalAppService() *LocalAppService {
	return &LocalAppService{}
}

func (s *LocalAppService) List() ([]dto.LocalAppItem, error) {
	appsDir, err := s.resolveAppsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, err
	}

	var items []dto.LocalAppItem
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		items = append(items, dto.LocalAppItem{
			Key:  name,
			Name: name,
		})
	}
	return items, nil
}

func (s *LocalAppService) Get(key string) (*dto.LocalAppDetail, error) {
	appsDir, err := s.resolveAppsDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(appsDir, key)
	if _, err := os.Stat(appDir); err != nil {
		return nil, err
	}

	composePath := filepath.Join(appDir, "docker-compose.yml")
	composeContent, _ := os.ReadFile(composePath)
	envPath := filepath.Join(appDir, ".env")
	envContent, _ := os.ReadFile(envPath)
	envMap, _ := gotenv.Unmarshal(string(envContent))

	var formFields []dto.LocalAppFormField
	dataPath := filepath.Join(appDir, "data.yml")
	dataContent, err := os.ReadFile(dataPath)
	if err == nil && len(dataContent) > 0 {
		var meta struct {
			AdditionalProperties struct {
				FormFields []dto.LocalAppFormField `yaml:"formFields"`
			} `yaml:"additionalProperties"`
		}
		if err := yaml.Unmarshal(dataContent, &meta); err == nil {
			formFields = meta.AdditionalProperties.FormFields
		}
	}

	return &dto.LocalAppDetail{
		Key:        key,
		Name:       key,
		FormFields: formFields,
		Compose:    string(composeContent),
		Env:        envMap,
	}, nil
}

func (s *LocalAppService) Install(ctx context.Context, req request.AppLocalInstallCreate) (*model.AppInstall, error) {
	if err := docker.CreateDefaultDockerNetwork(); err != nil {
		return nil, errors.New(constant.ErrGoPanelNetworkFailed + err.Error())
	}

	if list, _ := appInstallRepo.ListBy(commonRepo.WithByLowerName(req.Name)); len(list) > 0 {
		return nil, errors.New(constant.ErrAppNameExist)
	}

	appsDir, err := s.resolveAppsDir()
	if err != nil {
		return nil, err
	}
	templateDir := filepath.Join(appsDir, req.AppKey)
	if _, err := os.Stat(templateDir); err != nil {
		return nil, err
	}

	templateComposePath := filepath.Join(templateDir, "docker-compose.yml")
	composeRaw, err := os.ReadFile(templateComposePath)
	if err != nil {
		return nil, err
	}

	composeMap := make(map[string]interface{})
	if err := yaml.Unmarshal(composeRaw, &composeMap); err != nil {
		return nil, err
	}
	servicesVal, ok := composeMap["services"]
	if !ok || servicesVal == nil {
		return nil, errors.New(constant.ErrFileParse)
	}
	servicesMap, ok := servicesVal.(map[string]interface{})
	if !ok || len(servicesMap) == 0 {
		return nil, errors.New(constant.ErrFileParse)
	}

	serviceName := ""
	for k := range servicesMap {
		serviceName = k
		break
	}
	if len(servicesMap) == 1 && serviceName != req.Name {
		servicesMap[req.Name] = servicesMap[serviceName]
		delete(servicesMap, serviceName)
		serviceName = req.Name
	}
	composeMap["services"] = servicesMap

	containerName := constant.ContainerPrefix + req.AppKey + "-" + common.RandStr(4)
	if req.Advanced && req.ContainerName != "" {
		containerName = req.ContainerName
		appInstalls, _ := appInstallRepo.ListBy(appInstallRepo.WithContainerName(containerName))
		if len(appInstalls) > 0 {
			return nil, errors.New(constant.ErrContainerName)
		}
	}

	params := make(map[string]interface{}, len(req.Params)+4)
	for k, v := range req.Params {
		params[k] = v
	}
	params[constant.ContainerName] = containerName

	if err := addDockerComposeCommonParam(composeMap, serviceName, req.AppContainerConfig, params); err != nil {
		return nil, err
	}

	composeBytes, err := yaml.Marshal(composeMap)
	if err != nil {
		return nil, err
	}

	defaultEnv := make(map[string]interface{})
	if raw, err := os.ReadFile(filepath.Join(templateDir, ".env")); err == nil && len(raw) > 0 {
		m, err := gotenv.Unmarshal(string(raw))
		if err == nil {
			for k, v := range m {
				defaultEnv[k] = v
			}
		}
	}
	for k, v := range params {
		defaultEnv[k] = v
	}

	httpPort, _ := checkPort("PANEL_APP_PORT_HTTP", params)
	httpsPort, _ := checkPort("PANEL_APP_PORT_HTTPS", params)

	app, appDetail, err := s.ensureLocalAppAndDetail(ctx, req.AppKey, string(composeBytes))
	if err != nil {
		return nil, err
	}

	install := &model.AppInstall{
		Name:          req.Name,
		AppId:         app.ID,
		AppDetailId:   appDetail.ID,
		Version:       appDetail.Version,
		Status:        constant.Installing,
		ContainerName: containerName,
		ServiceName:   serviceName,
		HttpPort:      httpPort,
		HttpsPort:     httpsPort,
		App:           app,
		DockerCompose: string(composeBytes),
	}

	envJSON, err := json.Marshal(defaultEnv)
	if err != nil {
		return nil, err
	}
	install.Env = string(envJSON)

	baseDir := path.Join(global.CONF.System.BaseDir, "docker", "compose")
	fileOp := files.NewFileOp()
	if !fileOp.Stat(baseDir) {
		if err := fileOp.CreateDir(baseDir, 0755); err != nil {
			return nil, err
		}
	}
	if fileOp.Stat(path.Join(baseDir, req.Name)) {
		_ = fileOp.DeleteDir(path.Join(baseDir, req.Name))
	}
	if err := fileOp.CopyAndReName(templateDir, baseDir, req.Name, false); err != nil {
		return nil, err
	}

	envParams := make(map[string]string, len(defaultEnv))
	handleMap(defaultEnv, envParams)
	if err := env.Write(envParams, install.GetEnvPath()); err != nil {
		return nil, err
	}
	if err := fileOp.WriteFile(install.GetComposePath(), strings.NewReader(install.DockerCompose), 0755); err != nil {
		return nil, err
	}

	if err := appInstallRepo.Create(ctx, install); err != nil {
		return nil, err
	}
	_ = composeRepo.CreateRecord(&model.Compose{Name: install.Name, Path: install.GetComposePath()})

	go func() {
		defer func() {
			if err != nil {
				install.Status = constant.UpErr
				install.Message = err.Error()
				_ = appInstallRepo.Save(context.Background(), install)
			}
		}()
		if err = runScript(install, "init"); err != nil {
			return
		}
		out, upErr := compose.Up(install.GetComposePath())
		if upErr != nil {
			err = errors.New(out)
			return
		}
		install.Status = constant.Running
		_ = appInstallRepo.Save(context.Background(), install)
	}()

	return install, nil
}

func (s *LocalAppService) resolveAppsDir() (string, error) {
	if global.CONF.System.BaseDir != "" {
		p := filepath.Join(global.CONF.System.BaseDir, "apps")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	p := filepath.Join(wd, "apps")
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

func (s *LocalAppService) ensureLocalAppAndDetail(ctx context.Context, appKey string, dockerCompose string) (model.App, model.AppDetail, error) {
	app, err := appRepo.GetFirst(appRepo.WithKey(appKey))
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return model.App{}, model.AppDetail{}, err
		}
		newApp := &model.App{
			Name:      appKey,
			Key:       appKey,
			Type:      "local",
			Status:    constant.AppNormal,
			Website:   "",
			Github:    "",
			Document:  "",
			Recommend: 9999,
			Resource:  constant.AppResourceLocal,
			Limit:     0,
		}
		if err := appRepo.Create(ctx, newApp); err != nil {
			return model.App{}, model.AppDetail{}, err
		}
		app = *newApp
	}

	var detail model.AppDetail
	if err := global.DB.Where("app_id = ? AND version = ?", app.ID, "local").First(&detail).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return model.App{}, model.AppDetail{}, err
		}
		detail = model.AppDetail{
			AppId:         app.ID,
			Version:       "local",
			DockerCompose: dockerCompose,
			Status:        constant.StatusSuccStr,
		}
		if err := global.DB.Create(&detail).Error; err != nil {
			return model.App{}, model.AppDetail{}, err
		}
	} else {
		if detail.DockerCompose != dockerCompose {
			_ = global.DB.Model(&detail).Updates(map[string]interface{}{"docker_compose": dockerCompose}).Error
		}
	}

	return app, detail, nil
}
