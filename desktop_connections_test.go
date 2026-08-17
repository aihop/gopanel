//go:build desktop

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLoadDesktopConnectionMigratesLegacyServer(t *testing.T) {
	baseDir := t.TempDir()
	legacy := `{"mode":"remote","url":"http://127.0.0.1:15470","entrance":"safe-entry"}`
	if err := os.WriteFile(filepath.Join(baseDir, "desktop.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := loadDesktopConnection(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Servers) != 1 || config.Servers[0].Name != "127.0.0.1:15470" || config.Servers[0].Entrance != "safe-entry" {
		t.Fatalf("legacy server was not migrated: %#v", config)
	}
}

func TestRememberDesktopServerDeduplicatesAndLimitsHistory(t *testing.T) {
	config := desktopConnectionConfig{}
	for index := 0; index < desktopRecentServerLimit+2; index++ {
		rememberDesktopServer(&config, desktopServerConfig{URL: "http://127.0.0.1:" + strconv.Itoa(15000+index)})
	}
	rememberDesktopServer(&config, desktopServerConfig{Name: "Production", URL: "http://127.0.0.1:5", Entrance: "/secure/"})
	if len(config.Servers) != desktopRecentServerLimit {
		t.Fatalf("server history length = %d", len(config.Servers))
	}
	if config.Servers[0].Name != "Production" || config.Servers[0].URL != "http://127.0.0.1:5" || config.Servers[0].Entrance != "secure" {
		t.Fatalf("server was not normalized and moved to front: %#v", config.Servers[0])
	}
}

func TestRemoveDesktopServerRejectsCurrentConnection(t *testing.T) {
	config := desktopConnectionConfig{
		Mode:    "remote",
		URL:     "http://127.0.0.1:15470",
		Servers: []desktopServerConfig{{Name: "Current", URL: "http://127.0.0.1:15470"}},
	}
	if err := removeDesktopServer(&config, config.URL); err == nil {
		t.Fatal("current connection should not be removable")
	}
}

func TestDesktopConnectRetainsRecentServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			_, _ = io.WriteString(response, `{"code":0,"data":{"appName":"GoPanel"}}`)
			return
		}
		_, _ = io.WriteString(response, "ok")
	}))
	defer server.Close()
	gateway := &desktopGateway{
		baseDir: t.TempDir(),
		config:  desktopConnectionConfig{Servers: []desktopServerConfig{{Name: "Old", URL: "http://127.0.0.1:15470"}}},
	}
	payload, _ := json.Marshal(map[string]string{"name": "Production", "url": server.URL})
	response := httptest.NewRecorder()
	gateway.handleConnect(response, httptest.NewRequest(http.MethodPost, desktopSettingsPath+"/connect", strings.NewReader(string(payload))))
	if response.Code != http.StatusOK {
		t.Fatalf("connect response: %d %s", response.Code, response.Body.String())
	}
	if len(gateway.config.Servers) != 2 || gateway.config.Servers[0].Name != "Production" || gateway.config.Servers[1].Name != "Old" {
		t.Fatalf("recent servers were not preserved: %#v", gateway.config.Servers)
	}
}

func TestDesktopConnectReusesSavedEntranceForSameServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			_, _ = io.WriteString(response, `{"code":0,"data":{"appName":"GoPanel"}}`)
			return
		}
		if request.Header.Get("EntranceCode") != "c2F2ZWQtZW50cnk=" {
			response.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(response, "ok")
	}))
	defer server.Close()
	gateway := &desktopGateway{
		baseDir: t.TempDir(),
		config:  desktopConnectionConfig{Servers: []desktopServerConfig{{Name: "Saved", URL: server.URL, Entrance: "saved-entry"}}},
	}
	payload, _ := json.Marshal(map[string]string{"url": server.URL})
	response := httptest.NewRecorder()
	gateway.handleConnect(response, httptest.NewRequest(http.MethodPost, desktopSettingsPath+"/connect", strings.NewReader(string(payload))))
	if response.Code != http.StatusOK || gateway.config.Entrance != "saved-entry" {
		t.Fatalf("saved entrance was not reused: %d %#v %s", response.Code, gateway.config, response.Body.String())
	}
}

func TestDesktopConnectionStateDoesNotExposeEntrance(t *testing.T) {
	gateway := &desktopGateway{config: desktopConnectionConfig{Servers: []desktopServerConfig{{
		Name: "Production", URL: "https://panel.example.com", Entrance: "top-secret",
	}}}}
	data, err := json.Marshal(gateway.connectionState())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "top-secret") || !strings.Contains(string(data), `"hasEntrance":true`) {
		t.Fatalf("connection state exposed or lost entrance metadata: %s", data)
	}
}

func TestDesktopLauncherIncludesConnectionCenterStates(t *testing.T) {
	body := (&desktopGateway{baseDir: t.TempDir()}).launcherHTML()
	for _, expected := range []string{"服务器连接中心", "最近服务器", "正在加载服务器", "服务器列表加载失败", "暂无最近服务器", "clearServerSession"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("connection center missing %q", expected)
		}
	}
}
