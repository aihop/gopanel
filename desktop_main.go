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
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	panelapp "github.com/aihop/gopanel/app"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/cmd"
	"github.com/aihop/gopanel/global"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed public/* resource/*
var desktopFS embed.FS

type desktopApp struct {
	panel    *panelapp.App
	listener net.Listener
	serveErr chan error
	context  context.Context
	token    string
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
		Width:            1440,
		Height:           900,
		MinWidth:         1024,
		MinHeight:        700,
		BackgroundColour: options.NewRGB(15, 23, 42),
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets:     frontend,
			Handler:    app.proxy(),
			Middleware: app.bootstrapMiddleware(),
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
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config directory: %w", err)
	}
	baseDir := filepath.Join(configDir, "GoPanel")
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create desktop data directory: %w", err)
	}
	if err := os.Setenv("GOPANEL_BASE_DIR", baseDir); err != nil {
		return nil, fmt.Errorf("set desktop data directory: %w", err)
	}
	cmd.ConfFilePath = filepath.Join(baseDir, "conf.yaml")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}

	global.EmbedFS = desktopFS
	token, err := newDesktopToken()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	middleware.SetDesktopAccessToken(token)
	return &desktopApp{panel: &panelapp.App{}, listener: listener, serveErr: make(chan error, 1), token: token}, nil
}

func newDesktopToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("create desktop access token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (a *desktopApp) proxy() http.Handler {
	target := &url.URL{Scheme: "http", Host: a.listener.Addr().String()}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
		originalDirector(request)
		request.Header.Set("X-GoPanel-Desktop-Token", a.token)
	}
	proxy.ErrorHandler = func(response http.ResponseWriter, _ *http.Request, err error) {
		http.Error(response, err.Error(), http.StatusBadGateway)
	}
	return proxy
}

func (a *desktopApp) bootstrapMiddleware() assetserver.Middleware {
	script := "<script>" + websocketBootstrap(a.listener.Addr().String(), a.token) + "</script>"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodGet || (request.URL.Path != "/" && request.URL.Path != "/index.html") {
				next.ServeHTTP(response, request)
				return
			}

			recorder := httptest.NewRecorder()
			next.ServeHTTP(recorder, request)
			for key, values := range recorder.Header() {
				response.Header()[key] = values
			}
			body := strings.Replace(recorder.Body.String(), "</head>", script+"</head>", 1)
			response.Header().Del("Content-Length")
			response.WriteHeader(recorder.Code)
			_, _ = response.Write([]byte(body))
		})
	}
}

func (a *desktopApp) startup(ctx context.Context) {
	a.context = ctx
	a.panel.Init()
	go func() {
		a.serveErr <- a.panel.Serve(a.listener)
	}()
}

func (a *desktopApp) shutdown(context.Context) {
	middleware.SetDesktopAccessToken("")
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

func (a *desktopApp) showWindow(options.SecondInstanceData) {
	if a.context != nil {
		wailsruntime.WindowShow(a.context)
	}
}

func websocketBootstrap(address, token string) string {
	host := strings.ReplaceAll(address, "`", "")
	desktopToken := strings.ReplaceAll(token, "`", "")
	return fmt.Sprintf(`(() => {
  const NativeWebSocket = window.WebSocket;
  window.WebSocket = class extends NativeWebSocket {
    constructor(url, protocols) {
      const parsed = new URL(url, window.location.href);
      if (parsed.protocol === "wails:" || parsed.hostname === "wails" || parsed.hostname === "wails.localhost" || parsed.hostname.endsWith(".wails")) {
		parsed.protocol = "ws:";
		parsed.host = %q;
		parsed.searchParams.set("desktop_token", %q);
	  }
      super(parsed.toString(), protocols);
    }
  };
})();`, host, desktopToken)
}
