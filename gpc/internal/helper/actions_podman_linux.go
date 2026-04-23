package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func (s *Server) actionPodmanRegistriesGet(ctx context.Context, params map[string]interface{}) (string, error) {
	confPath, fallbackPath := podmanRegistriesConfPaths(params)

	var mirrors []string
	usedPath := ""
	readAndParse := func(p string) bool {
		if strings.TrimSpace(p) == "" {
			return false
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return false
		}
		var config map[string]interface{}
		if err := toml.Unmarshal(data, &config); err != nil {
			return false
		}
		if registries, ok := config["registry"].([]interface{}); ok {
			for _, r := range registries {
				reg, ok := r.(map[string]interface{})
				if !ok {
					continue
				}

				location, _ := reg["location"].(string)
				prefix, _ := reg["prefix"].(string)

				if location == "docker.io" || prefix == "docker.io" {
					if ms, ok := reg["mirror"].([]interface{}); ok {
						for _, m := range ms {
							mirror, ok := m.(map[string]interface{})
							if !ok {
								continue
							}
							if loc, ok := mirror["location"].(string); ok {
								mirrors = append(mirrors, loc)
							}
						}
					}
					break
				}
			}
		}
		usedPath = p
		return true
	}
	if !readAndParse(confPath) && fallbackPath != "" {
		_ = readAndParse(fallbackPath)
	}

	resBytes, err := json.Marshal(map[string]interface{}{
		"mirrors": mirrors,
		"path":    usedPath,
	})
	if err != nil {
		return "", err
	}
	return string(resBytes), nil
}

func (s *Server) actionPodmanRegistriesSet(ctx context.Context, params map[string]interface{}) (string, error) {
	mirrorsInterface, ok := params["mirrors"].([]interface{})
	var mirrors []string
	if ok {
		for _, m := range mirrorsInterface {
			if str, isStr := m.(string); isStr {
				str = strings.TrimPrefix(str, "https://")
				str = strings.TrimPrefix(str, "http://")
				str = strings.TrimSuffix(str, "/")
				if str != "" {
					mirrors = append(mirrors, str)
				}
			}
		}
	}

	confPath, _ := podmanRegistriesConfPaths(params)
	if strings.TrimSpace(confPath) == "" {
		return "", fmt.Errorf("invalid params: registries conf path is empty")
	}
	_ = os.MkdirAll(filepath.Dir(confPath), 0755)

	var config map[string]interface{}
	if data, err := os.ReadFile(confPath); err == nil {
		_ = toml.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	searchRaw, _ := config["unqualified-search-registries"].([]interface{})
	var searchList []string
	foundDockerSearch := false
	for _, s := range searchRaw {
		if str, ok := s.(string); ok {
			searchList = append(searchList, str)
			if str == "docker.io" {
				foundDockerSearch = true
			}
		}
	}
	if !foundDockerSearch {
		searchList = append(searchList, "docker.io")
	}
	config["unqualified-search-registries"] = searchList

	registriesRaw, _ := config["registry"].([]interface{})
	foundReg := false
	var newRegistries []interface{}

	for _, r := range registriesRaw {
		reg, ok := r.(map[string]interface{})
		if !ok {
			newRegistries = append(newRegistries, r)
			continue
		}

		location, _ := reg["location"].(string)
		prefix, _ := reg["prefix"].(string)

		if location == "docker.io" || prefix == "docker.io" {
			foundReg = true
			var newMirrors []interface{}
			for _, m := range mirrors {
				newMirrors = append(newMirrors, map[string]interface{}{"location": m})
			}
			reg["mirror"] = newMirrors
			if location == "" {
				reg["location"] = "docker.io"
			}
		}
		newRegistries = append(newRegistries, reg)
	}

	if !foundReg && len(mirrors) > 0 {
		newReg := map[string]interface{}{
			"location": "docker.io",
			"prefix":   "docker.io",
		}
		var newMirrors []interface{}
		for _, m := range mirrors {
			newMirrors = append(newMirrors, map[string]interface{}{"location": m})
		}
		newReg["mirror"] = newMirrors
		newRegistries = append(newRegistries, newReg)
	}
	config["registry"] = newRegistries

	outData, err := toml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal registries.conf: %w", err)
	}

	if err := os.WriteFile(confPath, outData, 0644); err != nil {
		return "", fmt.Errorf("failed to write registries.conf: %w", err)
	}

	return "ok", nil
}

func podmanRegistriesConfPaths(params map[string]interface{}) (string, string) {
	home := strings.TrimSpace(getString(params, "home"))
	systemPath := "/etc/containers/registries.conf"
	if home == "" {
		return systemPath, ""
	}
	userPath := filepath.Join(home, ".config", "containers", "registries.conf")
	return userPath, systemPath
}

func (s *Server) actionRepairPodmanSubuid(ctx context.Context, params map[string]interface{}) (string, error) {
	username := strings.TrimSpace(getString(params, "username"))
	if username == "" {
		return "", fmt.Errorf("invalid params: username is empty")
	}

	filesToFix := []string{"/etc/subuid", "/etc/subgid"}
	var outputs []string
	needsMigrate := true // 只要用户点了修复，不管映射有没有，都强制执行一次 migrate 刷新命名空间

	for _, path := range filesToFix {
		content, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}

		if strings.Contains(string(content), username+":") {
			outputs = append(outputs, path+" already contains mapping for "+username)
			continue
		}

		// usermod --add-subuids 100000-165535 username
		var cmd *exec.Cmd
		if strings.Contains(path, "subuid") {
			cmd = exec.CommandContext(ctx, "usermod", "--add-subuids", "100000-165535", username)
		} else {
			cmd = exec.CommandContext(ctx, "usermod", "--add-subgids", "100000-165535", username)
		}

		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to run usermod for %s: %w, output: %s", path, err, string(out))
		}

		outputs = append(outputs, "Fixed "+path+" for "+username)
		needsMigrate = true
	}

	if needsMigrate {
		// Run podman system migrate for the specific user
		// Since gpc runs as root, we need to run it as the target user via su or sudo
		cmd := exec.CommandContext(ctx, "su", "-", username, "-c", "podman system migrate")
		if out, err := cmd.CombinedOutput(); err != nil {
			outputs = append(outputs, fmt.Sprintf("Warning: podman system migrate failed: %v, output: %s", err, string(out)))
		} else {
			outputs = append(outputs, "Ran podman system migrate successfully")
		}
	}

	return strings.Join(outputs, "\n"), nil
}
