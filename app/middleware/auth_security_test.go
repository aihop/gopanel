package middleware

import (
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func TestXGetAuthDoesNotReadQueryParameter(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString(XGetAuth(c)) })

	resp, err := app.Test(httptest.NewRequest("GET", "/test?auth=leaked-token", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Fatalf("query auth value was accepted: %q", body)
	}
}

func TestApiKeyTimestampRequiresPositiveWindow(t *testing.T) {
	old := global.CONF.System.ApiKeyValidityTime
	t.Cleanup(func() { global.CONF.System.ApiKeyValidityTime = old })

	global.CONF.System.ApiKeyValidityTime = "5"
	if !isValidTimestamp(strconv.FormatInt(time.Now().Unix(), 10)) {
		t.Fatal("current timestamp should be valid")
	}
	global.CONF.System.ApiKeyValidityTime = "0"
	if isValidTimestamp(strconv.FormatInt(time.Now().Unix(), 10)) {
		t.Fatal("zero validity window must not disable replay protection")
	}
}
