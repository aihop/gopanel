package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/compose"
	composeV2 "github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/files"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/subosito/gotenv"
)

var DatabaseKeys = map[string]uint{
	constant.AppMysql:      3306,
	constant.AppMariaDB:    3306,
	constant.AppPostgresql: 5432,
	constant.AppPostgres:   5432,
	constant.AppMongodb:    27017,
	constant.AppRedis:      6379,
	constant.AppMemcached:  11211,
}

func checkPort(key string, params map[string]interface{}) (int, error) {
	port, ok := params[key]
	if ok {
		portN := 0
		var err error
		switch p := port.(type) {
		case string:
			portN, err = strconv.Atoi(p)
			if err != nil {
				return portN, nil
			}
		case float64:
			portN = int(math.Ceil(p))
		case int:
			portN = p
		}

		oldInstalled, _ := appInstallRepo.ListBy(appInstallRepo.WithPort(portN))
		if len(oldInstalled) > 0 {
			var apps []string
			for _, install := range oldInstalled {
				apps = append(apps, install.App.Name)
			}
			return portN, buserr.WithMap(constant.ErrPortInOtherApp, map[string]interface{}{"port": portN, "apps": apps}, nil)
		}
		if common.ScanPort(portN) {
			return portN, buserr.WithDetail(constant.ErrPortInUsed, portN, nil)
		} else {
			return portN, nil
		}
	}
	return 0, nil
}

func checkPortExist(port int) error {
	errMap := make(map[string]interface{})
	errMap["port"] = port
	appInstall, _ := appInstallRepo.GetFirst(appInstallRepo.WithPort(port))
	if appInstall.ID > 0 {
		errMap["type"] = "TYPE_APP"
		errMap["name"] = appInstall.Name
		return errors.New(constant.ErrPortInUsed)
	}
	websiteDomainRepo := repo.NewWebsiteDomain()
	domain, _ := websiteDomainRepo.GetFirst(websiteDomainRepo.WithPort(port))
	if domain.ID > 0 {
		errMap["type"] = "TYPE_DOMAIN"
		errMap["name"] = domain.Domain
		return errors.New(constant.ErrPortInUsed)
	}
	if common.ScanPort(port) {
		return errors.New(constant.ErrPortInUsed)
	}
	return nil
}

func downloadApp(app model.App, appDetail model.AppDetail, appInstall *model.AppInstall) (err error) {
	if app.IsLocalApp() {
		//本地应用,不去官网下载
		return nil
	}
	appResourceDir := path.Join(constant.AppResourceDir, app.Resource)
	appDownloadDir := app.GetAppResourcePath()
	appVersionDir := path.Join(appDownloadDir, appDetail.Version)
	fileOp := files.NewFileOp()
	if !appDetail.Update && fileOp.Stat(appVersionDir) {
		return
	}
	if !fileOp.Stat(appDownloadDir) {
		_ = fileOp.CreateDir(appDownloadDir, 0755)
	}
	if !fileOp.Stat(appVersionDir) {
		_ = fileOp.CreateDir(appVersionDir, 0755)
	}
	global.LOG.Infof("download app[%s] from %s", app.Name, appDetail.DownloadUrl)
	filePath := path.Join(appVersionDir, app.Key+"-"+appDetail.Version+".tar.gz")

	defer func() {
		if err != nil {
			if appInstall != nil {
				appInstall.Status = constant.DownloadErr
				appInstall.Message = err.Error()
			}
		}
	}()

	if err = fileOp.DownloadFileWithProxy(appDetail.DownloadUrl, filePath); err != nil {
		global.LOG.Errorf("download app[%s] error %v", app.Name, err)
		return
	}
	if err = fileOp.Decompress(filePath, appResourceDir, files.SdkTarGz, ""); err != nil {
		global.LOG.Errorf("decompress app[%s] error %v", app.Name, err)
		return
	}
	_ = fileOp.DeleteFile(filePath)

	if replaceErr := replace1PanelIdentifiers(appVersionDir); replaceErr != nil {
		global.LOG.Errorf("replace identifiers in app[%s] error %v", app.Name, replaceErr)
	}

	appDetail.Update = false

	_ = repo.NewAppDetail().CtxUpdate(context.Background(), appDetail)
	return
}

func handleMap(params map[string]interface{}, envParams map[string]string) {
	for k, v := range params {
		switch t := v.(type) {
		case string:
			envParams[k] = t
		case float64:
			envParams[k] = strconv.FormatFloat(t, 'f', -1, 32)
		case uint:
			envParams[k] = strconv.Itoa(int(t))
		case int:
			envParams[k] = strconv.Itoa(t)
		case []interface{}:
			strArray := make([]string, len(t))
			for i := range t {
				strArray[i] = strings.ToLower(fmt.Sprintf("%v", t[i]))
			}
			envParams[k] = strings.Join(strArray, ",")
		case map[string]interface{}:
			handleMap(t, envParams)
		}
	}
}

func rebuildApp(appInstall model.AppInstall) error {
	appInstall.Status = constant.Rebuilding
	_ = appInstallRepo.Save(context.Background(), &appInstall)
	go func() {
		dockerComposePath := appInstall.GetComposePath()
		out, err := compose.Down(dockerComposePath)
		if err != nil {
			_ = handleErr(appInstall, err, out)
			return
		}
		out, err = compose.Up(appInstall.GetComposePath())
		if err != nil {
			_ = handleErr(appInstall, err, out)
			return
		}
		containerNames, err := getContainerNames(appInstall)
		if err != nil {
			_ = handleErr(appInstall, err, out)
			return
		}
		appInstall.ContainerName = strings.Join(containerNames, ",")

		appInstall.Status = constant.Running
		_ = appInstallRepo.Save(context.Background(), &appInstall)
	}()
	return nil
}

func handleErr(install model.AppInstall, err error, out string) error {
	reErr := err
	install.Message = err.Error()
	if out != "" {
		install.Message = out
		reErr = errors.New(out)
	}
	install.Status = constant.UpErr
	_ = appInstallRepo.Save(context.Background(), &install)
	return reErr
}

func getContainerNames(install model.AppInstall) ([]string, error) {
	envStr, err := coverEnvJsonToStr(install.Env)
	if err != nil {
		return nil, err
	}
	project, err := composeV2.GetComposeProject(install.Name, install.GetPath(), []byte(install.DockerCompose), []byte(envStr), true)
	if err != nil {
		return nil, err
	}
	containerMap := make(map[string]struct{})
	for _, service := range project.AllServices() {
		if service.ContainerName == "${CONTAINER_NAME}" || service.ContainerName == "" {
			continue
		}
		containerMap[service.ContainerName] = struct{}{}
	}
	var containerNames []string
	for k := range containerMap {
		containerNames = append(containerNames, k)
	}
	if len(containerNames) == 0 {
		containerNames = append(containerNames, install.ContainerName)
	}
	return containerNames, nil
}

func coverEnvJsonToStr(envJson string) (string, error) {
	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(envJson), &envMap)
	newEnvMap := make(map[string]string, len(envMap))
	handleMap(envMap, newEnvMap)
	envStr, err := gotenv.Marshal(newEnvMap)
	if err != nil {
		return "", err
	}
	return envStr, nil
}

func synAppInstall(containers map[string]types.Container, appInstall *model.AppInstall, force bool) {
	containerNames := strings.Split(appInstall.ContainerName, ",")
	if len(containers) == 0 {
		if appInstall.Status == constant.UpErr && !force {
			return
		}
		appInstall.Status = constant.Error
		appInstall.Message = "ErrContainerNotFound" + strings.Join(containerNames, ",")
		_ = appInstallRepo.Save(context.Background(), appInstall)
		return
	}
	notFoundNames := make([]string, 0)
	exitNames := make([]string, 0)
	exitedCount := 0
	pausedCount := 0
	runningCount := 0
	total := len(containerNames)
	for _, name := range containerNames {
		if con, ok := containers["/"+name]; ok {
			switch con.State {
			case "exited":
				exitedCount++
				exitNames = append(exitNames, name)
			case "running":
				runningCount++
			case "paused":
				pausedCount++
			}
		} else {
			notFoundNames = append(notFoundNames, name)
		}
	}
	switch {
	case exitedCount == total:
		appInstall.Status = constant.Stopped
	case runningCount == total:
		appInstall.Status = constant.Running
	case pausedCount == total:
		appInstall.Status = constant.Paused
	case len(notFoundNames) == total:
		if appInstall.Status == constant.UpErr && !force {
			return
		}
		appInstall.Status = constant.Error
		appInstall.Message = "ErrContainerNotFound" + strings.Join(notFoundNames, ",")
	default:
		var msg string
		if exitedCount > 0 {
			msg = "ErrContainerMsg" + strings.Join(exitNames, ",") + " exited. "
		}
		if len(notFoundNames) > 0 {
			msg += "ErrContainerNotFound" + strings.Join(notFoundNames, ",") + " not found."
		}
		if msg == "" {
			msg = "ErrAppWarn"
		}
		appInstall.Message = msg
		appInstall.Status = constant.UnHealthy
	}
	_ = appInstallRepo.Save(context.Background(), appInstall)
}

func getAppInstallByKey(key string) (model.AppInstall, error) {
	app, err := appRepo.GetFirst(appRepo.WithKey(key))
	if err != nil {
		return model.AppInstall{}, err
	}
	appInstall, err := appInstallRepo.GetFirst(appInstallRepo.WithAppId(app.ID))
	if err != nil {
		return model.AppInstall{}, err
	}
	return appInstall, nil
}

func checkContainerNameIsExist(containerName, appDir string) (bool, error) {
	client, err := composeV2.NewDockerClient()
	if err != nil {
		return false, err
	}
	defer client.Close()
	var options container.ListOptions
	list, err := client.ContainerList(context.Background(), options)
	if err != nil {
		return false, err
	}
	for _, container := range list {
		if containerName == container.Names[0][1:] {
			if workDir, ok := container.Labels[composeWorkdirLabel]; ok {
				if workDir != appDir {
					return true, nil
				}
			} else {
				return true, nil
			}
		}

	}
	return false, nil
}

func createLink(ctx context.Context, app model.App, appInstall *model.AppInstall, params map[string]interface{}) error {
	var dbConfig dto.AppDatabase
	// if DatabaseKeys[app.Key] > 0 {
	// 	database := &model.Database{
	// 		AppInstallID: appInstall.ID,
	// 		Name:         appInstall.Name,
	// 		Type:         app.Key,
	// 		Version:      appInstall.Version,
	// 		From:         "local",
	// 		Address:      appInstall.ServiceName,
	// 		Port:         DatabaseKeys[app.Key],
	// 	}
	// 	detail, err := appDetailRepo.GetFirst(commonRepo.WithByID(appInstall.AppDetailId))
	// 	if err != nil {
	// 		return err
	// 	}

	// 	formFields := &dto.AppForm{}
	// 	if err := json.Unmarshal([]byte(detail.Params), formFields); err != nil {
	// 		return err
	// 	}
	// 	for _, form := range formFields.FormFields {
	// 		if form.EnvKey == "PANEL_APP_PORT_HTTP" {
	// 			portFloat, ok := form.Default.(float64)
	// 			if ok {
	// 				database.Port = uint(int(portFloat))
	// 			}
	// 			break
	// 		}
	// 	}

	// 	switch app.Key {
	// 	case constant.AppMysql, constant.AppMariaDB, constant.AppPostgresql, constant.AppMongodb:
	// 		if password, ok := params["PANEL_DB_ROOT_PASSWORD"]; ok {
	// 			if password != "" {
	// 				database.Password = password.(string)
	// 				if app.Key == "mysql" || app.Key == "mariadb" {
	// 					database.Username = "root"
	// 				}
	// 				if rootUser, ok := params["PANEL_DB_ROOT_USER"]; ok {
	// 					database.Username = rootUser.(string)
	// 				}
	// 				authParam := dto.AuthParam{
	// 					RootPassword: password.(string),
	// 					RootUser:     database.Username,
	// 				}
	// 				authByte, err := json.Marshal(authParam)
	// 				if err != nil {
	// 					return err
	// 				}
	// 				appInstall.Param = string(authByte)

	// 			}
	// 		}
	// 	case constant.AppRedis:
	// 		if password, ok := params["PANEL_REDIS_ROOT_PASSWORD"]; ok {
	// 			authParam := dto.RedisAuthParam{
	// 				RootPassword: "",
	// 			}
	// 			if password != "" {
	// 				authParam.RootPassword = password.(string)
	// 				database.Password = password.(string)
	// 			}
	// 			authByte, err := json.Marshal(authParam)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			appInstall.Param = string(authByte)
	// 		}
	// 	}
	// 	if err := databaseRepo.Create(ctx, database); err != nil {
	// 		return err
	// 	}
	// }
	// if ToolKeys[app.Key] > 0 {
	// 	if app.Key == "minio" {
	// 		authParam := dto.MinioAuthParam{}
	// 		if password, ok := params["PANEL_MINIO_ROOT_PASSWORD"]; ok {
	// 			authParam.RootPassword = password.(string)
	// 		}
	// 		if rootUser, ok := params["PANEL_MINIO_ROOT_USER"]; ok {
	// 			authParam.RootUser = rootUser.(string)
	// 		}
	// 		authByte, err := json.Marshal(authParam)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		appInstall.Param = string(authByte)
	// 	}
	// }

	if app.Type == "website" || app.Type == "tool" {
		paramByte, err := json.Marshal(params)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(paramByte, &dbConfig); err != nil {
			return err
		}
	}

	// if !reflect.DeepEqual(dbConfig, dto.AppDatabase{}) && dbConfig.ServiceName != "" {
	// 	hostName := params["PANEL_DB_HOST_NAME"]
	// 	if hostName == nil || hostName.(string) == "" {
	// 		return nil
	// 	}
	// 	database, _ := databaseRepo.Get(commonRepo.WithByName(hostName.(string)))
	// 	if database.ID == 0 {
	// 		return nil
	// 	}
	// 	var resourceId uint
	// 	if dbConfig.DbName != "" && dbConfig.DbUser != "" && dbConfig.Password != "" {
	// 		// switch database.Type {
	// 		// case constant.AppPostgresql, constant.AppPostgres:
	// 		// 	iPostgresqlRepo := repo.NewIPostgresqlRepo()
	// 		// 	oldPostgresqlDb, _ := iPostgresqlRepo.Get(commonRepo.WithByName(dbConfig.DbName), iPostgresqlRepo.WithByFrom(constant.ResourceLocal))
	// 		// 	resourceId = oldPostgresqlDb.ID
	// 		// 	if oldPostgresqlDb.ID > 0 {
	// 		// 		if oldPostgresqlDb.Username != dbConfig.DbUser || oldPostgresqlDb.Password != dbConfig.Password {
	// 		// 			return buserr.New(constant.ErrDbUserNotValid)
	// 		// 		}
	// 		// 	} else {
	// 		// 		var createPostgresql dto.PostgresqlDBCreate
	// 		// 		createPostgresql.Name = dbConfig.DbName
	// 		// 		createPostgresql.Username = dbConfig.DbUser
	// 		// 		createPostgresql.Database = database.Name
	// 		// 		createPostgresql.Format = "UTF8"
	// 		// 		createPostgresql.Password = dbConfig.Password
	// 		// 		createPostgresql.From = database.From
	// 		// 		createPostgresql.SuperUser = true
	// 		// 		pgdb, err := NewIPostgresqlService().Create(ctx, createPostgresql)
	// 		// 		if err != nil {
	// 		// 			return err
	// 		// 		}
	// 		// 		resourceId = pgdb.ID
	// 		// 	}
	// 		// case constant.AppMysql, constant.AppMariaDB:
	// 		// 	iMysqlRepo := repo.NewIMysqlRepo()
	// 		// 	oldMysqlDb, _ := iMysqlRepo.Get(commonRepo.WithByName(dbConfig.DbName), iMysqlRepo.WithByFrom(constant.ResourceLocal))
	// 		// 	resourceId = oldMysqlDb.ID
	// 		// 	if oldMysqlDb.ID > 0 {
	// 		// 		if oldMysqlDb.Username != dbConfig.DbUser || oldMysqlDb.Password != dbConfig.Password {
	// 		// 			return buserr.New(constant.ErrDbUserNotValid)
	// 		// 		}
	// 		// 	} else {
	// 		// 		var createMysql dto.MysqlDBCreate
	// 		// 		createMysql.Name = dbConfig.DbName
	// 		// 		createMysql.Username = dbConfig.DbUser
	// 		// 		createMysql.Database = database.Name
	// 		// 		createMysql.Format = "utf8mb4"
	// 		// 		createMysql.Permission = "%"
	// 		// 		createMysql.Password = dbConfig.Password
	// 		// 		createMysql.From = database.From
	// 		// 		mysqldb, err := NewIMysqlService().Create(ctx, createMysql)
	// 		// 		if err != nil {
	// 		// 			return err
	// 		// 		}
	// 		// 		resourceId = mysqldb.ID
	// 		// 	}
	// 		// }

	// 	}
	// 	// var installResource model.AppInstallResource
	// 	// installResource.ResourceId = resourceId
	// 	// installResource.AppInstallId = appInstall.ID
	// 	// if database.AppInstallID > 0 {
	// 	// 	installResource.LinkId = database.AppInstallID
	// 	// } else {
	// 	// 	installResource.LinkId = database.ID
	// 	// }
	// 	// installResource.Key = database.Type
	// 	// installResource.From = database.From
	// 	// if err := appInstallResourceRepo.Create(ctx, &installResource); err != nil {
	// 	// 	return err
	// 	// }
	// }

	return nil
}

func checkRequiredAndLimit(app model.App) error {
	if err := checkLimit(app); err != nil {
		return err
	}
	return nil
}
func checkLimit(app model.App) error {
	if app.Limit > 0 {
		installs, err := appInstallRepo.ListBy(appInstallRepo.WithAppId(app.ID))
		if err != nil {
			return err
		}
		if len(installs) >= app.Limit {
			return buserr.New(constant.ErrAppLimit)
		}
	}
	return nil
}
