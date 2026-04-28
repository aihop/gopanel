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

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
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
	"gopkg.in/yaml.v3"
)

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
		if fixed, changed, ferr := applyComposeLogCompatYAML(string(composeContent)); ferr == nil && changed {
			composeContent = []byte(fixed)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed), 0644)
			logger.Info("Applied compose log driver compatibility for current runtime")
		}
		if fixed, changed, ferr := applyComposeCommandCompatYAML(string(composeContent)); ferr == nil && changed {
			composeContent = []byte(fixed)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed), 0644)
			logger.Info("Applied compose command shell compatibility for 1Panel-style templates")
		}
		if fixed, changed, ferr := applyComposeTimezoneCompatYAML(string(composeContent)); ferr == nil && changed {
			composeContent = []byte(fixed)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed), 0644)
			logger.Info("Applied compose timezone compatibility for podman on darwin")
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
			if synced, syncErr := ensureInstalledDatabaseServer(appInstall); syncErr != nil {
				if logger != nil {
					logger.Error("Auto sync database_server failed: %s", syncErr.Error())
				}
			} else if synced && logger != nil {
				logger.Info("Database server record synced: %s", appInstall.Name)
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

const autoDatabaseServerRemarkPrefix = "auto created from app install: "

func ensureInstalledDatabaseServer(appInstall *model.AppInstall) (bool, error) {
	dbType, ok := installedDatabaseServerType(strings.TrimSpace(appInstall.App.Key))
	if !ok {
		return false, nil
	}

	envMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(appInstall.Env), &envMap)

	port := uint(appInstall.HttpPort)
	if port == 0 {
		if v := firstEnvString(envMap, "PANEL_APP_PORT_HTTP"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				port = uint(n)
			}
		}
	}
	if port == 0 {
		return false, fmt.Errorf("install %s has no exposed database port", appInstall.Name)
	}

	server := &model.DatabaseServer{
		Name:      strings.TrimSpace(appInstall.Name),
		Type:      dbType,
		Host:      "127.0.0.1",
		Port:      port,
		Username:  installedDatabaseServerUsername(dbType, envMap),
		Password:  installedDatabaseServerPassword(dbType, envMap),
		Mode:      model.DatabaseModeRemote,
		Remark:    autoDatabaseServerRemarkPrefix + appInstall.Name,
		UpdatedAt: time.Now(),
	}
	if server.Name == "" {
		return false, errors.New("app install name is empty")
	}
	if server.Username == "" {
		return false, fmt.Errorf("install %s is missing database username", appInstall.Name)
	}

	serverRepo := repo.NewDatabaseServer()
	existing, err := serverRepo.GetByNameType(server.Name, dbType)
	if err == nil && existing.ID > 0 {
		if existing.Remark != "" && !strings.HasPrefix(existing.Remark, autoDatabaseServerRemarkPrefix) {
			return false, fmt.Errorf("database_server `%s` already exists and is not auto managed", server.Name)
		}
		existing.Host = server.Host
		existing.Port = server.Port
		existing.Username = server.Username
		existing.Password = server.Password
		existing.Mode = server.Mode
		existing.Remark = server.Remark
		return true, serverRepo.Update(&existing)
	}
	return true, serverRepo.Create(server)
}

func installedDatabaseServerType(appKey string) (model.DatabaseType, bool) {
	switch strings.ToLower(strings.TrimSpace(appKey)) {
	case constant.AppMysql:
		return model.DatabaseTypeMysql, true
	case constant.AppPostgresql, constant.AppPostgres:
		return model.DatabaseTypePostgresql, true
	default:
		return "", false
	}
}

func installedDatabaseServerUsername(dbType model.DatabaseType, envMap map[string]interface{}) string {
	switch dbType {
	case model.DatabaseTypeMysql:
		if v := firstEnvString(envMap, "PANEL_DB_ROOT_USER", "MYSQL_USER", "MYSQL_ROOT_USER"); v != "" {
			return v
		}
		return "root"
	case model.DatabaseTypePostgresql:
		if v := firstEnvString(envMap, "PANEL_DB_ROOT_USER", "POSTGRES_USER"); v != "" {
			return v
		}
		return "postgres"
	default:
		return ""
	}
}

func installedDatabaseServerPassword(dbType model.DatabaseType, envMap map[string]interface{}) string {
	switch dbType {
	case model.DatabaseTypeMysql:
		return firstEnvString(envMap, "PANEL_DB_ROOT_PASSWORD", "MYSQL_ROOT_PASSWORD", "MYSQL_PASSWORD")
	case model.DatabaseTypePostgresql:
		return firstEnvString(envMap, "PANEL_DB_ROOT_PASSWORD", "POSTGRES_PASSWORD")
	default:
		return ""
	}
}

func firstEnvString(envMap map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		val, ok := envMap[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(val))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
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
	applyComposeLogCompat(services)
	applyComposeCommandCompat(services)
	services[serviceName] = serviceValue
	return nil
}

func applyComposeLogCompat(services map[string]interface{}) {
	if runtime.GOOS != "linux" {
		return
	}
	resolved := docker.ResolveRuntime(context.Background())
	if resolved.Kind != docker.RuntimePodman || !docker.IsRootlessPodmanHost(resolved.Host) {
		return
	}
	for name, raw := range services {
		serviceMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		logging, ok := serviceMap["logging"].(map[string]interface{})
		if !ok || logging == nil {
			logging = map[string]interface{}{}
		}
		driver := ""
		if rawDriver, exists := logging["driver"]; exists && rawDriver != nil {
			driver = strings.TrimSpace(fmt.Sprint(rawDriver))
		}
		if driver != "" && !strings.EqualFold(driver, "journald") {
			serviceMap["logging"] = logging
			services[name] = serviceMap
			continue
		}
		logging["driver"] = "k8s-file"
		if _, ok := logging["options"].(map[string]interface{}); !ok && logging["options"] == nil {
			logging["options"] = map[string]interface{}{}
		}
		serviceMap["logging"] = logging
		services[name] = serviceMap
	}
}

func applyComposeLogCompatYAML(composeYml string) (string, bool, error) {
	var composeMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYml), &composeMap); err != nil {
		return "", false, err
	}
	services, ok := composeMap["services"].(map[string]interface{})
	if !ok || services == nil {
		return composeYml, false, nil
	}
	before, err := yaml.Marshal(services)
	if err != nil {
		return "", false, err
	}
	applyComposeLogCompat(services)
	after, err := yaml.Marshal(services)
	if err != nil {
		return "", false, err
	}
	if string(before) == string(after) {
		return composeYml, false, nil
	}
	composeMap["services"] = services
	out, err := yaml.Marshal(composeMap)
	if err != nil {
		return "", false, err
	}
	return string(out), true, nil
}

func applyComposeCommandCompat(services map[string]interface{}) {
	for name, raw := range services {
		serviceMap, ok := raw.(map[string]interface{})
		if !ok || serviceMap == nil {
			continue
		}
		command, exists := serviceMap["command"]
		if !exists || command == nil {
			continue
		}
		normalized, changed := normalizeComposeCommandShellCompat(command)
		if !changed {
			continue
		}
		serviceMap["command"] = normalized
		services[name] = serviceMap
	}
}

func applyComposeCommandCompatYAML(composeYml string) (string, bool, error) {
	var composeMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYml), &composeMap); err != nil {
		return "", false, err
	}
	services, ok := composeMap["services"].(map[string]interface{})
	if !ok || services == nil {
		return composeYml, false, nil
	}
	before, err := yaml.Marshal(services)
	if err != nil {
		return "", false, err
	}
	applyComposeCommandCompat(services)
	after, err := yaml.Marshal(services)
	if err != nil {
		return "", false, err
	}
	if string(before) == string(after) {
		return composeYml, false, nil
	}
	composeMap["services"] = services
	out, err := yaml.Marshal(composeMap)
	if err != nil {
		return "", false, err
	}
	return string(out), true, nil
}

func applyComposeTimezoneCompat(services map[string]interface{}) {
	if runtime.GOOS != "darwin" || !docker.IsPodmanRuntime(context.Background()) {
		return
	}
	tz := resolveComposeTimezoneCompatValue()
	for name, raw := range services {
		serviceMap, ok := raw.(map[string]interface{})
		if !ok || serviceMap == nil {
			continue
		}
		if normalized, changed := normalizeComposeTimezoneVolumes(serviceMap["volumes"]); changed {
			if normalized == nil {
				delete(serviceMap, "volumes")
			} else {
				serviceMap["volumes"] = normalized
			}
		}
		if normalized, changed := ensureComposeTimezoneEnv(serviceMap["environment"], tz); changed {
			serviceMap["environment"] = normalized
		}
		services[name] = serviceMap
	}
}

func applyComposeTimezoneCompatYAML(composeYml string) (string, bool, error) {
	var composeMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYml), &composeMap); err != nil {
		return "", false, err
	}
	services, ok := composeMap["services"].(map[string]interface{})
	if !ok || services == nil {
		return composeYml, false, nil
	}
	before, err := yaml.Marshal(services)
	if err != nil {
		return "", false, err
	}
	applyComposeTimezoneCompat(services)
	after, err := yaml.Marshal(services)
	if err != nil {
		return "", false, err
	}
	if string(before) == string(after) {
		return composeYml, false, nil
	}
	composeMap["services"] = services
	out, err := yaml.Marshal(composeMap)
	if err != nil {
		return "", false, err
	}
	return string(out), true, nil
}

func normalizeComposeTimezoneVolumes(volumes interface{}) (interface{}, bool) {
	list, ok := volumes.([]interface{})
	if !ok {
		return volumes, false
	}
	filtered := make([]interface{}, 0, len(list))
	changed := false
	for _, item := range list {
		if shouldRemoveComposeTimezoneVolume(item) {
			changed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !changed {
		return volumes, false
	}
	if len(filtered) == 0 {
		return nil, true
	}
	return filtered, true
}

func shouldRemoveComposeTimezoneVolume(item interface{}) bool {
	switch v := item.(type) {
	case string:
		raw := strings.TrimSpace(v)
		return strings.HasPrefix(raw, "/etc/timezone:") || strings.HasPrefix(raw, "/etc/localtime:")
	case map[string]interface{}:
		source := strings.TrimSpace(fmt.Sprint(v["source"]))
		return source == "/etc/timezone" || source == "/etc/localtime"
	default:
		return false
	}
}

func ensureComposeTimezoneEnv(environment interface{}, timezone string) (interface{}, bool) {
	switch env := environment.(type) {
	case nil:
		return []interface{}{"TZ=" + timezone}, true
	case map[string]interface{}:
		if hasComposeTimezoneEnvMap(env) {
			return environment, false
		}
		env["TZ"] = timezone
		return env, true
	case []interface{}:
		if hasComposeTimezoneEnvList(env) {
			return environment, false
		}
		return append(env, "TZ="+timezone), true
	default:
		return environment, false
	}
}

func hasComposeTimezoneEnvMap(env map[string]interface{}) bool {
	for key, value := range env {
		if strings.EqualFold(strings.TrimSpace(key), "TZ") && strings.TrimSpace(fmt.Sprint(value)) != "" {
			return true
		}
	}
	return false
}

func hasComposeTimezoneEnvList(env []interface{}) bool {
	for _, item := range env {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(fmt.Sprint(item))), "TZ=") {
			return true
		}
	}
	return false
}

func resolveComposeTimezoneCompatValue() string {
	tz := strings.TrimSpace(common.LoadTimeZoneByCmd())
	if tz == "" || strings.EqualFold(tz, "local") {
		return "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "Asia/Shanghai"
	}
	return tz
}

func normalizeComposeCommandShellCompat(command interface{}) (interface{}, bool) {
	commandStr, ok := command.(string)
	if !ok {
		return command, false
	}
	commandStr = strings.TrimSpace(commandStr)
	if commandStr == "" || !composeShellConditionalExprPattern.MatchString(commandStr) {
		return command, false
	}
	compact := strings.TrimSpace(strings.ToLower(commandStr))
	if strings.HasPrefix(compact, "/bin/sh -c ") || strings.HasPrefix(compact, "sh -c ") {
		return command, false
	}
	return []interface{}{"/bin/sh", "-c", "exec " + commandStr}, true
}
