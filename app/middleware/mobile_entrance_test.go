package middleware

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func TestMobileAPIsBypassSecurityEntrance(t *testing.T) {
	for _, path := range []string{
		"/api/mobile/health",
		"/api/mobile/pair/exchange",
		"/api/mobile/app/overview",
		"/assets/mobile.js",
	} {
		if !shouldBypassEntrance(path) {
			t.Fatalf("mobile path should bypass entrance: %s", path)
		}
	}
}

func TestMobilePagesAndManagementKeepSecurityEntrance(t *testing.T) {
	for _, path := range []string{
		"/mobile",
		"/mobile/auth",
		"/mobile/",
		"/api/mobile/management/pair/issue",
		"/api/mobile/management/devices",
		"/api/file/list",
	} {
		if shouldBypassEntrance(path) {
			t.Fatalf("management path must keep entrance protection: %s", path)
		}
	}
}

func TestMobilePageRequiresSecurityEntrance(t *testing.T) {
	server := newEntranceTestServer(t)
	response, err := server.Test(httptest.NewRequest("GET", "/mobile", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "请使用管理员配置的") {
		t.Fatalf("expected security entrance prompt, got %q", string(body))
	}
}

func TestMobileLoginRequiresSecurityEntranceCookie(t *testing.T) {
	server := newEntranceTestServer(t)
	request := httptest.NewRequest("POST", "/api/mobile/login", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusForbidden)
	}

	authorizedRequest := httptest.NewRequest("POST", "/api/mobile/login", strings.NewReader(`{}`))
	authorizedRequest.Header.Set("Content-Type", "application/json")
	authorizedRequest.AddCookie(&http.Cookie{
		Name: constant.Entrance, Value: base64.StdEncoding.EncodeToString([]byte("secure-entry")),
	})
	authorizedResponse, err := server.Test(authorizedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if authorizedResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("authorized status = %d, want %d", authorizedResponse.StatusCode, fiber.StatusOK)
	}
}

func TestMobileSecurityEntranceRedirectsToMobile(t *testing.T) {
	server := newEntranceTestServer(t)
	request := httptest.NewRequest("GET", "/secure-entry", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile")
	response, err := server.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusFound)
	}
	if location := response.Header.Get("Location"); location != "/mobile" {
		t.Fatalf("location = %q, want /mobile", location)
	}
	cookies := response.Cookies()
	if len(cookies) == 0 || cookies[0].Name != constant.Entrance {
		t.Fatalf("expected entrance cookie, got %#v", cookies)
	}

	mobileRequest := httptest.NewRequest("GET", "/mobile", nil)
	for _, cookie := range cookies {
		mobileRequest.AddCookie(cookie)
	}
	mobileResponse, err := server.Test(mobileRequest)
	if err != nil {
		t.Fatal(err)
	}
	if mobileResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("authorized mobile status = %d, want %d", mobileResponse.StatusCode, fiber.StatusOK)
	}
}

func TestMobileSecurityEntrancePreservesValidPairingCode(t *testing.T) {
	server := newEntranceTestServer(t)
	pairingCode := "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI"
	request := httptest.NewRequest("GET", "/secure-entry?mobilePairing="+pairingCode, nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 15; Mobile)")
	response, err := server.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if location := response.Header.Get("Location"); location != "/mobile/auth?code="+pairingCode {
		t.Fatalf("location = %q, want pairing authorization route", location)
	}
}

func TestMobileSecurityEntranceRejectsInvalidPairingCode(t *testing.T) {
	server := newEntranceTestServer(t)
	request := httptest.NewRequest("GET", "/secure-entry?mobilePairing=not-a-pairing-code", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 15; Mobile)")
	response, err := server.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if location := response.Header.Get("Location"); location != "/mobile" {
		t.Fatalf("location = %q, want /mobile", location)
	}
}

func TestDesktopSecurityEntranceDoesNotRedirect(t *testing.T) {
	server := newEntranceTestServer(t)
	request := httptest.NewRequest("GET", "/secure-entry", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	response, err := server.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
}

func newEntranceTestServer(t *testing.T) *fiber.App {
	t.Helper()
	previousConfig := global.CONF
	global.CONF.System.Mode = "prod"
	global.CONF.System.Entrance = "secure-entry"
	t.Cleanup(func() { global.CONF = previousConfig })

	server := fiber.New()
	server.Use(Entrance)
	server.All("/*", func(c fiber.Ctx) error {
		return c.SendString("panel")
	})
	return server
}
