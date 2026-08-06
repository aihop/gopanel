package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/apisign"
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

func TestAPIKeyV2BindsRequestAndRejectsReplay(t *testing.T) {
	oldKey := global.CONF.System.ApiKey
	oldValidity := global.CONF.System.ApiKeyValidityTime
	t.Cleanup(func() {
		global.CONF.System.ApiKey = oldKey
		global.CONF.System.ApiKeyValidityTime = oldValidity
	})
	global.CONF.System.ApiKey = "test-secret"
	global.CONF.System.ApiKeyValidityTime = "5"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body := []byte(`{"operation":"delete"}`)

	newRequest := func(method, nonce, signature string, requestBody []byte) *http.Request {
		request := httptest.NewRequest(method, "/signed?target=demo", bytes.NewReader(requestBody))
		request.Header.Set("apiKey", signature)
		request.Header.Set("timestamp", timestamp)
		request.Header.Set("nonce", nonce)
		request.Header.Set("signatureVersion", "v2")
		return request
	}
	app := fiber.New()
	handler := func(c fiber.Ctx) error {
		return verifyAPIKey(c, c.Get("apiKey"), c.Get("timestamp"))
	}
	app.Post("/signed", handler)
	app.Put("/signed", handler)

	validNonce := "valid-request"
	validSign := apisign.Sign(global.CONF.System.ApiKey, timestamp, validNonce, "POST", "/signed", "target=demo", body)
	response, err := app.Test(newRequest("POST", validNonce, validSign, body))
	if err != nil || response.StatusCode != 200 {
		t.Fatalf("valid v2 request rejected: status=%v err=%v", response.StatusCode, err)
	}
	response, err = app.Test(newRequest("POST", validNonce, validSign, body))
	if err != nil || response.StatusCode == 200 {
		t.Fatalf("replayed nonce accepted: status=%v err=%v", response.StatusCode, err)
	}

	bodyNonce := "changed-body"
	bodySign := apisign.Sign(global.CONF.System.ApiKey, timestamp, bodyNonce, "POST", "/signed", "target=demo", body)
	response, err = app.Test(newRequest("POST", bodyNonce, bodySign, []byte(`{"operation":"stop"}`)))
	if err != nil || response.StatusCode == 200 {
		t.Fatalf("modified body accepted: status=%v err=%v", response.StatusCode, err)
	}

	methodNonce := "changed-method"
	methodSign := apisign.Sign(global.CONF.System.ApiKey, timestamp, methodNonce, "POST", "/signed", "target=demo", body)
	response, err = app.Test(newRequest("PUT", methodNonce, methodSign, body))
	if err != nil || response.StatusCode == 200 {
		t.Fatalf("modified method accepted: status=%v err=%v", response.StatusCode, err)
	}
}

func TestLegacyAPIKeyRejectsWriteRequests(t *testing.T) {
	oldKey := global.CONF.System.ApiKey
	t.Cleanup(func() { global.CONF.System.ApiKey = oldKey })
	global.CONF.System.ApiKey = "test-secret"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	legacyKey := GenerateMD5("gopanel_" + global.CONF.System.ApiKey + "_" + timestamp)
	app := fiber.New()
	app.Post("/legacy", func(c fiber.Ctx) error { return verifyAPIKey(c, legacyKey, timestamp) })

	response, err := app.Test(httptest.NewRequest("POST", "/legacy", nil))
	if err != nil || response.StatusCode == 200 {
		t.Fatalf("legacy signature accepted for write request: status=%v err=%v", response.StatusCode, err)
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
