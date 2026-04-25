package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
)

var websiteLogStatusPattern = regexp.MustCompile(`"status"\s*:\s*(\d{3})`)
var websiteLogAnsiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type websiteExtractedLogPayload struct {
	Timestamp string
	JSONText  string
}

type websiteLogRequestPayload struct {
	Request struct {
		RemoteIP string `json:"remote_ip"`
		ClientIP string `json:"client_ip"`
		Method   string `json:"method"`
		Host     string `json:"host"`
		URI      string `json:"uri"`
	} `json:"request"`
	Status int `json:"status"`
}

func websiteLogDir(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ""
	}
	alias = strings.TrimLeft(alias, "/")
	alias = strings.ReplaceAll(alias, "/", "_")
	alias = strings.ReplaceAll(alias, string(filepath.Separator), "_")
	return filepath.Join(global.CONF.System.BaseDir, "log", "website", alias)
}

func websiteAccessLogPath(alias string) string {
	return filepath.Join(websiteLogDir(alias), constant.AccessLog)
}

func websiteErrorLogPath(alias string) string {
	return filepath.Join(websiteLogDir(alias), constant.ErrorLog)
}

func ensureWebsiteLogDir(alias string) error {
	dir := websiteLogDir(alias)
	if dir == "" {
		return errors.New("网站别名不能为空")
	}
	return files.NewFileOp().CreateDir(dir, 0755)
}

func resolveWebsiteLogPath(alias, logType string) string {
	switch strings.ToLower(strings.TrimSpace(logType)) {
	case "error":
		return websiteErrorLogPath(alias)
	default:
		return websiteAccessLogPath(alias)
	}
}

func fileHasContent(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func isErrorAccessLogLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if strings.Contains(strings.ToLower(line), `"level":"error"`) {
		return true
	}
	matches := websiteLogStatusPattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return false
	}
	status, err := strconv.Atoi(matches[1])
	if err != nil {
		return false
	}
	return status >= 400
}

func readDerivedWebsiteErrorLines(alias string) ([]string, string, error) {
	sourcePath := websiteAccessLogPath(alias)
	file, err := os.Open(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, sourcePath, nil
		}
		return nil, sourcePath, err
	}
	defer file.Close()

	lines := make([]string, 0, 128)
	scanner := newWebsiteLogScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if isErrorAccessLogLine(line) {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, sourcePath, err
	}
	return lines, sourcePath, nil
}

func newWebsiteLogScanner(file *os.File) *bufio.Scanner {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return scanner
}

func stripWebsiteLogANSI(value string) string {
	return websiteLogAnsiPattern.ReplaceAllString(value, "")
}

func extractWebsiteLogPayload(raw string) *websiteExtractedLogPayload {
	clean := strings.TrimSpace(stripWebsiteLogANSI(raw))
	if clean == "" {
		return nil
	}
	jsonStart := strings.Index(clean, "{")
	if jsonStart < 0 {
		return nil
	}
	prefix := strings.TrimSpace(clean[:jsonStart])
	jsonText := strings.TrimSpace(clean[jsonStart:])
	return &websiteExtractedLogPayload{
		Timestamp: extractWebsiteLogTimestamp(prefix),
		JSONText:  jsonText,
	}
}

func extractWebsiteLogTimestamp(prefix string) string {
	fields := regexp.MustCompile(`\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2}(?:\.\d+)?`).FindStringSubmatch(prefix)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func parseWebsiteLogTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006/01/02 15:04:05.999999999",
		"2006/01/02 15:04:05.999999",
		"2006/01/02 15:04:05.999",
		"2006/01/02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func sameLocalDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func preferredWebsiteLogIP(payload websiteLogRequestPayload) string {
	if ip := strings.TrimSpace(payload.Request.ClientIP); ip != "" {
		return ip
	}
	return strings.TrimSpace(payload.Request.RemoteIP)
}

func paginateTextLines(lines []string, page, pageSize int, latest bool) ([]string, bool, int) {
	if pageSize <= 0 {
		pageSize = 100
	}
	totalPages := 0
	if len(lines) > 0 {
		totalPages = (len(lines) + pageSize - 1) / pageSize
	}
	if totalPages == 0 {
		return []string{}, true, 0
	}
	if latest {
		page = totalPages
	}
	if page <= 0 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end], page >= totalPages, totalPages
}

func (s *WebsiteService) ReadWebsiteLog(req request.WebsiteLogRead) (*response.FileLineContent, error) {
	website, err := s.repo.GetFirst(s.repo.WithID(req.WebsiteID))
	if err != nil {
		return nil, err
	}
	logFilePath := resolveWebsiteLogPath(website.Alias, req.LogType)

	var (
		lines       []string
		isEndOfFile bool
		total       int
	)
	if strings.EqualFold(strings.TrimSpace(req.LogType), "error") && !fileHasContent(logFilePath) {
		lines, logFilePath, err = readDerivedWebsiteErrorLines(website.Alias)
		if err != nil {
			return nil, err
		}
		lines, isEndOfFile, total = paginateTextLines(lines, req.Page, req.Limit, req.Latest)
	} else {
		lines, isEndOfFile, total, err = files.ReadFileByLine(logFilePath, req.Page, req.Limit, req.Latest)
		if err != nil {
			return nil, err
		}
	}
	return &response.FileLineContent{
		Content: strings.Join(lines, "\n"),
		End:     isEndOfFile,
		Path:    logFilePath,
		Total:   total,
		Lines:   lines,
	}, nil
}

func (s *WebsiteService) ReadWebsiteTodayIPStats(req request.WebsiteLogTodayIPStats) (*response.WebsiteLogTodayIPStats, error) {
	website, err := s.repo.GetFirst(s.repo.WithID(req.WebsiteID))
	if err != nil {
		return nil, err
	}
	logFilePath := websiteAccessLogPath(website.Alias)
	file, err := os.Open(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &response.WebsiteLogTodayIPStats{
				Date:          time.Now().Format("2006-01-02"),
				UniqueIPCount: 0,
				RequestCount:  0,
				Path:          logFilePath,
				TopIPs:        []response.WebsiteLogTopIP{},
			}, nil
		}
		return nil, err
	}
	defer file.Close()

	latestDate := ""
	ipCounter := map[string]int{}
	requestCount := 0
	scanner := newWebsiteLogScanner(file)
	for scanner.Scan() {
		extracted := extractWebsiteLogPayload(scanner.Text())
		if extracted == nil || extracted.JSONText == "" {
			continue
		}
		logTime, ok := parseWebsiteLogTimestamp(extracted.Timestamp)
		if !ok {
			continue
		}
		var payload websiteLogRequestPayload
		if err := json.Unmarshal([]byte(extracted.JSONText), &payload); err != nil {
			continue
		}
		ip := preferredWebsiteLogIP(payload)
		if ip == "" {
			continue
		}
		logDate := logTime.Format("2006-01-02")
		if latestDate == "" || logDate > latestDate {
			latestDate = logDate
			ipCounter = map[string]int{}
			requestCount = 0
		}
		if logDate != latestDate {
			continue
		}
		requestCount++
		ipCounter[ip]++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	topIPs := make([]response.WebsiteLogTopIP, 0, len(ipCounter))
	for ip, count := range ipCounter {
		topIPs = append(topIPs, response.WebsiteLogTopIP{
			IP:    ip,
			Count: count,
		})
	}
	sort.Slice(topIPs, func(i, j int) bool {
		if topIPs[i].Count == topIPs[j].Count {
			return topIPs[i].IP < topIPs[j].IP
		}
		return topIPs[i].Count > topIPs[j].Count
	})
	if len(topIPs) > 10 {
		topIPs = topIPs[:10]
	}

	return &response.WebsiteLogTodayIPStats{
		Date:          latestDate,
		UniqueIPCount: len(ipCounter),
		RequestCount:  requestCount,
		Path:          logFilePath,
		TopIPs:        topIPs,
	}, nil
}
