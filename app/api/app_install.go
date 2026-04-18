package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
)

// @Tags App
// @Summary Page app installed
// @Accept json
// @Param request body request.AppInstalledSearch true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/installed/search [post]
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
	return c.JSON(e.Succ(dto.PageResult{
		Items: list,
		Total: total,
	}))
}

// @Tags App
// @Summary List app installed
// @Accept json
// @Success 200 {array} dto.AppInstallInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/installed/list [get]
func ListAppInstalled(c fiber.Ctx) error {
	list, err := service.NewAppInstall().GetInstallList()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}

func UninstallApp(c fiber.Ctx) error {
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

	res := response.DatabaseConn{
		Status:        install.Status,
		Username:      username,
		Password:      password,
		Privilege:     true,
		ContainerName: install.ContainerName,
		ServiceName:   install.ServiceName,
		SystemIP:      service.GetOutboundIP(),
		Port:          int64(install.HttpPort),
	}
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

	res := response.AppInstalledCheck{
		IsExist:       install.ID > 0,
		Name:          install.Name,
		App:           app.Key,
		Version:       install.Version,
		Status:        install.Status,
		CreatedAt:     install.CreatedAt,
		LastBackupAt:  "",
		AppInstallID:  install.ID,
		ContainerName: install.ContainerName,
		InstallPath:   install.GetPath(),
		HttpPort:      install.HttpPort,
		HttpsPort:     install.HttpsPort,
	}
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
			params = append(params, response.AppParam{
				Key:      k,
				Value:    v,
				Edit:     true,
				LabelZh:  k,
				LabelEn:  k,
				Type:     "text",
				Required: false,
			})
		}
	}
	return c.JSON(e.Succ(response.AppConfig{
		Params: params,
	}))
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
	res := response.IgnoredApp{
		Icon:     app.Icon,
		Name:     app.Name,
		Version:  detail.Version,
		DetailID: detail.ID,
	}
	return c.JSON(e.Succ(res))
}

func AppLocalList(c fiber.Ctx) error {
	list, err := service.NewLocalAppService().List()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}

func AppLocalGet(c fiber.Ctx) error {
	key := c.Params("key")
	res, err := service.NewLocalAppService().Get(key)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

func AppLocalInstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppLocalInstallCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewLocalAppService().Install(context.Background(), *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

// 返回容器安装的基础目录
func AppGetBaseDir(c fiber.Ctx) error {
	return c.JSON(e.Succ(global.CONF.System.BaseDir + "/docker/compose/"))
}

// @Tags App
// @Summary Install app
// @Accept json
// @Param request body request.AppInstallCreate true "request"
// @Success 200 {object} model.AppInstall
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/install [post]
func AppInstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstallCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewAppService().Install(context.Background(), *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}

// AppInstallLogsStream streams the installation logs for a specific app install name via SSE.
func AppInstallLogsStream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	name := c.Params("name")
	if name == "" {
		return c.SendString("event: error\ndata: invalid app install name\n\n")
	}

	logger := service.GetAppInstallLogger(name)
	ch := logger.Subscribe()

	c.Status(200)
	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer logger.Unsubscribe(ch)

		// 发送历史日志
		for _, logMsg := range logger.GetLogs() {
			fmt.Fprintf(w, "data: %s\n\n", logMsg)
			if err := w.Flush(); err != nil {
				return
			}
		}

		for {
			select {
			case logMsg, ok := <-ch:
				if !ok || logMsg == "EOF" || logMsg == "[\"EOF\"]" || strings.Contains(logMsg, "EOF") {
					fmt.Fprintf(w, "data: EOF\n\n")
					_ = w.Flush()
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", logMsg)
				if err := w.Flush(); err != nil {
					return
				}
			case <-time.After(1 * time.Second): // keep-alive
				fmt.Fprintf(w, "event: ping\ndata: ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}

func getInstalledByName(name string) (model.AppInstall, error) {
	installRepo := repo.NewIAppInstallRepo()
	commonRepo := repo.NewCommonRepo()
	return installRepo.GetFirst(commonRepo.WithByName(name))
}
