package service

import (
	"context"
	"os"
	"path"

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
	return "", nil
}
