package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"strconv"
	"strings"
	"time"
)

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
	server := &model.DatabaseServer{Name: strings.TrimSpace(appInstall.Name), Type: dbType, Host: "127.0.0.1", Port: port, Username: installedDatabaseServerUsername(dbType, envMap), Password: installedDatabaseServerPassword(dbType, envMap), Mode: model.DatabaseModeRemote, Remark: autoDatabaseServerRemarkPrefix + appInstall.Name, UpdatedAt: time.Now()}
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
	runtimeDesc := installComposeRuntimeSummary()
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
			if runtimeDesc != "" {
				logger.Info("%s", runtimeDesc)
			}
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
		if runtimeDesc != "" {
			return fmt.Errorf("%w; %s", lastErr, runtimeDesc)
		}
		return lastErr
	}
	if strings.TrimSpace(appInstall.Message) != "" {
		if runtimeDesc != "" {
			return fmt.Errorf("%s; %s", appInstall.Message, runtimeDesc)
		}
		return errors.New(appInstall.Message)
	}
	if runtimeDesc != "" {
		return errors.New("container did not enter running state after compose up; " + runtimeDesc)
	}
	return errors.New("container did not enter running state after compose up")
}

func installComposeRuntimeSummary() string {
	resolved := docker.ResolveRuntime(context.Background())
	parts := []string{
		fmt.Sprintf("compose runtime -> kind=%s", resolved.Kind),
		fmt.Sprintf("host=%s", strings.TrimSpace(resolved.Host)),
		fmt.Sprintf("configuredHost=%s", docker.ConfiguredDockerSockPath()),
		fmt.Sprintf("hostPinned=%t", docker.RuntimeHostPinned()),
	}
	if bin, prefix, err := compose.ResolveCommand(); err == nil {
		cmd := strings.TrimSpace(strings.Join(append([]string{bin}, prefix...), " "))
		if cmd != "" {
			parts = append(parts, "command="+cmd)
		}
	} else if msg := strings.TrimSpace(err.Error()); msg != "" {
		parts = append(parts, "commandErr="+msg)
	}
	return strings.Join(parts, ", ")
}
