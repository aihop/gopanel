package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func (s *Server) actionPodmanRegistriesGet(ctx context.Context, params map[string]interface{}) (string, error) {
	confPath := "/etc/containers/registries.conf"

	var mirrors []string
	if data, err := os.ReadFile(confPath); err == nil {
		var config map[string]interface{}
		if err := toml.Unmarshal(data, &config); err == nil {
			// Parse V2 registry block
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
		}
	}

	resBytes, err := json.Marshal(map[string]interface{}{
		"mirrors": mirrors,
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

	confPath := "/etc/containers/registries.conf"
	_ = os.MkdirAll("/etc/containers", 0755)

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
