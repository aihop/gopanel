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
	"os"
	"path/filepath"
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
		Title:            "GoPanel",
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: options.NewRGB(15, 23, 42),
		Menu:             app.applicationMenu(),
		AssetServer: &assetserver.Options{
			Assets:     frontend,
			Handler:    app.gateway,
			Middleware: app.desktopMiddleware(),
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
}

func (a *desktopApp) shutdown(context.Context) {
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
	a.panel.Init()
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
	connectionMenu.AddText("连接设置…", nil, func(*menu.CallbackData) {
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
