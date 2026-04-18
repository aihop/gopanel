package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type IAIGroupRepo interface {
	CreateGroup(group *model.AIGroup) error
	GetGroupByID(id uint) (*model.AIGroup, error)
	GetGroups(page, pageSize int) ([]*model.AIGroup, int64, error)
	UpdateGroup(group *model.AIGroup) error
	DeleteGroup(id uint) error
}

type aiGroupRepo struct{}

func NewAIGroupRepo() IAIGroupRepo {
	_ = global.DB.AutoMigrate(&model.AIGroup{})
	return &aiGroupRepo{}
}

func (r *aiGroupRepo) CreateGroup(group *model.AIGroup) error {
	return global.DB.Create(group).Error
}

func (r *aiGroupRepo) GetGroupByID(id uint) (*model.AIGroup, error) {
	var group model.AIGroup
	err := global.DB.Where("id = ?", id).First(&group).Error
	return &group, err
}

func (r *aiGroupRepo) GetGroups(page, pageSize int) ([]*model.AIGroup, int64, error) {
	var groups []*model.AIGroup
	var total int64
	db := global.DB.Model(&model.AIGroup{})
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

func (r *aiGroupRepo) UpdateGroup(group *model.AIGroup) error {
	return global.DB.Save(group).Error
}

func (r *aiGroupRepo) DeleteGroup(id uint) error {
	// 连带删除关联的 AI 任务及消息
	global.DB.Where("project_id = ?", id).Delete(&model.AITask{})
	return global.DB.Delete(&model.AIGroup{}, id).Error
}
