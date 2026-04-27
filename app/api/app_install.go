package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
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
	"github.com/aihop/gopanel/utils/docker"
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
// @Router /apps/installed/list [post]
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
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /apps/install [post]
func AppsInstall(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.AppInstallCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := service.NewAppService().Install(context.Background(), *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"installId": res.ID, "name": res.Name}))
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

	active := service.IsAppInstallLoggerActive(name)
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
		if !active {
			fmt.Fprintf(w, "data: EOF\n\n")
			_ = w.Flush()
			return
		}

		for {
			select {
			case logMsg, ok := <-ch:
				trimmed := strings.TrimSpace(logMsg)
				if !ok || trimmed == "EOF" || trimmed == "[\"EOF\"]" || strings.HasSuffix(trimmed, " INFO: EOF") {
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

func AppInstalledRuntimeLogsStream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return c.SendString("event: error\ndata: invalid app install name\n\n")
	}

	appInstall, err := getInstalledByName(name)
	if err != nil {
		return c.SendString("event: error\ndata: app install not found\n\n")
	}

	containerNames := splitAppInstallContainerNames(appInstall.ContainerName)
	if len(containerNames) == 0 {
		return c.SendString("event: error\ndata: container name is empty\n\n")
	}
	c.Status(200)
	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			select {
			case <-ctxRaw.Done():
				cancel()
			case <-done:
			}
		}()
		defer close(done)

		writeLine := func(line string) {
			fmt.Fprintf(w, "data: %s\n\n", line)
			_ = w.Flush()
		}

		streamErr := streamInstalledContainerLogs(ctx, containerNames, writeLine)
		if streamErr != nil && ctx.Err() == nil {
			fmt.Fprintf(w, "data: [ERROR] %s\n\n", streamErr.Error())
			_ = w.Flush()
		}
		fmt.Fprintf(w, "data: EOF\n\n")
		_ = w.Flush()
	})
	return nil
}

func splitAppInstallContainerNames(raw string) []string {
	var names []string
	for _, item := range strings.Split(raw, ",") {
		name := strings.TrimSpace(strings.TrimPrefix(item, "/"))
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func streamInstalledContainerLogs(ctx context.Context, containerNames []string, onLine func(string)) error {
	if len(containerNames) == 0 {
		return errors.New("container name is empty")
	}
	if len(containerNames) == 1 {
		return streamSingleContainerLogs(ctx, containerNames[0], false, onLine)
	}

	lineCh := make(chan string, 64)
	errCh := make(chan error, len(containerNames))
	var wg sync.WaitGroup
	for _, name := range containerNames {
		containerName := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := streamSingleContainerLogs(ctx, containerName, true, func(line string) {
				select {
				case <-ctx.Done():
				case lineCh <- line:
				}
			}); err != nil && ctx.Err() == nil {
				errCh <- err
			}
		}()
	}

	go func() {
		wg.Wait()
		close(lineCh)
		close(errCh)
	}()

	var firstErr error
	for lineCh != nil || errCh != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, ok := <-lineCh:
			if !ok {
				lineCh = nil
				continue
			}
			onLine(line)
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func streamSingleContainerLogs(ctx context.Context, containerName string, prefix bool, onLine func(string)) error {
	cmd, err := docker.RuntimeCommand(ctx, "logs", "--tail", "200", "-f", containerName)
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var output strings.Builder
	for scanner.Scan() {
		line := strings.ReplaceAll(scanner.Text(), "\r", "")
		if line == "" {
			continue
		}
		if output.Len() > 0 {
			output.WriteByte('\n')
		}
		output.WriteString(line)
		if prefix {
			onLine(fmt.Sprintf("[%s] %s", containerName, line))
		} else {
			onLine(line)
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		_ = cmd.Wait()
		return err
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil
	}
	if waitErr != nil {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			return waitErr
		}
		return fmt.Errorf("%w: %s", waitErr, msg)
	}
	return nil
}

func getInstalledByName(name string) (model.AppInstall, error) {
	installRepo := repo.NewIAppInstallRepo()
	commonRepo := repo.NewCommonRepo()
	return installRepo.GetFirst(commonRepo.WithByName(name))
}
