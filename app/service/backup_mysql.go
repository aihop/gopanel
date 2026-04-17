package service

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
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

func (u *BackupService) MysqlBackup(req *dto.CommonBackup) error {
	localDir, err := loadLocalDir()
	if err != nil {
		return errors.New("load local dir failed, err: " + err.Error())
	}

	timeNow := time.Now().Format(constant.DateTimeSlimLayout)
	itemDir := fmt.Sprintf("database/%s/%s/%s", req.Type, req.Name, req.DetailName)
	targetDir := path.Join(localDir, itemDir)
	fileName := fmt.Sprintf("%s_%s.sql.gz", req.DetailName, timeNow+common.RandStrAndNum(5))

	if err := handleMysqlBackup(req.DetailId, req.DetailName, targetDir, fileName); err != nil {
		return errors.New("mysql backup failed, err: " + err.Error())
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
	}
	return nil
}

func (u *BackupService) MysqlRecover(req *dto.CommonRecover) error {
	if err := handleMysqlRecover(req, false); err != nil {
		return err
	}
	return nil
}

func (u *BackupService) MysqlRecoverByUpload(req *dto.CommonRecover) error {
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
	if err := handleMysqlRecover(req, false); err != nil {
		return err
	}
	global.LOG.Info("recover from uploads successful!")
	return nil
}

func handleMysqlBackup(serverId uint, dbName, targetDir, fileName string) error {
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

	backupInfo := client.BackupInfo{
		Name:      dbName,
		Type:      "mysql",
		Version:   version,
		Format:    "sql.gz",
		TargetDir: targetDir,
		FileName:  fileName,
		Timeout:   300,
	}
	if err := cli.Backup(backupInfo); err != nil {
		return err
	}
	return nil
}

func handleMysqlRecover(req *dto.CommonRecover, isRollback bool) error {
	isOk := false
	fileOp := files.NewFileOp()
	if !fileOp.Stat(req.File) {
		return errors.New("ErrFileNotFound: " + req.File)
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

	if !isRollback {
		rollbackFile := path.Join(global.CONF.System.TmpDir, fmt.Sprintf("database/%s/%s_%s.sql.gz", req.Type, req.DetailName, time.Now().Format(constant.DateTimeSlimLayout)))
		if err := cli.Backup(client.BackupInfo{
			Name:      req.DetailName,
			Type:      req.Type,
			Version:   version,
			Format:    "sql.gz",
			TargetDir: path.Dir(rollbackFile),
			FileName:  path.Base(rollbackFile),

			Timeout: 300,
		}); err != nil {
			return fmt.Errorf("backup mysql db %s for rollback before recover failed, err: %v", req.DetailName, err)
		}
		defer func() {
			if !isOk {
				global.LOG.Info("recover failed, start to rollback now")
				if err := cli.Recover(client.RecoverInfo{
					Name:       req.DetailName,
					Type:       req.Type,
					Version:    version,
					Format:     "sql.gz",
					SourceFile: rollbackFile,

					Timeout: 300,
				}); err != nil {
					global.LOG.Errorf("rollback mysql db %s from %s failed, err: %v", req.DetailName, rollbackFile, err)
				} else {
					global.LOG.Infof("rollback mysql db %s from %s successful", req.DetailName, rollbackFile)
				}
				_ = os.RemoveAll(rollbackFile)
			} else {
				_ = os.RemoveAll(rollbackFile)
			}
		}()
	}
	if err := cli.Recover(client.RecoverInfo{
		Name:       req.DetailName,
		Type:       req.Type,
		Version:    version,
		Format:     "sql.gz",
		SourceFile: req.File,

		Timeout: 300,
	}); err != nil {
		global.LOG.Errorf("recover mysql db %s from %s failed, err: %v", req.DetailName, req.File, err)
		return err
	}
	isOk = true
	return nil
}
