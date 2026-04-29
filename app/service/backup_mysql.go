package service

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/mysql/client"
)

func (u *BackupService) MysqlBackup(req *dto.CommonBackup, logger *BackupLogger) error {
	localDir := constant.BackupDir
	timeNow := time.Now().Format(constant.DateTimeSlimLayout)
	itemDir := fmt.Sprintf("database/%s/%s/%s", req.Type, req.Name, req.DetailName)
	targetDir := path.Join(localDir, itemDir)
	fileName := fmt.Sprintf("%s_%s.sql.gz", req.DetailName, timeNow+common.RandStrAndNum(5))
	if logger != nil {
		logger.Appendf("准备备份：类型=%s，数据库=%s，目标=%s", req.Type, req.DetailName, path.Join(targetDir, fileName))
	}
	if err := handleMysqlBackup(req.DetailId, req.DetailName, targetDir, fileName, logger); err != nil {
		return errors.New("mysql backup failed, err: " + err.Error())
	}
	if logger != nil {
		logger.AppendLine("备份文件已生成，正在保存记录")
	}
	record := &model.BackupRecord{
		Type:       req.Type,
		Name:       req.Name,
		DetailName: req.DetailName,
		Source:     "LOCAL",
		BackupType: "LOCAL",
		FileDir:    itemDir,
		FileName:   fileName,
	}
	backupRecordRepo := repo.NewBackupRecord()
	if err := backupRecordRepo.Create(record); err != nil {
		global.LOG.Errorf("save backup record failed, err: %v", err)
		if logger != nil {
			logger.Appendf("保存备份记录失败：%v", err)
		}
	}
	return nil
}

func (u *BackupService) MysqlRecover(req *dto.CommonRecover, logger *BackupLogger) error {
	if err := handleMysqlRecover(req, false, logger); err != nil {
		return err
	}
	return nil
}

func (u *BackupService) MysqlRecoverByUpload(req *dto.CommonRecover, logger *BackupLogger) error {
	file := req.File
	fileName := path.Base(req.File)
	if logger != nil {
		logger.Appendf("准备从上传文件恢复：%s", req.File)
	}
	if strings.HasSuffix(fileName, ".tar.gz") {
		fileNameItem := time.Now().Format(constant.DateTimeSlimLayout)
		dstDir := fmt.Sprintf("%s/%s", path.Dir(req.File), fileNameItem)
		if _, err := os.Stat(dstDir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(dstDir, os.ModePerm); err != nil {
				return fmt.Errorf("mkdir %s failed, err: %v", dstDir, err)
			}
		}
		if err := handleUnTar(req.File, dstDir, ""); err != nil {
			_ = os.RemoveAll(dstDir)
			return err
		}
		if logger != nil {
			logger.Appendf("压缩包已解压到：%s", dstDir)
		}
		global.LOG.Infof("decompress file %s successful, now start to check sql file", req.File)
		foundSQL := false
		var firstSQLPath string
		_ = filepath.Walk(dstDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			name := strings.ToLower(info.Name())
			if name == "test.sql" || name == "test.sql.gz" {
				foundSQL = true
				file = path
				fileName = info.Name()
				return filepath.SkipAll
			}
			if firstSQLPath == "" && (strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".sql.gz")) {
				firstSQLPath = path
			}
			return nil
		})
		if !foundSQL && firstSQLPath != "" {
			foundSQL = true
			file = firstSQLPath
			fileName = filepath.Base(firstSQLPath)
		}
		if !foundSQL {
			_ = os.RemoveAll(dstDir)
			return fmt.Errorf("压缩包中未找到可恢复的 SQL 文件（*.sql 或 *.sql.gz）：%s", fileName)
		}
		defer func() {
			_ = os.RemoveAll(dstDir)
		}()
	}

	req.File = path.Dir(file) + "/" + fileName
	if logger != nil {
		logger.Appendf("已确定恢复源文件：%s", req.File)
	}
	if err := handleMysqlRecover(req, false, logger); err != nil {
		return err
	}
	global.LOG.Info("recover from uploads successful!")
	if logger != nil {
		logger.AppendLine("上传文件恢复完成")
	}
	return nil
}

func handleMysqlBackup(serverId uint, dbName, targetDir, fileName string, logger *BackupLogger) error {
	databaseServerRepo := repo.NewDatabaseServer()
	dbInfo, err := databaseServerRepo.Get(serverId)
	if err != nil {
		return errors.New("get database server info failed, err: " + err.Error())
	}
	// 链接数据库
	cli, version, err := LoadMysqlClientByFrom(dbInfo)
	if err != nil {
		return errors.New("load mysql client failed, err: " + err.Error())
	}

	estimatedBytes := int64(0)
	if estimate, ok := estimateMysqlDBBytes(cli, dbName); ok && estimate > 0 {
		estimatedBytes = estimate
		if logger != nil {
			logger.Appendf("预估数据库大小：%s", formatBytes(estimatedBytes))
		}
	}

	backupInfo := client.BackupInfo{
		Name:      dbName,
		Type:      resolveMysqlBackupType(dbInfo.Type, "mysql"),
		Version:   version,
		Format:    "sql.gz",
		TargetDir: targetDir,
		FileName:  fileName,
		Timeout:   calcMysqlBackupTimeout(estimatedBytes),
	}
	if logger != nil {
		logger.AppendLine("开始执行 MySQL 备份")
		logger.Appendf("备份超时阈值：%s", formatDurationSeconds(int64(backupInfo.Timeout)))
	}

	outputFile := path.Join(targetDir, fileName)
	stop := make(chan struct{})
	if logger != nil {
		startAt := time.Now()
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			var lastSize int64
			var lastAt = time.Now()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					size := readFileSize(outputFile)
					dt := time.Since(lastAt).Seconds()
					if dt <= 0 {
						dt = 1
					}
					speed := int64(float64(size-lastSize) / dt)
					elapsed := time.Since(startAt).Round(time.Second)
					if estimatedBytes > 0 {
						logger.Appendf("备份中：耗时=%s，已输出=%s，速度=%s/s（数据库约=%s）", elapsed, formatBytes(size), formatBytes(speed), formatBytes(estimatedBytes))
					} else {
						logger.Appendf("备份中：耗时=%s，已输出=%s，速度=%s/s", elapsed, formatBytes(size), formatBytes(speed))
					}
					lastSize = size
					lastAt = time.Now()
				}
			}
		}()
	}

	if err := cli.Backup(backupInfo); err != nil {
		close(stop)
		if logger != nil {
			logger.Appendf("MySQL 备份失败：%v", err)
		}
		return err
	}
	close(stop)
	if logger != nil {
		logger.AppendLine("MySQL 备份完成")
		logger.Appendf("备份文件大小：%s", formatBytes(readFileSize(outputFile)))
	}
	return nil
}

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

func handleMysqlRecover(req *dto.CommonRecover, isRollback bool, logger *BackupLogger) error {
	isOk := false
	fileOp := files.NewFileOp()
	if !fileOp.Stat(req.File) {
		return errors.New("ErrFileNotFound: " + req.File)
	}
	if logger != nil {
		logger.Appendf("开始 MySQL 恢复：数据库=%s，文件=%s，回滚模式=%v", req.DetailName, req.File, isRollback)
	}
	databaseServiceRepo := repo.NewDatabaseServer()
	dbInfo, err := databaseServiceRepo.Get(req.DetailId)
	if err != nil {
		return errors.New("获取数据库服务器失败: " + err.Error())
	}
	cli, version, err := LoadMysqlClientByFrom(dbInfo)
	if err != nil {
		return errors.New("加载 MySQL 客户端失败: " + err.Error())
	}
	dbType := resolveMysqlBackupType(dbInfo.Type, req.Type)
	recoverTimeout := calcMysqlRecoverTimeout(readFileSize(req.File))
	if logger != nil {
		logger.Appendf("已连接 MySQL 服务：%s:%d，版本=%s", dbInfo.Host, dbInfo.Port, version)
		logger.Appendf("恢复超时阈值：%s", formatDurationSeconds(int64(recoverTimeout)))
	}

	if !isRollback {
		rollbackFile := path.Join(global.CONF.System.TmpDir, fmt.Sprintf("database/%s/%s_%s.sql.gz", req.Type, req.DetailName, time.Now().Format(constant.DateTimeSlimLayout)))
		if logger != nil {
			logger.Appendf("正在生成恢复前回滚备份：%s", rollbackFile)
		}
		if err := cli.Backup(client.BackupInfo{
			Name:      req.DetailName,
			Type:      dbType,
			Version:   version,
			Format:    "sql.gz",
			TargetDir: path.Dir(rollbackFile),
			FileName:  path.Base(rollbackFile),

			Timeout: calcMysqlBackupTimeout(0),
		}); err != nil {
			return fmt.Errorf("backup mysql db %s for rollback before recover failed, err: %v", req.DetailName, err)
		}
		if logger != nil {
			logger.AppendLine("回滚备份已完成")
		}
		defer func() {
			if !isOk {
				global.LOG.Info("recover failed, start to rollback now")
				if logger != nil {
					logger.AppendLine("恢复失败，开始执行回滚")
				}
				if err := cli.Recover(client.RecoverInfo{
					Name:       req.DetailName,
					Type:       dbType,
					Version:    version,
					Format:     "sql.gz",
					SourceFile: rollbackFile,
					Progress:   buildMysqlRecoverProgressLogger("回滚中", logger),

					Timeout: calcMysqlRecoverTimeout(readFileSize(rollbackFile)),
				}); err != nil {
					global.LOG.Errorf("rollback mysql db %s from %s failed, err: %v", req.DetailName, rollbackFile, err)
					if logger != nil {
						logger.Appendf("回滚失败：%v", err)
					}
				} else {
					global.LOG.Infof("rollback mysql db %s from %s successful", req.DetailName, rollbackFile)
					if logger != nil {
						logger.AppendLine("回滚完成")
					}
				}
				_ = os.RemoveAll(rollbackFile)
			} else {
				_ = os.RemoveAll(rollbackFile)
			}
		}()
	}
	if logger != nil {
		logger.AppendLine("开始执行 MySQL 恢复")
	}
	if err := cli.Recover(client.RecoverInfo{
		Name:       req.DetailName,
		Type:       dbType,
		Version:    version,
		Format:     "sql.gz",
		SourceFile: req.File,
		Progress:   buildMysqlRecoverProgressLogger("恢复中", logger),

		Timeout: recoverTimeout,
	}); err != nil {
		global.LOG.Errorf("recover mysql db %s from %s failed, err: %v", req.DetailName, req.File, err)
		return err
	}
	isOk = true
	if logger != nil {
		logger.AppendLine("MySQL 恢复完成")
	}
	return nil
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
