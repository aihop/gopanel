package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type IAITaskRepo interface {
	CreateTask(task *model.AITask) error
	GetTaskByID(id uint) (*model.AITask, error)
	GetTasksByUserID(userID uint, page, pageSize int) ([]*model.AITask, int64, error)
	GetTasksByProjectID(projectID uint, page, pageSize int) ([]*model.AITask, int64, error)
	UpdateTask(task *model.AITask) error
	DeleteTask(id uint) error

	CreateMessage(msg *model.AIMessage) error
	GetMessagesByTaskID(taskID uint) ([]*model.AIMessage, error)
}

type aiTaskRepo struct{}

func NewAITaskRepo() IAITaskRepo {
	_ = global.DB.AutoMigrate(&model.AITask{}, &model.AIMessage{})
	return &aiTaskRepo{}
}

func (r *aiTaskRepo) CreateTask(task *model.AITask) error {
	return global.DB.Create(task).Error
}

func (r *aiTaskRepo) GetTaskByID(id uint) (*model.AITask, error) {
	var task model.AITask
	err := global.DB.Where("id = ?", id).First(&task).Error
	return &task, err
}

func (r *aiTaskRepo) GetTasksByUserID(userID uint, page, pageSize int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64
	db := global.DB.Model(&model.AITask{}).Where("user_id = ?", userID)
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func (r *aiTaskRepo) GetTasksByProjectID(projectID uint, page, pageSize int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64
	db := global.DB.Model(&model.AITask{}).Where("project_id = ?", projectID)
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func (r *aiTaskRepo) UpdateTask(task *model.AITask) error {
	return global.DB.Save(task).Error
}

func (r *aiTaskRepo) DeleteTask(id uint) error {
	// 同时删除关联的消息
	global.DB.Where("task_id = ?", id).Delete(&model.AIMessage{})
	return global.DB.Delete(&model.AITask{}, id).Error
}

func (r *aiTaskRepo) CreateMessage(msg *model.AIMessage) error {
	return global.DB.Create(msg).Error
}

func (r *aiTaskRepo) GetMessagesByTaskID(taskID uint) ([]*model.AIMessage, error) {
	var messages []*model.AIMessage
	err := global.DB.Where("task_id = ?", taskID).Order("created_at asc").Find(&messages).Error
	return messages, err
}
