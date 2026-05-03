package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

func SearchAppInstalled(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstalledSearch](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	appInstallService := service.NewAppInstall()
	total, list, err := appInstallService.SearchForWebsite(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(dto.PageResult{Items: list, Total: total}))
}
func ListAppInstalled(c fiber.Ctx) error {
	list, err := service.NewAppInstall().GetInstallList()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}
func AppsUninstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[service.AppUninstall](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewAppInstall().Uninstall(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
func OperateAppInstalled(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstalledOperate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewAppInstall().Operate(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
func SyncAppInstalled(c fiber.Ctx) error {
	if err := service.NewAppInstall().SyncAll(); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
func LoadAppInstalledPort(c fiber.Ctx) error {
	var req struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := getInstalledByName(req.Name)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(int64(install.HttpPort)))
}
func GetAppInstalledConnInfo(c fiber.Ctx) error {
	var req struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := getInstalledByName(req.Name)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(install.Env), &envMap)
	password := ""
	if v, ok := envMap["PANEL_DB_ROOT_PASSWORD"].(string); ok {
		password = v
	} else if v, ok := envMap["PANEL_REDIS_ROOT_PASSWORD"].(string); ok {
		password = v
	} else if v, ok := envMap["PANEL_REDIS_PASSWORD"].(string); ok {
		password = v
	} else if v, ok := envMap["PASSWORD"].(string); ok {
		password = v
	}
	username := ""
	if v, ok := envMap["PANEL_DB_ROOT_USER"].(string); ok {
		username = v
	} else if v, ok := envMap["MYSQL_USER"].(string); ok {
		username = v
	}
	if username == "" {
		username = "root"
	}
	res := response.DatabaseConn{Status: install.Status, Username: username, Password: password, Privilege: true, ContainerName: install.ContainerName, ServiceName: install.ServiceName, SystemIP: service.GetOutboundIP(), Port: int64(install.HttpPort)}
	return c.JSON(e.Succ(res))
}
func CheckAppInstalled(c fiber.Ctx) error {
	var req struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(e.Fail(err))
	}
	appRepo := repo.NewIAppRepo()
	app, err := appRepo.GetFirst(appRepo.WithKey(req.Key))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	installRepo := repo.NewIAppInstallRepo()
	commonRepo := repo.NewCommonRepo()
	install, err := installRepo.GetFirst(installRepo.WithAppId(app.ID), commonRepo.WithByName(req.Name))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(e.Succ(response.AppInstalledCheck{IsExist: false}))
		}
		return c.JSON(e.Fail(err))
	}
	res := response.AppInstalledCheck{IsExist: install.ID > 0, Name: install.Name, App: app.Key, Version: install.Version, Status: install.Status, CreatedAt: install.CreatedAt, LastBackupAt: "", AppInstallID: install.ID, ContainerName: install.ContainerName, InstallPath: install.GetPath(), HttpPort: install.HttpPort, HttpsPort: install.HttpsPort}
	return c.JSON(e.Succ(res))
}
func AppInstalledDeleteCheck(c fiber.Ctx) error {
	return c.JSON(e.Succ([]map[string]string{}))
}
func GetAppInstalledParams(c fiber.Ctx) error {
	rawID := c.Params("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := repo.NewIAppInstallRepo().GetFirst(repo.NewCommonRepo().WithByID(uint(id)))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(install.Env), &envMap)
	var params []response.AppParam
	for k, v := range envMap {
		if strings.HasPrefix(k, "PANEL_") {
			params = append(params, response.AppParam{Key: k, Value: v, Edit: true, LabelZh: k, LabelEn: k, Type: "text", Required: false})
		}
	}
	return c.JSON(e.Succ(response.AppConfig{Params: params}))
}
func UpdateAppInstalledParams(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstalledUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := repo.NewIAppInstallRepo().GetFirst(repo.NewCommonRepo().WithByID(req.InstallId))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(install.Env), &envMap)
	for k, v := range req.Params {
		envMap[k] = v
	}
	envBytes, err := json.Marshal(envMap)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	install.Env = string(envBytes)
	if err := repo.NewIAppInstallRepo().Save(context.Background(), &install); err != nil {
		return c.JSON(e.Fail(err))
	}
	_ = service.NewAppInstall().Operate(request.AppInstalledOperate{InstallId: install.ID, Operate: constant.Restart, ForceDelete: true})
	return c.JSON(e.Succ())
}
func ChangeAppInstalledPort(c fiber.Ctx) error {
	var req struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		Port int64  `json:"port"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := getInstalledByName(req.Name)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(install.Env), &envMap)
	envMap["PANEL_APP_PORT_HTTP"] = req.Port
	envBytes, err := json.Marshal(envMap)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	install.Env = string(envBytes)
	if err := repo.NewIAppInstallRepo().Save(context.Background(), &install); err != nil {
		return c.JSON(e.Fail(err))
	}
	_ = service.NewAppInstall().Operate(request.AppInstalledOperate{InstallId: install.ID, Operate: constant.Restart, ForceDelete: true})
	return c.JSON(e.Succ())
}
func GetAppInstalledDefaultConfig(c fiber.Ctx) error {
	var req struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := getInstalledByName(req.Name)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(install.DockerCompose))
}
func UpdateAppInstalledVersions(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppUpdateVersion](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	install, err := repo.NewIAppInstallRepo().GetFirst(repo.NewCommonRepo().WithByID(req.AppInstallID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	install.Version = req.UpdateVersion
	if err := repo.NewIAppInstallRepo().Save(context.Background(), &install); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
func IgnoreAppInstalledUpgrade(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstalledIgnoreUpgrade](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	ignore := req.Operate == "ignore"
	if err := global.DB.Model(&model.AppDetail{}).Where("id = ?", req.DetailID).Updates(map[string]interface{}{"ignore_upgrade": ignore}).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
func GetIgnoredAppDetail(c fiber.Ctx) error {
	var detail model.AppDetail
	if err := global.DB.Where("ignore_upgrade = ?", true).First(&detail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(e.Succ(response.IgnoredApp{}))
		}
		return c.JSON(e.Fail(err))
	}
	commonRepo := repo.NewCommonRepo()
	app, _ := repo.NewIAppRepo().GetFirst(commonRepo.WithByID(detail.AppId))
	res := response.IgnoredApp{Icon: app.Icon, Name: app.Name, Version: detail.Version, DetailID: detail.ID}
	return c.JSON(e.Succ(res))
}
func getInstalledByName(name string) (model.AppInstall, error) {
	installRepo := repo.NewIAppInstallRepo()
	commonRepo := repo.NewCommonRepo()
	return installRepo.GetFirst(commonRepo.WithByName(name))
}
