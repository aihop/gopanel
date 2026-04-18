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
	pgClient "github.com/aihop/gopanel/utils/postgresql/client"
)

func (u *BackupService) PostgresqlBackup(req *dto.CommonBackup, logger *BackupLogger) error {
	localDir := constant.BackupDir
	timeNow := time.Now().Format(constant.DateTimeSlimLayout)
	itemDir := fmt.Sprintf("database/%s/%s/%s", req.Type, req.Name, req.DetailName)
	targetDir := path.Join(localDir, itemDir)
	fileName := fmt.Sprintf("%s_%s.sql.gz", req.DetailName, timeNow+common.RandStrAndNum(5))
	if logger != nil {
		logger.Appendf("prepare backup: type=%s db=%s target=%s", req.Type, req.DetailName, path.Join(targetDir, fileName))
	}
	if err := handlePostgresqlBackup(req.Name, req.DetailName, targetDir, fileName, logger); err != nil {
		return err
	}
	if logger != nil {
		logger.AppendLine("backup file generated, saving record")
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
	backupRepo := repo.NewBackupRecord()
	if err := backupRepo.Create(record); err != nil {
		global.LOG.Errorf("save backup record failed, err: %v", err)
		if logger != nil {
			logger.Appendf("save backup record failed: %v", err)
		}
	}
	return nil
}
func (u *BackupService) PostgresqlRecover(req *dto.CommonRecover) error {
	if err := handlePostgresqlRecover(req, false); err != nil {
		return err
	}
	return nil
}

func (u *BackupService) PostgresqlRecoverByUpload(req *dto.CommonRecover) error {
	file := req.File
	fileName := path.Base(req.File)
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
		global.LOG.Infof("decompress file %s successful, now start to check test.sql is exist", req.File)
		hasTestSql := false
		_ = filepath.Walk(dstDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == "test.sql" {
				hasTestSql = true
				file = path
				fileName = "test.sql"
			}
			return nil
		})
		if !hasTestSql {
			_ = os.RemoveAll(dstDir)
			return fmt.Errorf("no such file named test.sql in %s", fileName)
		}
		defer func() {
			_ = os.RemoveAll(dstDir)
		}()
	}

	req.File = path.Dir(file) + "/" + fileName
	if err := handlePostgresqlRecover(req, false); err != nil {
		return err
	}
	global.LOG.Info("recover from uploads successful!")
	return nil
}
func handlePostgresqlBackup(database, dbName, targetDir, fileName string, logger *BackupLogger) error {
	cli, err := LoadPostgresqlClientByFrom(database)
	if err != nil {
		return err
	}
	defer cli.Close()

	estimatedBytes := int64(0)
	if estimate, ok := estimatePostgresqlDBBytes(cli, dbName); ok && estimate > 0 {
		estimatedBytes = estimate
		if logger != nil {
			logger.Appendf("estimated db size: %s", formatBytes(estimatedBytes))
		}
	}

	backupInfo := pgClient.BackupInfo{
		Name:      dbName,
		TargetDir: targetDir,
		FileName:  fileName,

		Timeout: 300,
	}
	if logger != nil {
		logger.AppendLine("starting pg_dump via docker exec")
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
						logger.Appendf("dumping... elapsed=%s output=%s speed=%s/s (db≈%s)", elapsed, formatBytes(size), formatBytes(speed), formatBytes(estimatedBytes))
					} else {
						logger.Appendf("dumping... elapsed=%s output=%s speed=%s/s", elapsed, formatBytes(size), formatBytes(speed))
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
			logger.Appendf("pg_dump failed: %v", err)
		}
		return err
	}
	close(stop)
	if logger != nil {
		logger.AppendLine("pg_dump finished")
		logger.Appendf("output file size: %s", formatBytes(readFileSize(outputFile)))
	}
	return nil
}

func estimatePostgresqlDBBytes(cli interface{}, dbName string) (int64, bool) {
	execer, ok := cli.(interface {
		ExecSQLForRows(command string, timeout uint) ([]string, error)
	})
	if !ok {
		return 0, false
	}
	safeDB := strings.ReplaceAll(dbName, "'", "''")
	lines, err := execer.ExecSQLForRows(fmt.Sprintf("SELECT pg_database_size('%s');", safeDB), 30)
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

func handlePostgresqlRecover(req *dto.CommonRecover, isRollback bool) error {
	isOk := false
	fileOp := files.NewFileOp()
	if !fileOp.Stat(req.File) {
		return errors.New("the recover file does not exist" + req.File)
	}
	// dbInfo, err := postgresqlRepo.Get(commonRepo.WithByName(req.DetailName), postgresqlRepo.WithByPostgresqlName(req.Name))
	// if err != nil {
	// 	return err
	// }
	cli, err := LoadPostgresqlClientByFrom(req.Name)
	if err != nil {
		return err
	}
	defer cli.Close()

	if !isRollback {
		rollbackFile := path.Join(global.CONF.System.TmpDir, fmt.Sprintf("database/%s/%s_%s.sql.gz", req.Type, req.DetailName, time.Now().Format(constant.DateTimeSlimLayout)))
		if err := cli.Backup(pgClient.BackupInfo{
			Name:      req.DetailName,
			TargetDir: path.Dir(rollbackFile),
			FileName:  path.Base(rollbackFile),

			Timeout: 300,
		}); err != nil {
			return fmt.Errorf("backup postgresql db %s for rollback before recover failed, err: %v", req.DetailName, err)
		}
		defer func() {
			if !isOk {
				global.LOG.Info("recover failed, start to rollback now")
				if err := cli.Recover(pgClient.RecoverInfo{
					Name:       req.DetailName,
					SourceFile: rollbackFile,

					Timeout: 300,
				}); err != nil {
					global.LOG.Errorf("rollback postgresql db %s from %s failed, err: %v", req.DetailName, rollbackFile, err)
				} else {
					global.LOG.Infof("rollback postgresql db %s from %s successful", req.DetailName, rollbackFile)
				}
				_ = os.RemoveAll(rollbackFile)
			} else {
				_ = os.RemoveAll(rollbackFile)
			}
		}()
	}
	if err := cli.Recover(pgClient.RecoverInfo{
		Name:       req.DetailName,
		SourceFile: req.File,
		// Username:   dbInfo.Username,
		Timeout: 300,
	}); err != nil {
		global.LOG.Errorf("recover postgresql db %s from %s failed, err: %v", req.DetailName, req.File, err)
		return err
	}
	isOk = true
	return nil
}
