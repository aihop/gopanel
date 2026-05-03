package service

import (
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"os"
	"strconv"
	"strings"
	"time"
)

func estimateMysqlDBBytes(cli interface{}, dbName string) (int64, bool) {
	execer, ok := cli.(interface {
		ExecSQLForRows(command string, timeout uint) ([]string, error)
	})
	if !ok {
		return 0, false
	}
	safeDB := strings.ReplaceAll(dbName, "'", "''")
	lines, err := execer.ExecSQLForRows(fmt.Sprintf("SELECT SUM(DATA_LENGTH+INDEX_LENGTH) FROM information_schema.TABLES WHERE TABLE_SCHEMA='%s';", safeDB), 30)
	if err != nil {
		return 0, false
	}
	for i := len(lines) - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if s == "" {
			continue
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil && v > 0 {
			return v, true
		}
	}
	return 0, false
}
func readFileSize(p string) int64 {
	st, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return st.Size()
}
func formatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.2fGB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}
func resolveMysqlBackupType(serverType model.DatabaseType, fallback string) string {
	if serverType == model.DatabaseTypeMariaDB {
		return string(model.DatabaseTypeMariaDB)
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.ToLower(strings.TrimSpace(fallback))
	}
	return string(model.DatabaseTypeMysql)
}
func buildMysqlRecoverProgressLogger(stage string, logger *BackupLogger) func(readBytes, totalBytes int64) {
	if logger == nil {
		return nil
	}
	startAt := time.Now()
	lastBytes := int64(0)
	lastAt := time.Now()
	streamFinishedLogged := false
	return func(readBytes, totalBytes int64) {
		now := time.Now()
		dt := now.Sub(lastAt).Seconds()
		if dt <= 0 {
			dt = 1
		}
		speed := int64(float64(readBytes-lastBytes) / dt)
		elapsed := now.Sub(startAt).Round(time.Second)
		if totalBytes > 0 {
			percent := float64(readBytes) * 100 / float64(totalBytes)
			if percent > 100 {
				percent = 100
			}
			logger.Appendf("%s：耗时=%s，已读取=%s/%s，进度=%.1f%%，速度=%s/s", stage, elapsed, formatBytes(readBytes), formatBytes(totalBytes), percent, formatBytes(speed))
			if !streamFinishedLogged && readBytes >= totalBytes {
				streamFinishedLogged = true
				logger.Appendf("%s：恢复文件已读取完成，正在等待数据库执行收尾", stage)
			}
		} else {
			logger.Appendf("%s：耗时=%s，已读取=%s，速度=%s/s", stage, elapsed, formatBytes(readBytes), formatBytes(speed))
		}
		lastBytes = readBytes
		lastAt = now
	}
}
func calcMysqlBackupTimeout(estimatedBytes int64) uint {
	return calcMysqlDataTaskTimeout(estimatedBytes, 256*1024*1024, 10*60)
}
func calcMysqlRecoverTimeout(sourceBytes int64) uint {
	return calcMysqlDataTaskTimeout(sourceBytes, 128*1024*1024, 10*60)
}
func calcMysqlDataTaskTimeout(sizeBytes int64, chunkBytes int64, perChunkSeconds int64) uint {
	const (
		minSeconds = int64(30 * 60)
		maxSeconds = int64(24 * 60 * 60)
	)
	timeout := minSeconds
	if sizeBytes > 0 && chunkBytes > 0 && perChunkSeconds > 0 {
		chunks := (sizeBytes + chunkBytes - 1) / chunkBytes
		timeout += chunks * perChunkSeconds
	}
	if timeout < 300 {
		timeout = 300
	}
	if timeout > maxSeconds {
		timeout = maxSeconds
	}
	return uint(timeout)
}
func formatDurationSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	return (time.Duration(seconds) * time.Second).String()
}
