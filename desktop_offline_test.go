//go:build desktop

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type desktopRoundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip desktopRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestDesktopGatewayUsesFriendlyOfflinePage(t *testing.T) {
	target, err := normalizeDesktopTarget("http://127.0.0.1:15470")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &desktopGateway{}
	gateway.setTarget(target, "", "", "")
	gateway.proxy.Transport = desktopRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp 127.0.0.1:15470: connect: connection refused")
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), "GoPanel 服务暂未启动") {
		t.Fatalf("gateway did not render friendly offline page: %d %s", response.Code, response.Body.String())
	}
}

func TestDesktopConnectionErrorShowsFriendlyOfflinePage(t *testing.T) {
	target, err := normalizeDesktopTarget("http://127.0.0.1:15470")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	writeDesktopConnectionError(response, request, target, errors.New("dial tcp 127.0.0.1:15470: connect: connection refused"))

	body := response.Body.String()
	for _, expected := range []string{"GoPanel 服务暂未启动", "立即重试", "打开连接中心", "5 秒后自动检测", "查看技术详情", target.String()} {
		if !strings.Contains(body, expected) {
			t.Fatalf("offline page missing %q: %s", expected, body)
		}
	}
	if response.Code != http.StatusBadGateway || !strings.Contains(response.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("unexpected offline response: %d %q", response.Code, response.Header().Get("Content-Type"))
	}
}

func TestDesktopConnectionErrorKeepsAPIFailureStructured(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/code/desktop-summary", nil)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	writeDesktopConnectionError(response, request, nil, errors.New("dial tcp 127.0.0.1:15470: connect: connection refused"))

	body := response.Body.String()
	if response.Code != http.StatusBadGateway || !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected API response: %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, desktopConnectionUnavailableMessage) || strings.Contains(body, "dial tcp") {
		t.Fatalf("API response should be friendly and hide transport details: %s", body)
	}
}

func TestDesktopConnectionFailureTextCoversCommonNetworkErrors(t *testing.T) {
	tests := []struct {
		message string
		title   string
	}{
		{message: "i/o timeout", title: "连接服务器超时"},
		{message: "lookup panel.example.com: no such host", title: "找不到这台服务器"},
		{message: "tls: failed to verify certificate", title: "安全连接验证失败"},
		{message: "unexpected EOF", title: "暂时无法连接到 GoPanel"},
	}
	for _, test := range tests {
		if actual := desktopConnectionFailureText(errors.New(test.message)).Title; actual != test.title {
			t.Fatalf("failure title for %q = %q, want %q", test.message, actual, test.title)
		}
	}
}
