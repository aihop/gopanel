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

func TestValidateDBImportChunk(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		chunkIndex int
		chunkCount int
		wantError  bool
	}{
		{name: "valid", filename: "backup.sql", chunkIndex: 0, chunkCount: 2},
		{name: "parent traversal", filename: "../backup.sql", chunkIndex: 0, chunkCount: 1, wantError: true},
		{name: "absolute path", filename: "/tmp/backup.sql", chunkIndex: 0, chunkCount: 1, wantError: true},
		{name: "windows separator", filename: `..\\backup.sql`, chunkIndex: 0, chunkCount: 1, wantError: true},
		{name: "negative index", filename: "backup.sql", chunkIndex: -1, chunkCount: 1, wantError: true},
		{name: "index outside count", filename: "backup.sql", chunkIndex: 1, chunkCount: 1, wantError: true},
		{name: "empty count", filename: "backup.sql", chunkIndex: 0, chunkCount: 0, wantError: true},
		{name: "excessive count", filename: "backup.sql", chunkIndex: 0, chunkCount: maxDBImportChunkCount + 1, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDBImportChunk(test.filename, test.chunkIndex, test.chunkCount)
			if (err != nil) != test.wantError {
				t.Fatalf("validateDBImportChunk() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}
