package service

import (
	"context"
	"fmt"
	"path"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
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
	fmt.Println(records)
	return backupRepo.DeleteRecord(context.Background(), commonRepo.WithIdsIn(ids))
}

func (u *BackupService) DownloadRecord(info dto.DownloadRecord) (string, error) {
	if info.Source == "LOCAL" {
		return path.Join(constant.BackupDir, info.FileDir, info.FileName), nil
	}
	return "", nil
}
