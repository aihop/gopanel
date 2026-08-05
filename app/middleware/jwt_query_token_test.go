package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// SSE 接口漏加后缀 = 连接静默 401，前端只看到 onerror，极难排查。
// /host/disk/scan/stream 就是这么坏掉的，这里锁住允许名单。
func TestIsQueryTokenAllowed(t *testing.T) {
	allowed := []string{
		"/api/pipeline/logs",
		"/api/file/compress/logs",
		"/api/file/wget/logs",
		"/api/apps/install/nginx/logs",
		"/api/apps/installed/mysql/runtime/logs",
		"/api/setting/system/upgrade/logs",
		"/api/agent/ensure/logs",
		"/api/backup/logs",
		"/api/ssl/1/logs",
		"/api/container/logs",
		"/api/container/compose/logs",
		"/api/node-proxy-ws/1/container/logs",
		"/api/code/terminal",
		"/api/mobile/app/terminal",
		"/api/file/ws",
		"/api/process/ws",
		"/api/host/terminal/sessions/1/ws",
		"/api/code/project-terminal/1/ws",
		"/api/host/disk/scan/stream",
		"/api/container/exec",
		"/api/node-proxy-ws/1/container/exec",
	}
	for _, p := range allowed {
		if !isQueryTokenAllowed(fiber.MethodGet, p) {
			t.Errorf("%s 应允许用 ?token= 鉴权（EventSource 无法设置请求头）", p)
		}
	}

	// 普通接口不能开这个口子：URL 里的 token 会进访问日志和 Referer
	denied := []string{
		"/api/host/disk/clean",
		"/api/file/batch/del",
		"/api/host/disk/scan",
		"/api/database/manager/exec",
		"/api/arbitrary/exec",
	}
	for _, p := range denied {
		if isQueryTokenAllowed(fiber.MethodGet, p) {
			t.Errorf("%s 不应允许 ?token= 鉴权", p)
		}
	}
	if isQueryTokenAllowed(fiber.MethodPost, "/api/container/clean/logs") {
		t.Fatal("普通 POST 接口不应允许 ?token= 鉴权")
	}
}

func TestGetUserAccessTokenRestrictsQueryParameter(t *testing.T) {
	app := fiber.New()
	app.Get("/api/file/download", func(c fiber.Ctx) error {
		return c.SendString(getUserAccessToken(c))
	})
	app.Get("/api/process/ws", func(c fiber.Ctx) error {
		return c.SendString(getUserAccessToken(c))
	})
	app.Get("/api/container/exec", func(c fiber.Ctx) error {
		return c.SendString(getUserAccessToken(c))
	})
	app.Get("/api/node-proxy-ws/:id/container/exec", func(c fiber.Ctx) error {
		return c.SendString(getUserAccessToken(c))
	})
	app.Post("/api/container/clean/logs", func(c fiber.Ctx) error {
		return c.SendString(getUserAccessToken(c))
	})

	assertToken := func(path, want string) {
		t.Helper()
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != want {
			t.Fatalf("%s token = %q, want %q", path, body, want)
		}
	}

	assertToken("/api/file/download?token=query-secret", "")
	assertToken("/api/process/ws?token=query-secret", "query-secret")
	assertToken("/api/container/exec?token=query-secret", "query-secret")
	assertToken("/api/node-proxy-ws/1/container/exec?token=query-secret", "query-secret")

	resp, err := app.Test(httptest.NewRequest("POST", "/api/container/clean/logs?token=query-secret", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Fatalf("POST query token was accepted: %q", body)
	}
}
