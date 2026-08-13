package service

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

type BackupService struct {
	repo *repo.BackupRepo
}

func NewBackup() *BackupService {
	return &BackupService{
		repo: repo.NewBackup(),
	}
}

func (u *BackupService) BatchDeleteRecord(ids []uint) error {
	backupRepo := repo.NewBackup()
	records, err := backupRepo.ListRecord(commonRepo.WithIdsIn(ids))
	if err != nil {
		return err
	}
	for _, record := range records {
		if string(record.Source) != "LOCAL" {
			continue
		}
		filePath := path.Join(constant.BackupDir, record.FileDir, record.FileName)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			global.LOG.Errorf("delete backup file %s failed, err: %v", filePath, err)
		}
	}
	return backupRepo.DeleteRecord(context.Background(), commonRepo.WithIdsIn(ids))
}

func (u *BackupService) DownloadRecord(info dto.DownloadRecord) (string, error) {
	if info.Source == "LOCAL" {
		return path.Join(constant.BackupDir, info.FileDir, info.FileName), nil
	}
	return "", errors.New("direct download currently supports local backup files only")
}

func (u *BackupService) DownloadRecordByID(id uint) (string, string, error) {
	record, err := repo.NewBackupRecord().GetByID(id)
	if err != nil {
		return "", "", err
	}
	if record.Source != "LOCAL" {
		return "", "", errors.New("direct download currently supports local backup files only")
	}
	backupRoot, err := filepath.Abs(constant.BackupDir)
	if err != nil {
		return "", "", err
	}
	filePath, err := filepath.Abs(filepath.Join(backupRoot, record.FileDir, record.FileName))
	if err != nil {
		return "", "", err
	}
	if filePath != backupRoot && !strings.HasPrefix(filePath, backupRoot+string(os.PathSeparator)) {
		return "", "", errors.New("backup file path is outside the backup directory")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return "", "", errors.New("backup path is a directory")
	}
	return filePath, record.FileName, nil
}
