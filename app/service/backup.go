package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
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
		localDir, err := loadLocalDir()
		if err != nil {
			return "", err
		}
		return path.Join(localDir, info.FileDir, info.FileName), nil
	}
	return "", nil
}

func loadLocalDir() (string, error) {
	backupRepo := repo.NewBackup()
	backup, err := backupRepo.Get(commonRepo.WithByType("LOCAL"))
	if err != nil {
		return "", err
	}
	varMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(backup.Vars), &varMap); err != nil {
		return "", err
	}
	if _, ok := varMap["dir"]; !ok {
		return "", errors.New("load local backup dir failed")
	}
	baseDir, ok := varMap["dir"].(string)
	if ok {
		if _, err := os.Stat(baseDir); err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(baseDir, os.ModePerm); err != nil {
				return "", fmt.Errorf("mkdir %s failed, err: %v", baseDir, err)
			}
		}
		return baseDir, nil
	}
	return "", fmt.Errorf("error type dir: %T", varMap["dir"])
}
