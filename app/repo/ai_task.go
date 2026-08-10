package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type IAITaskRepo interface {
	CreateTask(task *model.AITask) error
	GetTaskByID(id uint) (*model.AITask, error)
	GetTasksByUserID(userID uint, page, limit int) ([]*model.AITask, int64, error)
	GetTasksByProjectID(projectID uint, page, limit int) ([]*model.AITask, int64, error)
	GetTasksByProjectAndUserID(projectID, userID uint, page, limit int) ([]*model.AITask, int64, error)
	UpdateTask(task *model.AITask) error
	DeleteTask(id uint) error
	DeleteTaskAndSession(taskID, sessionID uint) error

	CreateMessage(msg *model.AIMessage) error
	GetMessagesByTaskID(taskID uint) ([]*model.AIMessage, error)
	GetMessagesBySessionID(sessionID uint) ([]*model.AIMessage, error)
	GetMessagesBySessionAndTaskID(sessionID, taskID uint) ([]*model.AIMessage, error)
}

type aiTaskRepo struct{}

func NewAITaskRepo() IAITaskRepo {
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

func (r *aiTaskRepo) GetTasksByUserID(userID uint, page, limit int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64
	db := global.DB.Model(&model.AITask{}).Where("user_id = ?", userID)
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func (r *aiTaskRepo) GetTasksByProjectID(projectID uint, page, limit int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64
	db := global.DB.Model(&model.AITask{}).Where("project_id = ?", projectID)
	db.Count(&total)
	err := orderCodeTasks(db).Offset((page - 1) * limit).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func (r *aiTaskRepo) GetTasksByProjectAndUserID(projectID, userID uint, page, limit int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64
	db := global.DB.Model(&model.AITask{}).Where("project_id = ? AND user_id = ?", projectID, userID)
	db.Count(&total)
	err := orderCodeTasks(db).Offset((page - 1) * limit).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func orderCodeTasks(db *gorm.DB) *gorm.DB {
	return db.Order("CASE status WHEN 'pending_approval' THEN 0 WHEN 'delivering' THEN 1 WHEN 'running' THEN 2 WHEN 'queued' THEN 3 ELSE 4 END").
		Order("CASE WHEN status IN ('pending_approval', 'delivering', 'running', 'queued') THEN updated_at ELSE created_at END desc")
}

func (r *aiTaskRepo) UpdateTask(task *model.AITask) error {
	return global.DB.Save(task).Error
}

func (r *aiTaskRepo) DeleteTask(id uint) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		return r.deleteTask(tx, id, 0)
	})
}

func (r *aiTaskRepo) DeleteTaskAndSession(taskID, sessionID uint) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		return r.deleteTask(tx, taskID, sessionID)
	})
}

func (r *aiTaskRepo) deleteTask(db *gorm.DB, taskID, sessionID uint) error {
	if sessionID > 0 {
		var jobIDs []uint
		if err := db.Model(&model.AICodeDeliveryJob{}).Where("session_id = ?", sessionID).Pluck("id", &jobIDs).Error; err != nil {
			return err
		}
		if len(jobIDs) > 0 {
			if err := db.Where("job_id IN ?", jobIDs).Delete(&model.AICodeDeliveryLease{}).Error; err != nil {
				return err
			}
		}
		for _, target := range []any{
			&model.AICodeDeliveryJob{}, &model.AICodeDelivery{}, &model.AIDevSessionRepository{},
		} {
			if err := db.Where("session_id = ?", sessionID).Delete(target).Error; err != nil {
				return err
			}
		}
	}
	for _, target := range []any{
		&model.AIMessage{}, &model.AIApproval{}, &model.AIExecutionRun{},
		&model.AIPreview{}, &model.AITimelineEvent{}, &model.AIInstruction{},
	} {
		query := db.Where("task_id = ?", taskID)
		if sessionID > 0 {
			query = db.Where("task_id = ? OR session_id = ?", taskID, sessionID)
		}
		if err := query.Delete(target).Error; err != nil {
			return err
		}
	}
	if sessionID > 0 {
		if err := db.Delete(&model.AIDevSession{}, sessionID).Error; err != nil {
			return err
		}
	} else if err := db.Model(&model.AIDevSession{}).
		Where("last_task_id = ?", taskID).
		Updates(map[string]any{"last_task_id": 0, "current_stage": "idle"}).Error; err != nil {
		return err
	}
	return db.Delete(&model.AITask{}, taskID).Error
}

func (r *aiTaskRepo) CreateMessage(msg *model.AIMessage) error {
	return global.DB.Create(msg).Error
}

func (r *aiTaskRepo) GetMessagesByTaskID(taskID uint) ([]*model.AIMessage, error) {
	var messages []*model.AIMessage
	err := global.DB.Where("task_id = ?", taskID).Order("created_at asc").Find(&messages).Error
	return messages, err
}

func (r *aiTaskRepo) GetMessagesBySessionID(sessionID uint) ([]*model.AIMessage, error) {
	var messages []*model.AIMessage
	err := global.DB.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error
	return messages, err
}

// GetMessagesBySessionAndTaskID 只取某个任务的对话。
// 同时带上 session_id 是为了防止越过会话边界读到别的会话的消息。
func (r *aiTaskRepo) GetMessagesBySessionAndTaskID(sessionID, taskID uint) ([]*model.AIMessage, error) {
	var messages []*model.AIMessage
	err := global.DB.Where("session_id = ? AND task_id = ?", sessionID, taskID).
		Order("created_at asc").Find(&messages).Error
	return messages, err
}
