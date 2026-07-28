package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestExportTableRejectsRawWhere(t *testing.T) {
	app := fiber.New()
	app.Post("/export", ExportDBManagerTable)
	body := `{"serverId":1,"databaseName":"test","tableName":"users","format":"csv","where":"1=1; DROP TABLE users"}`
	resp, err := app.Test(httptest.NewRequest("POST", "/export", strings.NewReader(body)))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "raw WHERE conditions are not supported") {
		t.Fatalf("raw WHERE was not rejected: %s", data)
	}
}
