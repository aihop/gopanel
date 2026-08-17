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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/middleware"
	"github.com/spf13/viper"
)

const desktopSettingsPath = "/__desktop"

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
	// targetChanged 在连接目标变化后触发，用来把新的 WebSocket 目标推给前端。
	// 注入脚本里的地址是 HTML 生成那一刻的快照，运行时切换服务器不会自动更新——
	// HTTP 走服务端代理感知不到这点，只有 WebSocket 会一直连向旧地址。
	targetChanged func()
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
				name := target.Host
				if saved, ok := desktopSavedServer(&gateway.config, target); ok {
					name = saved.Name
				}
				rememberDesktopServer(&gateway.config, desktopServerConfig{Name: name, URL: target.String(), Entrance: entrance})
				gateway.setTarget(target, "", desktopMobileURL(target), entrance)
				_ = gateway.saveConfig()
			}
		}
	}
	if gateway.target == nil && config.Mode == "" {
		if target := discoverLocalDesktopTarget(baseDir); target != nil && desktopTargetHealthy(target) {
			entrance := discoverLocalDesktopEntrance(baseDir, target)
			if desktopTargetAccessError(target, entrance) == nil {
				gateway.config.Mode = "remote"
				gateway.config.URL = target.String()
				gateway.config.Entrance = entrance
				rememberDesktopServer(&gateway.config, desktopServerConfig{Name: target.Host, URL: target.String(), Entrance: entrance})
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
	gateway.config.Mode = "builtin"
	gateway.config.URL = ""
	gateway.config.Entrance = ""
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
	listener := gateway.targetChanged
	gateway.Unlock()
	if listener != nil {
		listener()
	}
}

func (gateway *desktopGateway) setTargetChangedListener(listener func()) {
	gateway.Lock()
	gateway.targetChanged = listener
	gateway.Unlock()
}

// desktopWebSocketConfigScript 生成一段把当前 WebSocket 目标写进全局的 JS。
// 注入 HTML 时和运行时切换服务器后都用它，两处共用同一份拼装逻辑，
// 免得改了一处忘了另一处——那种漏改的症状正是「HTTP 正常、终端连不上」。
func (gateway *desktopGateway) webSocketConfigScript() string {
	gateway.RLock()
	target := gateway.target
	token := gateway.desktopToken
	entrance := gateway.entrance
	gateway.RUnlock()
	if target == nil {
		return "window.__GOPANEL_DESKTOP_WS__=null;"
	}
	scheme := "ws:"
	if target.Scheme == "https" {
		scheme = "wss:"
	}
	encodedEntrance := ""
	if entrance != "" {
		encodedEntrance = base64.StdEncoding.EncodeToString([]byte(entrance))
	}
	return fmt.Sprintf("window.__GOPANEL_DESKTOP_WS__={scheme:%s,host:%s,token:%s,entrance:%s};",
		strconv.Quote(scheme), strconv.Quote(target.Host),
		strconv.Quote(token), strconv.Quote(encodedEntrance))
}

func (gateway *desktopGateway) handleDesktopRoute(response http.ResponseWriter, request *http.Request) bool {
	if request.URL.Path != desktopSettingsPath && request.URL.Path != desktopSettingsPath+"/state" && request.URL.Path != desktopSettingsPath+"/connect" && request.URL.Path != desktopSettingsPath+"/builtin" && request.URL.Path != desktopSettingsPath+"/delete" {
		return false
	}
	switch request.URL.Path {
	case desktopSettingsPath:
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Cache-Control", "no-store")
		_, _ = response.Write([]byte(gateway.launcherHTML()))
	case desktopSettingsPath + "/state":
		gateway.handleConnectionState(response, request)
	case desktopSettingsPath + "/connect":
		gateway.handleConnect(response, request)
	case desktopSettingsPath + "/builtin":
		gateway.handleBuiltin(response, request)
	case desktopSettingsPath + "/delete":
		gateway.handleDeleteServer(response, request)
	}
	return true
}

func (gateway *desktopGateway) bootstrapHTML() string {
	gateway.RLock()
	target := gateway.target
	mobileURL := gateway.mobileURL
	gateway.RUnlock()
	if target == nil {
		return ""
	}
	// 补丁在构造每一条 WebSocket 时才去读 __GOPANEL_DESKTOP_WS__，
	// 而不是把地址编译进闭包：切换服务器后网关会推一份新的全局进来，
	// 已经打开的页面无需重新加载也能连对地方。
	return fmt.Sprintf(`<style>*{scrollbar-width:none!important}*::-webkit-scrollbar{display:none!important;width:0!important;height:0!important}</style><script>(()=>{window.__GOPANEL_DESKTOP_MOBILE_URL__=%s;%sconst N=window.WebSocket;window.WebSocket=class extends N{constructor(u,p){const c=window.__GOPANEL_DESKTOP_WS__;const x=new URL(u,window.location.href);if(c&&c.host&&(x.protocol==="wails:"||x.hostname==="wails"||x.hostname==="wails.localhost"||x.hostname.endsWith(".wails"))){x.protocol=c.scheme;x.host=c.host;if(c.token)x.searchParams.set("desktop_token",c.token);if(c.entrance)x.searchParams.set("entrance",c.entrance)}super(x.toString(),p)}}})();</script>`,
		strconv.Quote(mobileURL), gateway.webSocketConfigScript())
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
	return desktopTargetHealthyWithToken(target, "")
}

func desktopTargetHealthyWithToken(target *url.URL, desktopToken string) bool {
	if target == nil {
		return false
	}
	healthURL := *target
	healthURL.Path = "/health"
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	request, err := http.NewRequest(http.MethodGet, healthURL.String(), nil)
	if err != nil {
		return false
	}
	if desktopToken != "" {
		request.Header.Set("X-GoPanel-Desktop-Token", desktopToken)
	}
	response, err := client.Do(request)
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
