package service

import (
	"context"
	"encoding/json"
	"path"
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

type loadSizeHelper struct {
	isOk       bool
	backupPath string
	client     storage.Client
}

func (u *BackupRecordService) loadRecordSize(records []*model.BackupRecord) ([]dto.BackupFile, error) {
	var datas []dto.BackupFile
	clientMap := make(map[string]loadSizeHelper)
	var wg sync.WaitGroup
	var backupAccountRepo = repo.NewCloudAccount()
	for i := 0; i < len(records); i++ {
		var item dto.BackupFile
		item.ID = records[i].ID
		item.Name = records[i].FileName
		itemPath := path.Join(records[i].FileDir, records[i].FileName)
		if _, ok := clientMap[string(records[i].Source)]; !ok {
			backup, err := backupAccountRepo.Get(context.Background(), string(records[i].Source))
			if err != nil {
				global.LOG.Errorf("load backup model %s from db failed, err: %v", records[i].Source, err)
				clientMap[string(records[i].Source)] = loadSizeHelper{}
				datas = append(datas, item)
				continue
			}
			client, config, err := u.NewClient(&backup)
			if err != nil {
				global.LOG.Errorf("load backup client %s from db failed, err: %v", records[i].Source, err)
				clientMap[string(records[i].Source)] = loadSizeHelper{}
				datas = append(datas, item)
				continue
			}
			item.Size, _ = client.Size(path.Join(strings.TrimLeft(config.BackupPath, "/"), itemPath))
			datas = append(datas, item)
			clientMap[string(records[i].Source)] = loadSizeHelper{backupPath: strings.TrimLeft(config.BackupPath, "/"), client: client, isOk: true}
			continue
		}
		if clientMap[string(records[i].Source)].isOk {
			wg.Add(1)
			go func(index int) {
				item.Size, _ = clientMap[string(records[index].Source)].client.Size(path.Join(clientMap[string(records[index].Source)].backupPath, itemPath))
				datas = append(datas, item)
				wg.Done()
			}(i)
		} else {
			datas = append(datas, item)
		}
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
