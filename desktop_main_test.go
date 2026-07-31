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
	listener := testListener{address: "127.0.0.1:54701"}
	app := &desktopApp{listener: listener, token: "desktop-secret"}
	next := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(response, "<html><head></head><body></body></html>")
	})
	handler := app.bootstrapMiddleware()(next)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	body := response.Body.String()
	if !strings.Contains(body, listener.Addr().String()) {
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
	initrepo.Init()
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
