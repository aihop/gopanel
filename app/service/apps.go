package service

import (
	"encoding/json"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/random"
	"github.com/gofiber/fiber/v3"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"regexp"
	"strings"
)

var composeShellConditionalExprPattern = regexp.MustCompile(`\$\{[A-Za-z_][A-Za-z0-9_]*:\+[\s\S]*?\}`)

type AppService struct{}
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
		appDTO := &response.AppItem{ID: ap.ID, Name: ap.Name, Key: ap.Key, ShortDescZh: ap.ShortDescZh, ShortDescEn: ap.ShortDescEn, Type: ap.Type, Icon: ap.Icon, Resource: ap.Resource, Limit: ap.Limit, GpuSupport: ap.GpuSupport}
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
	installs, _ := appInstallRepo.ListBy(appInstallRepo.WithAppId(app.ID))
	appDTO.Installed = len(installs) > 0
	appDTO.GpuSupport = app.GpuSupport
	return &appDTO, nil
}
func (a AppService) GetAppDetail(ctx fiber.Ctx, id uint, version string) (*response.AppDetailDTO, error) {
	res := &response.AppDetailDTO{}
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
	app, err := appRepo.GetFirst(appRepo.WithID(appDetail.AppId))
	if err != nil {
		return nil, err
	}
	if appDetail.DockerCompose == "" || app.Type == "runtime" || app.Type == "openclaw" {
		fileOp := files.NewFileOp()
		versionPath := filepath.Join(app.GetAppResourcePath(), appDetail.Version)
		if !fileOp.Stat(versionPath) || appDetail.Update {
			if err = downloadApp(app, appDetail, nil); err != nil && !fileOp.Stat(versionPath) {
				return nil, err
			}
		}
		paramsPath := filepath.Join(versionPath, "data.yml")
		if fileOp.Stat(paramsPath) {
			paramContent, err := fileOp.GetContent(paramsPath)
			if err == nil {
				paramMap := make(map[ // Default to getting the latest version if no version is provided
				string]interface{})
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
