package service

import (
	"context"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"gopkg.in/yaml.v3"
	"runtime"
	"strconv"
	"strings"
)

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
	resource["limits"] = map[string]interface{}{"cpus": "${CPUS}", "memory": "${MEMORY_LIMIT}"}
	deploy["resources"] = resource
	serviceValue["deploy"] = deploy
	if req.GpuConfig {
		resource["reservations"] = map[string]interface{}{"devices": []map[string]interface{}{{"driver": "nvidia", "count": "all", "capabilities": []string{"gpu"}}}}
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
		if runtime.GOOS == "darwin" && docker.IsPodmanRuntime(context.Background()) {
			allowHost = "0.0.0.0"
		}
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
func applyComposeCommandEnvCompat(services map[string]interface{}, envMap map[string]string) {
	for name, raw := range services {
		serviceMap, ok := raw.(map[string]interface{})
		if !ok || serviceMap == nil {
			continue
		}
		command, exists := serviceMap["command"]
		if !exists || command == nil {
			continue
		}
		normalized, changed := normalizeComposeCommandEnvCompat(command, envMap)
		if !changed {
			continue
		}
		serviceMap["command"] = normalized
		services[name] = serviceMap
	}
}
func applyComposeCommandEnvCompatYAML(composeYml string, envMap map[string]string) (string, bool, error) {
	if len(envMap) == 0 {
		return composeYml, false, nil
	}
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
	applyComposeCommandEnvCompat(services, envMap)
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
