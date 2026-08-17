//go:build desktop && darwin

package main

/*
#cgo CFLAGS: -fblocks
#cgo LDFLAGS: -framework Cocoa

void gopanel_statusbar_start(void);
void gopanel_statusbar_update(int attention, int running, int queued);
void gopanel_statusbar_stop(void);
*/
import "C"

import (
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	statusBarAppMu sync.RWMutex
	statusBarApp   *desktopApp
)

func startDesktopStatusBar(app *desktopApp) {
	statusBarAppMu.Lock()
	statusBarApp = app
	statusBarAppMu.Unlock()
	C.gopanel_statusbar_start()
}

func updateDesktopStatusBar(summary desktopCodeSummary) {
	C.gopanel_statusbar_update(C.int(summary.Attention), C.int(summary.Running), C.int(summary.Queued))
}

func stopDesktopStatusBar() {
	C.gopanel_statusbar_stop()
	statusBarAppMu.Lock()
	statusBarApp = nil
	statusBarAppMu.Unlock()
}

func activeStatusBarApp() *desktopApp {
	statusBarAppMu.RLock()
	defer statusBarAppMu.RUnlock()
	return statusBarApp
}

//export GoPanelOpenCodeWorkspace
func GoPanelOpenCodeWorkspace() {
	app := activeStatusBarApp()
	if app == nil || app.context == nil {
		return
	}
	wailsruntime.WindowShow(app.context)
	wailsruntime.WindowUnminimise(app.context)
	wailsruntime.WindowExecJS(app.context, `window.location.href="/code/index"`)
}

//export GoPanelShowMainWindow
func GoPanelShowMainWindow() {
	app := activeStatusBarApp()
	if app == nil || app.context == nil {
		return
	}
	wailsruntime.WindowShow(app.context)
	wailsruntime.WindowUnminimise(app.context)
}

//export GoPanelQuitApplication
func GoPanelQuitApplication() {
	app := activeStatusBarApp()
	if app == nil || app.context == nil {
		return
	}
	wailsruntime.Quit(app.context)
}
