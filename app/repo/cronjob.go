package repo

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

func NewCronjob() *CronjobRepo {
	return &CronjobRepo{}
}

type CronjobRepo struct {
}

func (u *CronjobRepo) StartRecords(cronjobID uint, fromLocal bool, targetPath string) model.JobRecords {
	var record model.JobRecords
	record.StartTime = time.Now()
	record.CronjobID = cronjobID
	record.FromLocal = fromLocal
	record.Status = constant.StatusWaiting
	if err := global.DB.Create(&record).Error; err != nil {
		global.LOG.Errorf("create record status failed, err: %v", err)
	}
	return record
}

func (u *CronjobRepo) UpdateRecords(id uint, vars map[string]interface{}) error {
	return global.DB.Model(&model.JobRecords{}).Where("id = ?", id).Updates(vars).Error
}
