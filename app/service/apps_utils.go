package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/i18n"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/files"
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
		app := model.App{Name: strings.ReplaceAll(appProperty.Name, "1Panel", "GoPanel"), Key: appProperty.Key, ShortDescZh: strings.ReplaceAll(strings.ReplaceAll(appProperty.ShortDescZh, "1Panel", "GoPanel"), "1panel", "gopanel"), ShortDescEn: strings.ReplaceAll(strings.ReplaceAll(appProperty.ShortDescEn, "1Panel", "GoPanel"), "1panel", "gopanel"), Description: "", Icon: appDef.Icon, Type: appProperty.Type, Status: "published", Required: strings.Join(appProperty.Required, ","), GpuSupport: appProperty.GpuSupport, CrossVersionUpdate: appProperty.CrossVersionUpdate, Limit: appProperty.Limit, Website: appProperty.Website, Github: appProperty.Github, Document: appProperty.Document, Recommend: appProperty.Recommend, Resource: constant.AppResourceRemote, ReadMe: strings.ReplaceAll(strings.ReplaceAll(appDef.ReadMe, "1Panel", "GoPanel"), "1panel", "gopanel"), LastModified: appDef.LastModified}
		descBytes, _ := json.Marshal(appProperty.Description)
		descStr := strings.ReplaceAll(string(descBytes), "1Panel", "GoPanel")
		descStr = strings.ReplaceAll(descStr, "1panel", "gopanel")
		app.Description = descStr
		var existApp model.App
		if err := global.DB.Where("key = ?", app.Key).First(&existApp).Error; err == nil && existApp.ID > 0 {
			app.ID = existApp.ID
		}
		global.DB.Save(&app)
		for _, v := range // Check if exists
		appDef.Versions {
			detail := model.AppDetail{AppId: app.ID, Version: v.Name, DownloadUrl: v.DownloadUrl, DownloadCallBackUrl: v.DownloadCallBackUrl}
			formBytes, _ := json.Marshal(v.AppForm)
			detail.Params = string(formBytes)
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

func sanitizeComposeProviderWarnings(out string) string {
	if strings.TrimSpace(out) == "" {
		return out
	}
	lines := strings.Split(out, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		plain := strings.TrimSpace(stripANSIEscapeCodes(line))
		if plain == "" {
			continue
		}
		if strings.Contains(plain, `Executing external compose provider "`) &&
			strings.Contains(plain, "Please see podman-compose(1) for how to disable this message.") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func stripANSIEscapeCodes(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); i++ {
		if text[i] != 0x1b {
			b.WriteByte(text[i])
			continue
		}
		i++
		if i >= len(text) {
			break
		}
		if text[i] != '[' {
			continue
		}
		for i+1 < len(text) {
			i++
			ch := text[i]
			if ch >= '@' && ch <= '~' {
				break
			}
		}
	}
	return b.String()
}

var InitTypes = map[string]struct{}{"runtime": {}, "php": {}, "node": {}}

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
		envMap := parseDotEnv(string(envContent))
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
		if fixed, changed, ferr := applyComposeCommandEnvCompatYAML(string(composeContent), envMap); ferr == nil && changed {
			composeContent = []byte(fixed)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed), 0644)
			logger.Info("Applied compose command env compatibility for conditional variables")
		}
		if fixed, changed, ferr := applyComposeTimezoneCompatYAML(string(composeContent)); ferr == nil && changed {
			composeContent = []byte(fixed)
			_ = files.NewFileOp().WriteFile(appInstall.GetComposePath(), strings.NewReader(fixed), 0644)
			logger.Info("Applied compose timezone compatibility for podman runtime")
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
		if runtimeDesc := installComposeRuntimeSummary(); runtimeDesc != "" {
			logger.Info("%s", runtimeDesc)
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
		out = sanitizeComposeProviderWarnings(out)
		logger.Info("Container(s) started successfully. Output: %s", out)
		return
	}
	runErr := upProject(appInstall)
	if runErr == nil {
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

var composeVarRe = regexp.MustCompile(`\\$\\{([^}]+)\\}`)
