package middleware

import (
	"bytes"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func TestOperationLogRecordsUnmappedWriteWithIdentity(t *testing.T) {
	database := setupOperationLogDatabase(t)
	app := fiber.New()
	app.Use(OperationLog())
	app.Post("/api/unmapped/action", func(c fiber.Ctx) error {
		c.Locals(constant.AppAuthName, &token.CustomClaims{UserId: 42})
		c.Locals(constant.AuthMethodName, constant.AuthMethodMobile)
		return c.JSON(map[string]any{"code": 0})
	})

	response, err := app.Test(httptest.NewRequest("POST", "/api/unmapped/action", bytes.NewBufferString(`{"password":"must-not-be-logged"}`)))
	if err != nil || response.StatusCode != 200 {
		t.Fatalf("request failed: status=%v err=%v", response.StatusCode, err)
	}
	var record model.OperationLog
	if err := database.First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.DetailZH != "POST /api/unmapped/action" || record.UserID != 42 || record.AuthMethod != constant.AuthMethodMobile || record.Source != constant.AuthMethodMobile {
		t.Fatalf("unexpected operation log: %#v", record)
	}
	if bytes.Contains([]byte(record.DetailZH+record.DetailEN+record.Message), []byte("must-not-be-logged")) {
		t.Fatal("unmapped operation log captured request body")
	}
}

func TestOperationLogKeepsMappedDetailAndSkipsReads(t *testing.T) {
	database := setupOperationLogDatabase(t)
	app := fiber.New()
	app.Use(OperationLog())
	app.Post("/api/container/clean/log", func(c fiber.Ctx) error { return c.JSON(map[string]any{"code": 0}) })
	app.Get("/api/read-only", func(c fiber.Ctx) error { return c.SendStatus(200) })

	request := httptest.NewRequest("POST", "/api/container/clean/log", bytes.NewBufferString(`{"name":"demo"}`))
	request.Header.Set("Content-Type", "application/json")
	if response, err := app.Test(request); err != nil || response.StatusCode != 200 {
		t.Fatalf("mapped request failed: status=%v err=%v", response.StatusCode, err)
	}
	if response, err := app.Test(httptest.NewRequest("GET", "/api/read-only", nil)); err != nil || response.StatusCode != 200 {
		t.Fatalf("read request failed: status=%v err=%v", response.StatusCode, err)
	}
	var records []model.OperationLog
	if err := database.Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].DetailZH != "清理容器 [demo] 日志" {
		t.Fatalf("mapped detail or read filtering failed: %#v", records)
	}
}

func setupOperationLogDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	previous := global.DB
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "operation.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.OperationLog{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
	return database
}
