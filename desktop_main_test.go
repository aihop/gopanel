//go:build desktop

package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
