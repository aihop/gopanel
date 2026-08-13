package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/aihop/gopanel/utils/storage"
	"github.com/jinzhu/copier"
)

type BackupRecordService struct {
	repo *repo.BackupRecordRepo
}

func NewBackupRecord() *BackupRecordService {
	return &BackupRecordService{
		repo: repo.NewBackupRecord(),
	}
}

func (s *BackupRecordService) Search(c *gormx.Contextx) ([]dto.BackupRecords, error) {
	records, err := s.repo.List(c)
	if err != nil || records == nil {
		return nil, err
	}
	var list []dto.BackupRecords
	for _, item := range records {
		var itemRecord dto.BackupRecords
		if err := copier.Copy(&itemRecord, &item); err != nil {
			continue
		}
		list = append(list, itemRecord)
	}
	return list, nil
}

func (s *BackupRecordService) Sizes(c *gormx.Contextx) ([]dto.BackupFile, error) {
	records, err := s.repo.List(c)
	if err != nil || records == nil {
		return nil, err
	}
	return s.loadRecordSize(records)
}

func (s *BackupRecordService) SizeByID(id uint) (dto.BackupFile, error) {
	record, err := s.repo.GetByID(id)
	if err != nil {
		return dto.BackupFile{}, err
	}
	result := dto.BackupFile{ID: record.ID, Name: record.FileName}
	itemPath := path.Join(record.FileDir, record.FileName)
	if record.Source == model.BackupSourceLOCAL {
		backupRoot, err := filepath.Abs(constant.BackupDir)
		if err != nil {
			return dto.BackupFile{}, err
		}
		filePath, err := filepath.Abs(filepath.Join(backupRoot, record.FileDir, record.FileName))
		if err != nil || (filePath != backupRoot && !strings.HasPrefix(filePath, backupRoot+string(os.PathSeparator))) {
			return dto.BackupFile{}, errors.New("backup file path is outside the backup directory")
		}
		info, err := os.Stat(filePath)
		if err != nil {
			return dto.BackupFile{}, err
		}
		result.Size = info.Size()
		return result, nil
	}

	backup, err := repo.NewCloudAccount().Get(context.Background(), string(record.Source))
	if err != nil {
		return dto.BackupFile{}, err
	}
	client, config, err := s.NewClient(&backup)
	if err != nil {
		return dto.BackupFile{}, err
	}
	result.Size, err = client.Size(path.Join(strings.TrimLeft(config.BackupPath, "/"), itemPath))
	return result, err
}

type loadSizeHelper struct {
	isOk       bool
	backupPath string
	client     storage.Client
}

func (u *BackupRecordService) loadRecordSize(records []*model.BackupRecord) ([]dto.BackupFile, error) {
	datas := make([]dto.BackupFile, len(records))
	clientMap := make(map[string]loadSizeHelper)
	var backupAccountRepo = repo.NewCloudAccount()
	for i := 0; i < len(records); i++ {
		datas[i].ID = records[i].ID
		datas[i].Name = records[i].FileName
		if _, ok := clientMap[string(records[i].Source)]; !ok {
			backup, err := backupAccountRepo.Get(context.Background(), string(records[i].Source))
			if err != nil {
				global.LOG.Errorf("load backup model %s from db failed, err: %v", records[i].Source, err)
				clientMap[string(records[i].Source)] = loadSizeHelper{}
				continue
			}
			client, config, err := u.NewClient(&backup)
			if err != nil {
				global.LOG.Errorf("load backup client %s from db failed, err: %v", records[i].Source, err)
				clientMap[string(records[i].Source)] = loadSizeHelper{}
				continue
			}
			clientMap[string(records[i].Source)] = loadSizeHelper{backupPath: strings.TrimLeft(config.BackupPath, "/"), client: client, isOk: true}
		}
	}

	// 并发获取远端文件大小（每条记录一次网络往返，串行会让接口耗时随记录数线性放大），
	// 结果按下标写入各自的槽位，避免旧版并发 append 的数据竞争
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := 0; i < len(records); i++ {
		helper := clientMap[string(records[i].Source)]
		if !helper.isOk {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(index int, h loadSizeHelper) {
			defer func() {
				<-sem
				wg.Done()
			}()
			itemPath := path.Join(records[index].FileDir, records[index].FileName)
			datas[index].Size, _ = h.client.Size(path.Join(h.backupPath, itemPath))
		}(i, helper)
	}
	wg.Wait()
	return datas, nil
}

func (u *BackupRecordService) NewClient(backup *model.CloudAccount) (client storage.Client, config *model.CloudAccountStorage, err error) {
	varMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(backup.Authorization), &config); err != nil {
		return nil, nil, err
	}
	if config.Bucket != "" {
		varMap["bucket"] = config.Bucket
	}
	switch backup.Type {
	case constant.Sftp, constant.WebDAV:
		varMap["username"] = config.AccessKey
		varMap["password"] = config.Credential
	case constant.OSS, constant.S3, constant.MinIo, constant.Cos, constant.Kodo:
		varMap["accessKey"] = config.AccessKey
		varMap["secretKey"] = config.SecretKey
	}

	backClient, err := storage.NewClient(backup.Type, varMap)
	if err != nil {
		return backClient, config, err
	}

	return backClient, config, nil
}

func (s *BackupRecordService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}

func (s *BackupRecordService) DeleteByIds(ids []uint) error {

	backupRecordRepo := repo.NewBackupRecord()
	records, err := backupRecordRepo.ListRecord(commonRepo.WithIdsIn(ids))
	if err != nil {
		return err
	}
	cloudAccountRepo := repo.NewCloudAccount()
	for _, record := range records {
		cloudAccount, err := cloudAccountRepo.Get(context.Background(), string(record.Source))
		if err != nil {
			global.LOG.Errorf("load backup account %s info from db failed, err: %v", record.Source, err)
			continue
		}
		client,
			_, err := s.NewClient(&cloudAccount)
		if err != nil {
			global.LOG.Errorf("new client for backup account %s failed, err: %v", record.Source, err)
			continue
		}
		if _, err = client.Delete(path.Join(record.FileDir, record.FileName)); err != nil {
			global.LOG.Errorf("remove file %s from %s failed, err: %v", path.Join(record.FileDir, record.FileName), record.Source, err)
		}
	}
	return backupRecordRepo.Delete(context.Background(), commonRepo.WithIdsIn(ids))
}
