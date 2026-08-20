//go:build desktop

package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	panelapp "github.com/aihop/gopanel/app"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/cmd"
	"github.com/aihop/gopanel/global"
	initconf "github.com/aihop/gopanel/init/conf"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed public/* resource/*
var desktopFS embed.FS

type desktopApp struct {
	panel              *panelapp.App
	serveErr           chan error
	context            context.Context
	initialCredentials *desktopCredentials
	baseDir            string
	gateway            *desktopGateway
	connectionStatus   *menu.MenuItem
	builtinMu          sync.Mutex
}

func main() {
	app, err := newDesktopApp()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	frontend, err := fs.Sub(desktopFS, "public")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = wails.Run(&options.App{
		Title:             "GoPanel",
		Width:             1280,
		Height:            800,
		MinWidth:          900,
		MinHeight:         600,
		HideWindowOnClose: runtime.GOOS == "darwin",
		BackgroundColour:  options.NewRGB(15, 23, 42),
		Menu:              app.applicationMenu(),
		AssetServer: &assetserver.Options{
			Assets:     frontend,
			Handler:    app.gateway,
			Middleware: app.desktopMiddleware(),
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "io.aihop.gopanel",
			OnSecondInstanceLaunch: app.showWindow,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newDesktopApp() (*desktopApp, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home directory: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".gopanel")
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create desktop data directory: %w", err)
	}
	if err := os.Setenv("GOPANEL_BASE_DIR", baseDir); err != nil {
		return nil, fmt.Errorf("set desktop data directory: %w", err)
	}
	cmd.ConfFilePath = filepath.Join(baseDir, "conf.yaml")
	global.EmbedFS = desktopFS
	gateway, err := newDesktopGateway(baseDir)
	if err != nil {
		return nil, err
	}
	return &desktopApp{
		baseDir:  baseDir,
		gateway:  gateway,
		serveErr: make(chan error, 1),
	}, nil
}

func newDesktopToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("create desktop access token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (a *desktopApp) desktopMiddleware() assetserver.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if a.gateway.handleDesktopRoute(response, request) {
				return
			}
			if !a.gateway.ready() {
				if request.URL.Path == "/" || request.URL.Path == "/index.html" {
					response.Header().Set("Content-Type", "text/html; charset=utf-8")
					response.Header().Set("Cache-Control", "no-store")
					_, _ = response.Write([]byte(a.gateway.launcherHTML()))
					return
				}
				http.Redirect(response, request, "/", http.StatusSeeOther)
				return
			}
			if request.Method != http.MethodGet || (request.URL.Path != "/" && request.URL.Path != "/index.html") {
				next.ServeHTTP(response, request)
				return
			}

			recorder := httptest.NewRecorder()
			next.ServeHTTP(recorder, request)
			for key, values := range recorder.Header() {
				response.Header()[key] = values
			}
			body := strings.Replace(recorder.Body.String(), "</head>", a.gateway.bootstrapHTML()+"</head>", 1)
			response.Header().Del("Content-Length")
			response.WriteHeader(recorder.Code)
			_, _ = response.Write([]byte(body))
		})
	}
}

func (a *desktopApp) startup(ctx context.Context) {
	a.context = ctx
	a.gateway.setBuiltinStarter(a.startBuiltin)
	if a.gateway.shouldStartBuiltin() {
		if err := a.startBuiltin(); err == nil {
			wailsruntime.WindowReload(ctx)
		}
	}
	// 切换服务器只改了服务端的代理目标；已打开页面里的 WebSocket 补丁
	// 还指着旧地址，必须主动推一份新的过去，否则表现为「面板正常、终端连不上」。
	a.gateway.setTargetChangedListener(func() { a.pushDesktopWebSocketConfig() })
	a.pushDesktopWebSocketConfig()
	go a.monitorConnectionStatus(ctx)
	a.startCodeStatusSync(ctx)
}

func (a *desktopApp) shutdown(context.Context) {
	stopDesktopStatusBar()
	middleware.SetDesktopAccessToken("")
	if a.panel == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = a.panel.Shutdown(shutdownCtx)
	select {
	case err := <-a.serveErr:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			fmt.Fprintln(os.Stderr, err)
		}
	case <-shutdownCtx.Done():
	}
}

func (a *desktopApp) startBuiltin() error {
	a.builtinMu.Lock()
	defer a.builtinMu.Unlock()
	if a.panel != nil {
		return nil
	}
	credentials, err := prepareDesktopCredentials(a.baseDir)
	if err != nil {
		return err
	}
	if credentials != nil {
		initconf.InitInstall.User = credentials.Email
		initconf.InitInstall.Password = credentials.Password
	}
	listener, target, mobileURL, token, err := newBuiltinDesktopTarget(a.baseDir)
	if err != nil {
		return err
	}
	a.panel = &panelapp.App{}
	a.initialCredentials = credentials
	if err := a.panel.Init(); err != nil {
		a.panel = nil
		_ = listener.Close()
		return err
	}
	a.gateway.useBuiltin(target, mobileURL, token)
	if credentials != nil && a.context != nil {
		go a.showInitialCredentials(a.context)
	}
	go func() { a.serveErr <- a.panel.Serve(listener) }()
	return nil
}

func (a *desktopApp) showWindow(options.SecondInstanceData) {
	if a.context != nil {
		wailsruntime.WindowShow(a.context)
	}
}

func (a *desktopApp) applicationMenu() *menu.Menu {
	applicationMenu := menu.NewMenuFromItems(menu.AppMenu())
	connectionMenu := applicationMenu.AddSubmenu("连接")
	a.connectionStatus = connectionMenu.AddText("当前：检测中", nil, nil).Disable()
	connectionMenu.AddSeparator()
	connectionMenu.AddText("切换服务器…", nil, func(*menu.CallbackData) {
		if a.context != nil {
			wailsruntime.WindowExecJS(a.context, `window.location.href="/__desktop"`)
		}
	})
	connectionMenu.AddText("重新加载", nil, func(*menu.CallbackData) {
		if a.context != nil {
			wailsruntime.WindowReload(a.context)
		}
	})
	applicationMenu.Append(menu.EditMenu())
	applicationMenu.Append(menu.WindowMenu())
	return applicationMenu
}

func (a *desktopApp) monitorConnectionStatus(ctx context.Context) {
	a.updateConnectionStatus(ctx, "检测中")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		target, mode, desktopToken := a.desktopConnectionTarget()
		status := "离线"
		if desktopTargetHealthyWithToken(target, desktopToken) {
			status = "在线"
		}
		a.updateConnectionStatusWithTarget(ctx, target, mode, status)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *desktopApp) desktopConnectionTarget() (*url.URL, string, string) {
	a.gateway.RLock()
	defer a.gateway.RUnlock()
	if a.gateway.target == nil {
		return nil, a.gateway.config.Mode, a.gateway.desktopToken
	}
	target := *a.gateway.target
	return &target, a.gateway.config.Mode, a.gateway.desktopToken
}

func (a *desktopApp) updateConnectionStatus(ctx context.Context, status string) {
	target, mode, _ := a.desktopConnectionTarget()
	a.updateConnectionStatusWithTarget(ctx, target, mode, status)
}

func (a *desktopApp) updateConnectionStatusWithTarget(ctx context.Context, target *url.URL, mode, status string) {
	title, menuLabel := desktopConnectionLabels(target, mode, status)
	wailsruntime.WindowSetTitle(ctx, title)
	if a.connectionStatus != nil {
		a.connectionStatus.SetLabel(menuLabel)
		wailsruntime.MenuUpdateApplicationMenu(ctx)
	}
}

func desktopConnectionLabels(target *url.URL, mode, status string) (string, string) {
	if target == nil {
		detail := fmt.Sprintf("未连接 · %s", status)
		return "GoPanel — " + detail, "当前：" + detail
	}
	modeLabel := "远程服务器"
	if mode == "builtin" {
		modeLabel = "内置服务"
	}
	detail := fmt.Sprintf("%s · %s · %s", modeLabel, target.Host, status)
	return "GoPanel — " + detail, "当前：" + detail
}

// pushDesktopWebSocketConfig 把当前 WebSocket 目标推给已经打开的页面。
// 连接目标可能在窗口就绪之前就变化（启动时的健康检查会调 setTarget），
// 所以 context 还没准备好时直接跳过——窗口加载时的注入会带上正确的值。
func (a *desktopApp) pushDesktopWebSocketConfig() {
	if a.context == nil || a.gateway == nil {
		return
	}
	wailsruntime.WindowExecJS(a.context, a.gateway.webSocketConfigScript())
}
