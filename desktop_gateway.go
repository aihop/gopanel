//go:build desktop

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/middleware"
	"github.com/spf13/viper"
)

const desktopSettingsPath = "/__desktop"

type desktopConnectionConfig struct {
	Mode     string `json:"mode"`
	URL      string `json:"url,omitempty"`
	Entrance string `json:"entrance,omitempty"`
}

type desktopGateway struct {
	sync.RWMutex
	baseDir        string
	config         desktopConnectionConfig
	target         *url.URL
	proxy          *httputil.ReverseProxy
	desktopToken   string
	entrance       string
	mobileURL      string
	builtinStarter func() error
	builtinRunning bool
}

func newDesktopGateway(baseDir string) (*desktopGateway, error) {
	gateway := &desktopGateway{baseDir: baseDir}
	config, err := loadDesktopConnection(baseDir)
	if err != nil {
		return nil, err
	}
	gateway.config = config
	if config.Mode == "remote" && config.URL != "" {
		if target, normalizeErr := normalizeDesktopTarget(config.URL); normalizeErr == nil && desktopTargetHealthy(target) {
			entrance := config.Entrance
			if entrance == "" {
				entrance = discoverLocalDesktopEntrance(baseDir, target)
			}
			if desktopTargetAccessError(target, entrance) == nil {
				gateway.config.Entrance = entrance
				gateway.setTarget(target, "", desktopMobileURL(target), entrance)
				_ = gateway.saveConfig()
			}
		}
	}
	if gateway.target == nil && config.Mode == "" {
		if target := discoverLocalDesktopTarget(baseDir); target != nil && desktopTargetHealthy(target) {
			entrance := discoverLocalDesktopEntrance(baseDir, target)
			if desktopTargetAccessError(target, entrance) == nil {
				gateway.config = desktopConnectionConfig{Mode: "remote", URL: target.String(), Entrance: entrance}
				gateway.setTarget(target, "", desktopMobileURL(target), entrance)
				_ = gateway.saveConfig()
			}
		}
	}
	return gateway, nil
}

func (gateway *desktopGateway) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	gateway.RLock()
	proxy := gateway.proxy
	token := gateway.desktopToken
	entrance := gateway.entrance
	gateway.RUnlock()
	if proxy == nil {
		http.Error(response, "GoPanel desktop is not connected", http.StatusServiceUnavailable)
		return
	}
	if token != "" {
		request.Header.Set("X-GoPanel-Desktop-Token", token)
	}
	if entrance != "" {
		request.Header.Set("EntranceCode", base64.StdEncoding.EncodeToString([]byte(entrance)))
	}
	proxy.ServeHTTP(response, request)
}

func (gateway *desktopGateway) ready() bool {
	gateway.RLock()
	defer gateway.RUnlock()
	return gateway.proxy != nil
}

func (gateway *desktopGateway) shouldStartBuiltin() bool {
	return gateway.config.Mode == "builtin" && !gateway.ready()
}

func (gateway *desktopGateway) setBuiltinStarter(starter func() error) {
	gateway.Lock()
	gateway.builtinStarter = starter
	gateway.Unlock()
}

func (gateway *desktopGateway) useBuiltin(target *url.URL, mobileURL, token string) {
	gateway.setTarget(target, token, mobileURL, "")
	gateway.Lock()
	gateway.config = desktopConnectionConfig{Mode: "builtin"}
	gateway.builtinRunning = true
	gateway.Unlock()
	_ = gateway.saveConfig()
}

func (gateway *desktopGateway) setTarget(target *url.URL, token, mobileURL, entrance string) {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(response http.ResponseWriter, _ *http.Request, err error) {
		http.Error(response, err.Error(), http.StatusBadGateway)
	}
	gateway.Lock()
	gateway.target = target
	gateway.proxy = proxy
	gateway.desktopToken = token
	gateway.entrance = entrance
	gateway.mobileURL = mobileURL
	gateway.Unlock()
}

func (gateway *desktopGateway) handleDesktopRoute(response http.ResponseWriter, request *http.Request) bool {
	if request.URL.Path != desktopSettingsPath && request.URL.Path != desktopSettingsPath+"/connect" && request.URL.Path != desktopSettingsPath+"/builtin" {
		return false
	}
	switch request.URL.Path {
	case desktopSettingsPath:
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Cache-Control", "no-store")
		_, _ = response.Write([]byte(gateway.launcherHTML()))
	case desktopSettingsPath + "/connect":
		gateway.handleConnect(response, request)
	case desktopSettingsPath + "/builtin":
		gateway.handleBuiltin(response, request)
	}
	return true
}

func (gateway *desktopGateway) handleConnect(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		URL      string `json:"url"`
		Entrance string `json:"entrance"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeDesktopJSON(response, http.StatusBadRequest, err)
		return
	}
	target, entrance, err := normalizeDesktopConnection(payload.URL, payload.Entrance)
	if err != nil || !desktopTargetHealthy(target) {
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
		gateway.Lock()
		gateway.config = desktopConnectionConfig{Mode: "remote", URL: target.String(), Entrance: entrance}
		gateway.Unlock()
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
	gateway.Lock()
	gateway.config = desktopConnectionConfig{Mode: "remote", URL: target.String(), Entrance: entrance}
	gateway.Unlock()
	if err := gateway.saveConfig(); err != nil {
		writeDesktopJSON(response, http.StatusInternalServerError, err)
		return
	}
	writeDesktopJSON(response, http.StatusOK, nil)
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
	writeDesktopJSON(response, http.StatusOK, nil)
}

func (gateway *desktopGateway) bootstrapHTML() string {
	gateway.RLock()
	target := gateway.target
	token := gateway.desktopToken
	entrance := gateway.entrance
	mobileURL := gateway.mobileURL
	gateway.RUnlock()
	if target == nil {
		return ""
	}
	websocketScheme := "ws"
	if target.Scheme == "https" {
		websocketScheme = "wss"
	}
	return fmt.Sprintf(`<style>*{scrollbar-width:none!important}*::-webkit-scrollbar{display:none!important;width:0!important;height:0!important}</style><script>(()=>{window.__GOPANEL_DESKTOP_MOBILE_URL__=%s;const N=window.WebSocket;window.WebSocket=class extends N{constructor(u,p){const x=new URL(u,window.location.href);if(x.protocol==="wails:"||x.hostname==="wails"||x.hostname==="wails.localhost"||x.hostname.endsWith(".wails")){x.protocol=%s;x.host=%s;%s}super(x.toString(),p)}}})();</script>`,
		strconv.Quote(mobileURL), strconv.Quote(websocketScheme+":"), strconv.Quote(target.Host), desktopAccessQuery(token, entrance))
}

func desktopAccessQuery(token, entrance string) string {
	result := ""
	if token != "" {
		result += "x.searchParams.set(\"desktop_token\"," + strconv.Quote(token) + ");"
	}
	if entrance != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(entrance))
		result += "x.searchParams.set(\"entrance\"," + strconv.Quote(encoded) + ");"
	}
	return result
}

func normalizeDesktopConnection(rawURL, rawEntrance string) (*url.URL, string, error) {
	value := strings.TrimSpace(rawURL)
	if value != "" && !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, "", err
	}
	entrance := strings.Trim(strings.TrimSpace(rawEntrance), "/")
	if entrance == "" {
		entrance = strings.Trim(strings.TrimSpace(parsed.Path), "/")
	}
	target, err := normalizeDesktopTarget(value)
	return target, entrance, err
}

func normalizeDesktopTarget(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("GoPanel 服务地址不能为空")
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	target, err := url.Parse(value)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" || target.User != nil {
		return nil, errors.New("请输入有效的 HTTP 或 HTTPS 地址")
	}
	target.Path, target.RawPath, target.RawQuery, target.Fragment = "", "", "", ""
	return target, nil
}

func desktopTargetHealthy(target *url.URL) bool {
	if target == nil {
		return false
	}
	healthURL := *target
	healthURL.Path = "/health"
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	response, err := client.Get(healthURL.String())
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return false
	}
	var payload struct {
		Code int `json:"code"`
		Data struct {
			AppName  string `json:"appName"`
			AppBrand string `json:"appBrand"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &payload) != nil || payload.Code != 0 {
		return false
	}
	return strings.Contains(strings.ToLower(payload.Data.AppName+payload.Data.AppBrand), "gopanel")
}

func desktopTargetAccessError(target *url.URL, entrance string) error {
	if target == nil {
		return errors.New("GoPanel 服务地址无效")
	}
	accessURL := *target
	accessURL.Path = "/"
	request, err := http.NewRequest(http.MethodGet, accessURL.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if entrance != "" {
		request.Header.Set("EntranceCode", base64.StdEncoding.EncodeToString([]byte(entrance)))
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		return errors.New("无法访问该 GoPanel 服务")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden {
		if entrance == "" {
			return errors.New("该服务已开启安全防护，请填写专属安全入口")
		}
		return errors.New("专属安全入口不正确，请重新填写")
	}
	return nil
}

func desktopMobileURL(target *url.URL) string {
	return desktopMobileURLWithIP(target, desktopLANIP())
}

func desktopMobileURLWithIP(target *url.URL, lanIP string) string {
	result := *target
	host := result.Hostname()
	if host == "localhost" || net.ParseIP(host).IsLoopback() {
		port := result.Port()
		result.Host = lanIP
		if port != "" {
			result.Host = net.JoinHostPort(result.Host, port)
		}
	}
	result.Path = "/mobile"
	return result.String()
}

func discoverLocalDesktopTarget(baseDir string) *url.URL {
	configuration := viper.New()
	configuration.SetConfigFile(filepath.Join(baseDir, "conf.yaml"))
	if err := configuration.ReadInConfig(); err != nil {
		return nil
	}
	address := strings.TrimSpace(configuration.GetString("system.port"))
	if strings.HasPrefix(address, ":") {
		address = "127.0.0.1" + address
	} else if _, err := strconv.ParseUint(address, 10, 16); err == nil {
		address = "127.0.0.1:" + address
	}
	target, _ := normalizeDesktopTarget(address)
	return target
}

func discoverLocalDesktopEntrance(baseDir string, target *url.URL) string {
	configured := discoverLocalDesktopTarget(baseDir)
	if configured == nil || target == nil || configured.Host != target.Host {
		return ""
	}
	configuration := viper.New()
	configuration.SetConfigFile(filepath.Join(baseDir, "conf.yaml"))
	if err := configuration.ReadInConfig(); err != nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(configuration.GetString("system.entrance")), "/")
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

func writeDesktopJSON(response http.ResponseWriter, status int, err error) {
	payload := map[string]any{"ok": err == nil}
	if err != nil {
		payload["error"] = err.Error()
	}
	writeDesktopResult(response, status, payload)
}

func writeDesktopResult(response http.ResponseWriter, status int, payload map[string]any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}

func newBuiltinDesktopTarget(baseDir string) (net.Listener, *url.URL, string, string, error) {
	configured := discoverLocalDesktopTarget(baseDir)
	port := "5470"
	if configured != nil {
		port = configured.Port()
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("0.0.0.0", port))
	if err != nil {
		listener, err = net.Listen("tcp4", "0.0.0.0:0")
	}
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("start built-in web service: %w", err)
	}
	actualPort := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	target, _ := url.Parse("http://" + net.JoinHostPort("127.0.0.1", actualPort))
	mobileURL := "http://" + net.JoinHostPort(desktopLANIP(), actualPort) + "/mobile"
	token, err := newDesktopToken()
	if err != nil {
		_ = listener.Close()
		return nil, nil, "", "", err
	}
	middleware.SetDesktopAccessToken(token)
	return listener, target, mobileURL, token, nil
}

func desktopLANIP() string {
	connection, err := net.Dial("udp4", "8.8.8.8:80")
	if err == nil {
		defer connection.Close()
		if address, ok := connection.LocalAddr().(*net.UDPAddr); ok && address.IP != nil {
			return address.IP.String()
		}
	}
	addresses, _ := net.InterfaceAddrs()
	for _, address := range addresses {
		ip, _, parseErr := net.ParseCIDR(address.String())
		if parseErr == nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return ip.String()
		}
	}
	return "127.0.0.1"
}
