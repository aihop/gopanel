//go:build desktop

package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	initconf "github.com/aihop/gopanel/init/conf"
	initrepo "github.com/aihop/gopanel/init/repo"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type testAddr string

func (address testAddr) Network() string { return "tcp" }
func (address testAddr) String() string  { return string(address) }

type testListener struct{ address testAddr }

func (listener testListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (listener testListener) Close() error              { return nil }
func (listener testListener) Addr() net.Addr            { return listener.address }

func TestBootstrapMiddlewareInjectsWebSocketTarget(t *testing.T) {
	target, err := normalizeDesktopTarget("http://127.0.0.1:54701")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &desktopGateway{}
	gateway.setTarget(target, "desktop-secret", "http://192.168.1.8:54701/mobile", "safe-entry")
	app := &desktopApp{gateway: gateway}
	next := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(response, "<html><head></head><body></body></html>")
	})
	handler := app.desktopMiddleware()(next)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	body := response.Body.String()
	if !strings.Contains(body, target.Host) {
		t.Fatalf("expected loopback address in bootstrap: %s", body)
	}
	if !strings.Contains(body, "window.WebSocket") {
		t.Fatalf("expected WebSocket bootstrap: %s", body)
	}
	if !strings.Contains(body, "desktop-secret") {
		t.Fatalf("expected desktop token in bootstrap: %s", body)
	}
	if !strings.Contains(body, "wails.localhost") {
		t.Fatalf("expected Windows WebView host support: %s", body)
	}
	if !strings.Contains(body, "gopanel-desktop-context-menu") || !strings.Contains(body, "刷新页面") || !strings.Contains(body, "返回首页") || !strings.Contains(body, "打开连接中心") {
		t.Fatalf("expected desktop context menu bootstrap: %s", body)
	}
}

func TestDesktopMiddlewareShowsLauncherWhenDisconnected(t *testing.T) {
	app := &desktopApp{gateway: &desktopGateway{baseDir: t.TempDir()}}
	handler := app.desktopMiddleware()(http.NotFoundHandler())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "服务器连接中心") {
		t.Fatalf("unexpected launcher response: %d %s", response.Code, response.Body.String())
	}
}

func TestNormalizeDesktopTarget(t *testing.T) {
	target, err := normalizeDesktopTarget("127.0.0.1:15470/mobile")
	if err != nil {
		t.Fatal(err)
	}
	if target.String() != "http://127.0.0.1:15470" {
		t.Fatalf("normalizeDesktopTarget() = %q", target.String())
	}
}

func TestDesktopConnectionLabels(t *testing.T) {
	target, err := normalizeDesktopTarget("https://panel.example.com:8443")
	if err != nil {
		t.Fatal(err)
	}
	title, menuLabel := desktopConnectionLabels(target, "remote", "在线")
	if title != "GoPanel — 远程服务器 · panel.example.com:8443 · 在线" {
		t.Fatalf("unexpected window title: %q", title)
	}
	if menuLabel != "当前：远程服务器 · panel.example.com:8443 · 在线" {
		t.Fatalf("unexpected connection menu label: %q", menuLabel)
	}

	title, menuLabel = desktopConnectionLabels(nil, "builtin", "离线")
	if title != "GoPanel — 未连接 · 离线" || menuLabel != "当前：未连接 · 离线" {
		t.Fatalf("unexpected disconnected labels: %q, %q", title, menuLabel)
	}
}

func TestNormalizeDesktopConnectionExtractsEntrancePath(t *testing.T) {
	target, entrance, err := normalizeDesktopConnection("http://127.0.0.1:15470/private-entry", "")
	if err != nil {
		t.Fatal(err)
	}
	if target.String() != "http://127.0.0.1:15470" || entrance != "private-entry" {
		t.Fatalf("normalizeDesktopConnection() = %q, %q", target.String(), entrance)
	}
}

func TestDesktopMobileURLReplacesLoopbackHost(t *testing.T) {
	target, err := normalizeDesktopTarget("http://127.0.0.1:15470")
	if err != nil {
		t.Fatal(err)
	}
	mobileURL := desktopMobileURLWithIP(target, "192.168.1.8")
	if mobileURL != "http://192.168.1.8:15470/mobile" {
		t.Fatalf("desktopMobileURL() = %q", mobileURL)
	}
}

func TestDesktopTargetHealthRequiresGoPanel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/health" {
			http.NotFound(response, request)
			return
		}
		_, _ = io.WriteString(response, `{"code":0,"data":{"appName":"GoPanel","appBrand":"GoPanel"}}`)
	}))
	defer server.Close()
	target, err := normalizeDesktopTarget(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !desktopTargetHealthy(target) {
		t.Fatal("expected GoPanel health endpoint to be accepted")
	}
}

func TestNewDesktopGatewayMigratesLocalSecurityEntrance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			_, _ = io.WriteString(response, `{"code":0,"data":{"appName":"GoPanel"}}`)
			return
		}
		if request.Header.Get("EntranceCode") != "bG9jYWwtZW50cnk=" {
			response.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(response, "ok")
	}))
	defer server.Close()
	target, err := normalizeDesktopTarget(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	baseDir := t.TempDir()
	config := "system:\n  port: " + target.Host + "\n  entrance: local-entry\n"
	if err := os.WriteFile(filepath.Join(baseDir, "conf.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "desktop.json"), []byte(`{"mode":"remote","url":"`+target.String()+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	gateway, err := newDesktopGateway(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if !gateway.ready() || gateway.config.Entrance != "local-entry" {
		t.Fatalf("expected migrated local entrance, got %#v", gateway.config)
	}
	saved, err := loadDesktopConnection(baseDir)
	if err != nil || saved.Entrance != "local-entry" {
		t.Fatalf("expected saved local entrance, got %#v, %v", saved, err)
	}
}

func TestDesktopGatewayProxiesRemoteService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(response, request.URL.Path+" "+request.Header.Get("EntranceCode"))
	}))
	defer server.Close()
	target, err := normalizeDesktopTarget(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	gateway := &desktopGateway{}
	gateway.setTarget(target, "", desktopMobileURL(target), "private-entry")
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/test", nil))
	if response.Code != http.StatusOK || response.Body.String() != "/api/test cHJpdmF0ZS1lbnRyeQ==" {
		t.Fatalf("unexpected proxy response: %d %q", response.Code, response.Body.String())
	}
}

func TestDesktopTargetAccessRequiresCorrectEntrance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("EntranceCode") != "cHJpdmF0ZS1lbnRyeQ==" {
			response.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(response, "ok")
	}))
	defer server.Close()
	target, err := normalizeDesktopTarget(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := desktopTargetAccessError(target, ""); err == nil || !strings.Contains(err.Error(), "填写") {
		t.Fatalf("expected missing entrance error, got %v", err)
	}
	if err := desktopTargetAccessError(target, "wrong"); err == nil || !strings.Contains(err.Error(), "不正确") {
		t.Fatalf("expected invalid entrance error, got %v", err)
	}
	if err := desktopTargetAccessError(target, "private-entry"); err != nil {
		t.Fatalf("expected entrance to be accepted: %v", err)
	}
}

func TestPrepareDesktopCredentialsForNewDatabase(t *testing.T) {
	credentials, err := prepareDesktopCredentials(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if credentials == nil || credentials.Email != desktopAdminEmail || len(credentials.Password) < 20 {
		t.Fatalf("unexpected initial desktop credentials: %#v", credentials)
	}
	if strings.Contains(credentials.Password, "=") {
		t.Fatalf("desktop password should be URL-safe without padding: %q", credentials.Password)
	}
}

func TestPrepareDesktopCredentialsKeepsExistingUserDatabase(t *testing.T) {
	baseDir := t.TempDir()
	databasePath := filepath.Join(baseDir, "db", "gopanel.db")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	credentials, err := prepareDesktopCredentials(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if credentials != nil {
		t.Fatalf("existing desktop database should not create new credentials: %#v", credentials)
	}
}

func TestDesktopFirstRunMigratesDatabase(t *testing.T) {
	baseDir := t.TempDir()
	credentials, err := prepareDesktopCredentials(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(sqlite.Open(filepath.Join(baseDir, "gopanel.db")), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	global.MonitorDB = nil
	initconf.InitInstall.User = credentials.Email
	initconf.InitInstall.Password = credentials.Password
	if err := initrepo.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDatabase, dbErr := database.DB(); dbErr == nil {
			_ = sqlDatabase.Close()
		}
	})

	for _, table := range []string{"user", "cronjob", "node", "ai_instructions"} {
		if !global.DB.Migrator().HasTable(table) {
			t.Fatalf("desktop first run did not create %s table", table)
		}
	}
}

// 切换服务器只改服务端的代理目标，已打开页面里的 WebSocket 补丁还指着旧地址：
// HTTP 走代理感知不到，只有终端连不上。配置必须跟着目标走。
func TestDesktopWebSocketConfigFollowsTargetChange(t *testing.T) {
	first, err := normalizeDesktopTarget("http://127.0.0.1:54701")
	if err != nil {
		t.Fatal(err)
	}
	second, err := normalizeDesktopTarget("https://panel.example.com")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &desktopGateway{}
	gateway.setTarget(first, "token-a", "http://127.0.0.1:54701/mobile", "entry-a")
	before := gateway.webSocketConfigScript()
	if !strings.Contains(before, first.Host) || !strings.Contains(before, `"ws:"`) {
		t.Fatalf("首个目标应产出 ws 与对应主机：%s", before)
	}

	gateway.setTarget(second, "token-b", "https://panel.example.com/mobile", "entry-b")
	after := gateway.webSocketConfigScript()
	if strings.Contains(after, first.Host) {
		t.Fatalf("切换后不该还带着旧主机：%s", after)
	}
	if !strings.Contains(after, second.Host) {
		t.Fatalf("切换后应指向新主机：%s", after)
	}
	// https 目标必须用 wss，否则浏览器会以混合内容为由直接拒绝。
	if !strings.Contains(after, `"wss:"`) {
		t.Fatalf("https 目标应使用 wss：%s", after)
	}
	if !strings.Contains(after, "token-b") {
		t.Fatalf("切换后应带上新令牌：%s", after)
	}
}

// 目标变化必须通知出去，否则已打开的页面永远不知道该换地址。
func TestDesktopGatewayNotifiesOnTargetChange(t *testing.T) {
	target, err := normalizeDesktopTarget("http://127.0.0.1:54701")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &desktopGateway{}
	notified := 0
	gateway.setTargetChangedListener(func() { notified++ })
	gateway.setTarget(target, "", "", "")
	if notified != 1 {
		t.Fatalf("设置目标应触发一次通知，实际 %d", notified)
	}
	gateway.setTarget(target, "", "", "")
	if notified != 2 {
		t.Fatalf("每次设置目标都应通知，实际 %d", notified)
	}
}

// 还没连上任何服务时，补丁要能识别出「没有目标」而不是拿空主机去连。
func TestDesktopWebSocketConfigIsNullWithoutTarget(t *testing.T) {
	gateway := &desktopGateway{}
	if script := gateway.webSocketConfigScript(); !strings.Contains(script, "null") {
		t.Fatalf("没有目标时应产出 null：%s", script)
	}
}
