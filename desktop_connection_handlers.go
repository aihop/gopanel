//go:build desktop

package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

func (gateway *desktopGateway) handleConnectionState(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeDesktopResult(response, http.StatusOK, map[string]any{"ok": true, "data": gateway.connectionState()})
}

func (gateway *desktopGateway) handleConnect(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		Entrance      string `json:"entrance"`
		ClearEntrance bool   `json:"clearEntrance"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeDesktopJSON(response, http.StatusBadRequest, err)
		return
	}
	target, entrance, err := normalizeDesktopConnection(payload.URL, payload.Entrance)
	if err != nil {
		writeDesktopJSON(response, http.StatusBadRequest, errors.New("请输入有效的 GoPanel 服务地址"))
		return
	}
	if entrance == "" && !payload.ClearEntrance {
		gateway.RLock()
		if saved, ok := desktopSavedServer(&gateway.config, target); ok {
			entrance = saved.Entrance
		}
		gateway.RUnlock()
	}
	if !desktopTargetHealthy(target) {
		writeDesktopJSON(response, http.StatusBadGateway, errors.New("无法连接该 GoPanel 服务，请确认地址和端口"))
		return
	}
	if err := desktopTargetAccessError(target, entrance); err != nil {
		writeDesktopJSON(response, http.StatusForbidden, err)
		return
	}
	gateway.RLock()
	builtinRunning := gateway.builtinRunning
	gateway.RUnlock()
	if builtinRunning {
		gateway.rememberRemoteConnection(payload.Name, target.String(), entrance)
		if err := gateway.saveConfig(); err != nil {
			writeDesktopJSON(response, http.StatusInternalServerError, err)
			return
		}
		writeDesktopResult(response, http.StatusConflict, map[string]any{
			"ok": false, "restart": true, "error": "连接地址已保存，请重启 GoPanel 完成切换",
		})
		return
	}
	gateway.setTarget(target, "", desktopMobileURL(target), entrance)
	gateway.rememberRemoteConnection(payload.Name, target.String(), entrance)
	if err := gateway.saveConfig(); err != nil {
		writeDesktopJSON(response, http.StatusInternalServerError, err)
		return
	}
	writeDesktopJSON(response, http.StatusOK, nil)
}

func (gateway *desktopGateway) rememberRemoteConnection(name, targetURL, entrance string) {
	target, _ := normalizeDesktopTarget(targetURL)
	gateway.Lock()
	gateway.config.Mode = "remote"
	gateway.config.URL = targetURL
	gateway.config.Entrance = entrance
	rememberDesktopServer(&gateway.config, desktopServerConfig{Name: desktopServerName(name, target), URL: targetURL, Entrance: entrance})
	gateway.Unlock()
}

func (gateway *desktopGateway) handleBuiltin(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gateway.RLock()
	starter := gateway.builtinStarter
	gateway.RUnlock()
	if starter == nil {
		writeDesktopJSON(response, http.StatusServiceUnavailable, errors.New("内置服务尚未准备好"))
		return
	}
	if err := starter(); err != nil {
		writeDesktopJSON(response, http.StatusInternalServerError, err)
		return
	}
	gateway.Lock()
	gateway.config.Mode = "builtin"
	gateway.config.URL = ""
	gateway.config.Entrance = ""
	gateway.Unlock()
	if err := gateway.saveConfig(); err != nil {
		writeDesktopJSON(response, http.StatusInternalServerError, err)
		return
	}
	writeDesktopJSON(response, http.StatusOK, nil)
}

func (gateway *desktopGateway) handleDeleteServer(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeDesktopJSON(response, http.StatusBadRequest, err)
		return
	}
	gateway.Lock()
	err := removeDesktopServer(&gateway.config, payload.URL)
	gateway.Unlock()
	if err != nil {
		writeDesktopJSON(response, http.StatusConflict, err)
		return
	}
	if err := gateway.saveConfig(); err != nil {
		writeDesktopJSON(response, http.StatusInternalServerError, err)
		return
	}
	writeDesktopJSON(response, http.StatusOK, nil)
}
