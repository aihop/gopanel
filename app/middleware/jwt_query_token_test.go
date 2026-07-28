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
		"/api/ai/terminal",
		"/api/host/disk/scan/stream",
	}
	for _, p := range allowed {
		if !isQueryTokenAllowed(fiber.MethodGet, p) {
			t.Errorf("%s 应允许用 ?token= 鉴权（EventSource 无法设置请求头）", p)
		}
	}

	// 普通接口不能开这个口子：URL 里的 token 会进访问日志和 Referer
	denied := []string{"/api/host/disk/clean", "/api/file/batch/del", "/api/host/disk/scan"}
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
