package service

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

const websiteDiagnosticCaddyReadLimit = 4 * 1024 * 1024

type websiteDiagnosticCaddyCursor struct {
	Offset int64 `json:"offset"`
	Size   int64 `json:"size"`
}

type websiteDiagnosticCaddyEntry struct {
	Level     string  `json:"level"`
	Timestamp float64 `json:"ts"`
	Status    int     `json:"status"`
	Duration  float64 `json:"duration"`
	Error     string  `json:"error"`
	Request   struct {
		Method string              `json:"method"`
		URI    string              `json:"uri"`
		Host   string              `json:"host"`
		Header map[string][]string `json:"headers"`
	} `json:"request"`
}

func collectWebsiteCaddyEvents(website *model.Website, setting *model.WebsiteDiagnosticSetting) error {
	if !setting.Enabled || !setting.CaddyMonitoring {
		return nil
	}
	logPath := websiteAccessLogPath(website.Alias)
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	trackingDir, err := ensureWebsiteTrackingDirs(website.Alias)
	if err != nil {
		return err
	}
	cursorPath := filepath.Join(trackingDir, "caddy.cursor.json")
	cursor := readWebsiteCaddyCursor(cursorPath)
	if info.Size() < cursor.Offset {
		cursor.Offset = 0
	}
	if cursor.Offset == 0 && info.Size() > websiteDiagnosticCaddyReadLimit {
		cursor.Offset = info.Size() - websiteDiagnosticCaddyReadLimit
	}
	if _, err = file.Seek(cursor.Offset, io.SeekStart); err != nil {
		return err
	}
	reader := bufio.NewReader(io.LimitReader(file, websiteDiagnosticCaddyReadLimit))
	offset := cursor.Offset
	for {
		line, readErr := reader.ReadBytes('\n')
		if errors.Is(readErr, io.EOF) && len(line) > 0 {
			break
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if len(line) == 0 {
			break
		}
		lineOffset := offset
		offset += int64(len(line))
		line = bytes.TrimSpace(line)
		var entry websiteDiagnosticCaddyEntry
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		envelope := caddyEntryDiagnosticEnvelope(website, setting, &entry, line, lineOffset)
		if envelope == nil {
			continue
		}
		_, _, _ = ingestWebsiteDiagnosticEnvelope(website.ID, envelope)
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	return writeWebsiteCaddyCursor(cursorPath, websiteDiagnosticCaddyCursor{Offset: offset, Size: info.Size()})
}

func caddyEntryDiagnosticEnvelope(website *model.Website, setting *model.WebsiteDiagnosticSetting, entry *websiteDiagnosticCaddyEntry, raw []byte, offset int64) *WebsiteDiagnosticEnvelope {
	durationMS := int64(entry.Duration * 1000)
	kind := ""
	severity := "error"
	switch {
	case entry.Status >= 500 && setting.MonitorHTTP5xx:
		kind = "http_5xx"
	case entry.Status >= 400 && entry.Status < 500 && setting.MonitorHTTP4xx:
		kind, severity = "http_4xx", "warning"
	case strings.TrimSpace(entry.Error) != "" && setting.MonitorUpstreamErrors:
		kind = "upstream_error"
	case durationMS >= int64(setting.SlowRequestThresholdMS) && setting.MonitorSlowRequests:
		kind, severity = "slow_request", "warning"
	default:
		return nil
	}
	sum := sha256.Sum256(append([]byte(fmt.Sprintf("%d:", offset)), raw...))
	occurredAt := time.Now()
	if entry.Timestamp > 0 {
		occurredAt = time.UnixMilli(int64(entry.Timestamp * 1000))
	}
	requestID := ""
	for _, key := range []string{"X-Request-Id", "X-Request-ID"} {
		if values := entry.Request.Header[key]; len(values) > 0 {
			requestID = values[0]
			break
		}
	}
	return &WebsiteDiagnosticEnvelope{
		Schema: websiteDiagnosticSchema, EventID: "caddy-" + hex.EncodeToString(sum[:12]), WebsiteID: website.ID,
		Source: "caddy", Kind: kind, Severity: severity, Title: diagnosticDefaultTitle(kind, entry.Status, ""),
		Message: entry.Error, RequestID: requestID, Method: entry.Request.Method, Route: entry.Request.URI,
		HTTPStatus: entry.Status, DurationMS: durationMS, Release: activeWebsiteRelease(website.ID), OccurredAt: occurredAt,
	}
}

func readWebsiteCaddyCursor(path string) websiteDiagnosticCaddyCursor {
	data, err := os.ReadFile(path)
	if err != nil {
		return websiteDiagnosticCaddyCursor{}
	}
	var cursor websiteDiagnosticCaddyCursor
	_ = json.Unmarshal(data, &cursor)
	return cursor
}

func writeWebsiteCaddyCursor(path string, cursor websiteDiagnosticCaddyCursor) error {
	data, err := json.Marshal(cursor)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, data, 0640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
