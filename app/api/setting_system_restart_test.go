package api

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestSettingSystemRestartReturnsSchedulingFailure(t *testing.T) {
	originalRestart := settingRestartGoPanel
	settingRestartGoPanel = func() error { return errors.New("gpc unavailable") }
	t.Cleanup(func() { settingRestartGoPanel = originalRestart })

	app := fiber.New()
	app.Post("/restart/:operation", SettingSystemRestart)
	response, err := app.Test(httptest.NewRequest("POST", "/restart/panel", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "gpc unavailable") || strings.Contains(string(body), `"code":0`) {
		t.Fatalf("unexpected restart response: %s", body)
	}
}

func TestSettingSystemRestartSchedulesPanel(t *testing.T) {
	originalRestart := settingRestartGoPanel
	called := false
	settingRestartGoPanel = func() error {
		called = true
		return nil
	}
	t.Cleanup(func() { settingRestartGoPanel = originalRestart })

	app := fiber.New()
	app.Post("/restart/:operation", SettingSystemRestart)
	response, err := app.Test(httptest.NewRequest("POST", "/restart/panel", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !called || !strings.Contains(string(body), `"code":0`) {
		t.Fatalf("unexpected restart response: %s", body)
	}
}
