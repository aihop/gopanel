//go:build desktop

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const desktopRecentServerLimit = 8

const (
	desktopServerNameLimit     = 80
	desktopServerEntranceLimit = 255
)

type desktopServerConfig struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Entrance string `json:"entrance,omitempty"`
}

type desktopConnectionConfig struct {
	Mode     string                `json:"mode"`
	URL      string                `json:"url,omitempty"`
	Entrance string                `json:"entrance,omitempty"`
	Servers  []desktopServerConfig `json:"servers,omitempty"`
}

type desktopCurrentConnection struct {
	Mode   string `json:"mode"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Online bool   `json:"online"`
}

type desktopServerSummary struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	HasEntrance bool   `json:"hasEntrance"`
}

type desktopConnectionState struct {
	Current desktopCurrentConnection `json:"current"`
	Servers []desktopServerSummary   `json:"servers"`
}

func loadDesktopConnection(baseDir string) (desktopConnectionConfig, error) {
	data, err := os.ReadFile(filepath.Join(baseDir, "desktop.json"))
	if os.IsNotExist(err) {
		return desktopConnectionConfig{}, nil
	}
	if err != nil {
		return desktopConnectionConfig{}, fmt.Errorf("read desktop connection: %w", err)
	}
	var config desktopConnectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return desktopConnectionConfig{}, fmt.Errorf("parse desktop connection: %w", err)
	}
	normalizeDesktopServers(&config)
	return config, nil
}

func (gateway *desktopGateway) saveConfig() error {
	gateway.RLock()
	config := gateway.config
	gateway.RUnlock()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(gateway.baseDir, "desktop.json"), data, 0o600); err != nil {
		return fmt.Errorf("save desktop connection: %w", err)
	}
	return nil
}

func (gateway *desktopGateway) connectionState() desktopConnectionState {
	gateway.RLock()
	config := gateway.config
	servers := append([]desktopServerConfig(nil), config.Servers...)
	var target *url.URL
	if gateway.target != nil {
		copyTarget := *gateway.target
		target = &copyTarget
	}
	desktopToken := gateway.desktopToken
	gateway.RUnlock()
	current := desktopCurrentConnection{Mode: config.Mode, Name: "未连接"}
	if target != nil {
		current.URL = target.String()
		current.Online = desktopTargetHealthyWithToken(target, desktopToken)
		if config.Mode == "builtin" {
			current.Name = "本机内置服务"
		} else {
			current.Name = desktopServerName("", target)
			for _, server := range servers {
				if server.URL == config.URL {
					current.Name = server.Name
					break
				}
			}
		}
	}
	summaries := make([]desktopServerSummary, 0, len(servers))
	for _, server := range servers {
		summaries = append(summaries, desktopServerSummary{
			Name:        server.Name,
			URL:         server.URL,
			HasEntrance: server.Entrance != "",
		})
	}
	return desktopConnectionState{Current: current, Servers: summaries}
}

func normalizeDesktopServers(config *desktopConnectionConfig) {
	servers := append([]desktopServerConfig(nil), config.Servers...)
	if config.Mode == "remote" && config.URL != "" {
		name := ""
		for _, server := range servers {
			if target, err := normalizeDesktopTarget(server.URL); err == nil && target.String() == config.URL {
				name = server.Name
				break
			}
		}
		servers = append([]desktopServerConfig{{Name: name, URL: config.URL, Entrance: config.Entrance}}, servers...)
	}
	config.Servers = nil
	seen := make(map[string]struct{}, len(servers))
	for _, server := range servers {
		target, err := normalizeDesktopTarget(server.URL)
		if err != nil {
			continue
		}
		if _, exists := seen[target.String()]; exists || len(config.Servers) >= desktopRecentServerLimit {
			continue
		}
		seen[target.String()] = struct{}{}
		config.Servers = append(config.Servers, desktopServerConfig{
			Name:     desktopServerName(server.Name, target),
			URL:      target.String(),
			Entrance: trimDesktopValue(server.Entrance, desktopServerEntranceLimit),
		})
	}
}

func rememberDesktopServer(config *desktopConnectionConfig, server desktopServerConfig) {
	target, err := normalizeDesktopTarget(server.URL)
	if err != nil {
		return
	}
	server.Name = desktopServerName(server.Name, target)
	server.URL = target.String()
	server.Entrance = trimDesktopValue(server.Entrance, desktopServerEntranceLimit)
	filtered := make([]desktopServerConfig, 0, desktopRecentServerLimit)
	filtered = append(filtered, server)
	for _, existing := range config.Servers {
		if existing.URL == server.URL || len(filtered) >= desktopRecentServerLimit {
			continue
		}
		filtered = append(filtered, existing)
	}
	config.Servers = filtered
}

func desktopServerName(name string, target *url.URL) string {
	name = trimDesktopValue(name, desktopServerNameLimit)
	if name != "" {
		return name
	}
	if target == nil {
		return "GoPanel"
	}
	return target.Host
}

func desktopSavedServer(config *desktopConnectionConfig, target *url.URL) (desktopServerConfig, bool) {
	if target == nil {
		return desktopServerConfig{}, false
	}
	for _, server := range config.Servers {
		if server.URL == target.String() {
			return server, true
		}
	}
	return desktopServerConfig{}, false
}

func trimDesktopValue(value string, limit int) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	characters := []rune(value)
	if len(characters) > limit {
		value = string(characters[:limit])
	}
	return value
}

func removeDesktopServer(config *desktopConnectionConfig, rawURL string) error {
	target, err := normalizeDesktopTarget(rawURL)
	if err != nil {
		return errors.New("服务器地址无效")
	}
	if config.Mode == "remote" && config.URL == target.String() {
		return errors.New("当前连接不能删除，请先切换到其他服务器")
	}
	filtered := config.Servers[:0]
	for _, server := range config.Servers {
		if server.URL != target.String() {
			filtered = append(filtered, server)
		}
	}
	config.Servers = filtered
	return nil
}
