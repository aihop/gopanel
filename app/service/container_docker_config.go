package service

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
	"os"
	"path"
	"runtime"
	"strings"
)

func (u *DockerService) UpdateConf(req dto.SettingUpdate) error {
	if err := ensureLinuxDockerConfigRuntime(); err != nil {
		if isPodmanRuntimeConfigured() && req.Key == "Mirrors" {
			req.Value = strings.TrimSuffix(req.Value, ",")
			var mirrors []string
			if len(req.Value) > 0 {
				mirrors = strings.Split(req.Value, ",")
			}
			if runtime.GOOS == "darwin" {
				return podmanMachineRegistriesSet(context.Background(), mirrors)
			}
			params := map[string]interface{}{"mirrors": mirrors}
			if home := podmanRootlessHomeFromHost(docker.ResolveRuntime(context.Background()).Host); home != "" {
				params["home"] = home
			}
			_, err := gpc.Do(context.Background(), "PODMAN_REGISTRIES_SET", params)
			return err
		}
		return err
	}
	err := createIfNotExistDaemonJsonFile()
	if err != nil {
		return err
	}
	file, err := os.ReadFile(constant.DaemonJsonPath)
	if err != nil {
		return err
	}
	daemonMap := make(map[string]interface{})
	_ = json.Unmarshal(file, &daemonMap)
	switch req.Key {
	case "Registries":
		req.Value = strings.TrimSuffix(req.Value, ",")
		if len(req.Value) == 0 {
			delete(daemonMap, "insecure-registries")
		} else {
			daemonMap["insecure-registries"] = strings.Split(req.Value, ",")
		}
	case "Mirrors":
		req.Value = strings.TrimSuffix(req.Value, ",")
		if len(req.Value) == 0 {
			delete(daemonMap, "registry-mirrors")
		} else {
			daemonMap["registry-mirrors"] = strings.Split(req.Value, ",")
		}
	case "Ipv6":
		if req.Value == "disable" {
			delete(daemonMap, "ipv6")
			delete(daemonMap, "fixed-cidr-v6")
			delete(daemonMap, "ip6tables")
			delete(daemonMap, "experimental")
		}
	case "LogOption":
		if req.Value == "disable" {
			delete(daemonMap, "log-opts")
		}
	case "LiveRestore":
		if req.Value == "disable" {
			delete(daemonMap, "live-restore")
		} else {
			daemonMap["live-restore"] = true
		}
	case "IPtables":
		if req.Value == "enable" {
			delete(daemonMap, "iptables")
		} else {
			daemonMap["iptables"] = false
		}
	case "Driver":
		if opts, ok := daemonMap["exec-opts"]; ok {
			if optsValue, isArray := opts.([]interface{}); isArray {
				for i := 0; i < len(optsValue); i++ {
					if opt, isStr := optsValue[i].(string); isStr {
						if strings.HasPrefix(opt, "native.cgroupdriver=") {
							optsValue[i] = "native.cgroupdriver=" + req.Value
							break
						}
					}
				}
			}
		} else {
			if req.Value == "systemd" {
				daemonMap["exec-opts"] = []string{"native.cgroupdriver=systemd"}
			}
		}
	case "http-proxy", "https-proxy":
		delete(daemonMap, "proxies")
		if len(req.Value) > 0 {
			proxies := map[string]interface{}{req.Key: req.Value}
			daemonMap["proxies"] = proxies
		}
	case "socks5-proxy", "close-proxy":
		delete(daemonMap, "proxies")
		if len(req.Value) > 0 {
			proxies := map[string]interface{}{"http-proxy": req.Value, "https-proxy": req.Value}
			daemonMap["proxies"] = proxies
		}
	}
	if len(daemonMap) == 0 {
		_ = os.Remove(constant.DaemonJsonPath)
		if err := restartDocker(); err != nil {
			return err
		}
		return nil
	}
	newJson, err := json.MarshalIndent(daemonMap, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(constant.DaemonJsonPath, newJson, 0640); err != nil {
		return err
	}
	if err := validateDockerConfig(); err != nil {
		return err
	}
	if err := restartDocker(); err != nil {
		return err
	}
	return nil
}
func createIfNotExistDaemonJsonFile() error {
	if _, err := os.Stat(constant.DaemonJsonPath); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(constant.DaemonJsonPath), os.ModePerm); err != nil {
			return err
		}
		var daemonFile *os.File
		daemonFile, err = os.Create(constant.DaemonJsonPath)
		if err != nil {
			return err
		}
		defer daemonFile.Close()
	}
	return nil
}
func (u *DockerService) UpdateLogOption(req dto.LogOption) error {
	if err := ensureLinuxDockerConfigRuntime(); err != nil {
		return err
	}
	err := createIfNotExistDaemonJsonFile()
	if err != nil {
		return err
	}
	file, err := os.ReadFile(constant.DaemonJsonPath)
	if err != nil {
		return err
	}
	daemonMap := make(map[string]interface{})
	_ = json.Unmarshal(file, &daemonMap)
	changeLogOption(daemonMap, req.LogMaxFile, req.LogMaxSize)
	if len(daemonMap) == 0 {
		_ = os.Remove(constant.DaemonJsonPath)
		return nil
	}
	newJson, err := json.MarshalIndent(daemonMap, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(constant.DaemonJsonPath, newJson, 0640); err != nil {
		return err
	}
	if err := validateDockerConfig(); err != nil {
		return err
	}
	if err := restartDocker(); err != nil {
		return err
	}
	return nil
}
func (u *DockerService) UpdateIpv6Option(req dto.Ipv6Option) error {
	if err := ensureLinuxDockerConfigRuntime(); err != nil {
		return err
	}
	err := createIfNotExistDaemonJsonFile()
	if err != nil {
		return err
	}
	file, err := os.ReadFile(constant.DaemonJsonPath)
	if err != nil {
		return err
	}
	daemonMap := make(map[string]interface{})
	_ = json.Unmarshal(file, &daemonMap)
	daemonMap["ipv6"] = true
	daemonMap["fixed-cidr-v6"] = req.FixedCidrV6
	if req.Ip6Tables {
		daemonMap["ip6tables"] = req.Ip6Tables
	}
	if req.Experimental {
		daemonMap["experimental"] = req.Experimental
	}
	if len(daemonMap) == 0 {
		_ = os.Remove(constant.DaemonJsonPath)
		return nil
	}
	newJson, err := json.MarshalIndent(daemonMap, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(constant.DaemonJsonPath, newJson, 0640); err != nil {
		return err
	}
	if err := validateDockerConfig(); err != nil {
		return err
	}
	if err := restartDocker(); err != nil {
		return err
	}
	return nil
}
func (u *DockerService) UpdateConfByFile(req dto.DaemonJsonUpdateByFile) error {
	if err := ensureLinuxDockerConfigRuntime(); err != nil {
		return err
	}
	if len(req.File) == 0 {
		_ = os.Remove(constant.DaemonJsonPath)
		if err := restartDocker(); err != nil {
			return err
		}
		return nil
	}
	err := createIfNotExistDaemonJsonFile()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(constant.DaemonJsonPath, os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer file.Close()
	write := bufio.NewWriter(file)
	_, _ = write.WriteString(req.File)
	write.Flush()
	if err := validateDockerConfig(); err != nil {
		return err
	}
	if err := restartDocker(); err != nil {
		return err
	}
	return nil
}
func changeLogOption(daemonMap map[string]interface{}, logMaxFile, logMaxSize string) {
	if opts, ok := daemonMap["log-opts"]; ok {
		if len(logMaxFile) != 0 || len(logMaxSize) != 0 {
			daemonMap["log-driver"] = "json-file"
		}
		optsMap, isMap := opts.(map[string]interface{})
		if isMap {
			if len(logMaxFile) != 0 {
				optsMap["max-file"] = logMaxFile
			} else {
				delete(optsMap, "max-file")
			}
			if len(logMaxSize) != 0 {
				optsMap["max-size"] = logMaxSize
			} else {
				delete(optsMap, "max-size")
			}
			if len(optsMap) == 0 {
				delete(daemonMap, "log-opts")
			}
		} else {
			optsMap := make(map[string]interface{})
			if len(logMaxFile) != 0 {
				optsMap["max-file"] = logMaxFile
			}
			if len(logMaxSize) != 0 {
				optsMap["max-size"] = logMaxSize
			}
			if len(optsMap) != 0 {
				daemonMap["log-opts"] = optsMap
			}
		}
	} else {
		if len(logMaxFile) != 0 || len(logMaxSize) != 0 {
			daemonMap["log-driver"] = "json-file"
		}
		optsMap := make(map[string]interface{})
		if len(logMaxFile) != 0 {
			optsMap["max-file"] = logMaxFile
		}
		if len(logMaxSize) != 0 {
			optsMap["max-size"] = logMaxSize
		}
		if len(optsMap) != 0 {
			daemonMap["log-opts"] = optsMap
		}
	}
}
