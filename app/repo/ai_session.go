package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type IAIDevSessionRepo interface {
	CreateSession(session *model.AIDevSession) error
	GetSessionByID(id uint) (*model.AIDevSession, error)
	GetSessionsByUserID(userID, projectID uint, page, limit int) ([]*model.AIDevSession, int64, error)
	UpdateSession(session *model.AIDevSession) error
	CreateExecutionRun(run *model.AIExecutionRun) error
	UpdateExecutionRun(run *model.AIExecutionRun) error
	GetExecutionRunsBySessionID(sessionID uint, page, limit int) ([]*model.AIExecutionRun, int64, error)
	GetExecutionRunByID(id uint) (*model.AIExecutionRun, error)
	GetLatestExecutionRunBySessionID(sessionID uint) (*model.AIExecutionRun, error)

	CreateInstruction(instruction *model.AIInstruction) error
	GetInstructionByID(id uint) (*model.AIInstruction, error)
	GetInstructionsBySessionID(sessionID uint, page, limit int) ([]*model.AIInstruction, int64, error)
	GetLatestInstructionBySessionID(sessionID uint) (*model.AIInstruction, error)
	GetPendingInstructionsBySessionID(sessionID uint) ([]*model.AIInstruction, error)
	ClaimInstruction(id uint) (bool, error)
	CancelQueuedInstructions(sessionID uint) error
	UpdateInstruction(instruction *model.AIInstruction) error

	CreatePreview(preview *model.AIPreview) error
	GetPreviewsBySessionID(sessionID uint, limit int) ([]*model.AIPreview, error)
	FindPreviewByURL(sessionID uint, previewURL string) (*model.AIPreview, error)
	UpdatePreview(preview *model.AIPreview) error

	CreateTimelineEvent(event *model.AITimelineEvent) error
	GetTimelineEventsBySessionID(sessionID uint, limit int) ([]*model.AITimelineEvent, error)

	CreateApproval(approval *model.AIApproval) error
	GetApprovalByID(id uint) (*model.AIApproval, error)
	GetApprovalsByUserID(userID uint, status string, limit int) ([]*model.AIApproval, error)
	GetPendingApprovalBySessionID(sessionID uint) (*model.AIApproval, error)
	UpdateApproval(approval *model.AIApproval) error
}

type aiDevSessionRepo struct{}

func NewAIDevSessionRepo() IAIDevSessionRepo {
	return &aiDevSessionRepo{}
}

func (r *aiDevSessionRepo) CreateSession(session *model.AIDevSession) error {
	return global.DB.Create(session).Error
}

func (r *aiDevSessionRepo) GetSessionByID(id uint) (*model.AIDevSession, error) {
	var session model.AIDevSession
	err := global.DB.Where("id = ?", id).First(&session).Error
	return &session, err
}

func (r *aiDevSessionRepo) GetSessionsByUserID(userID, projectID uint, page, limit int) ([]*model.AIDevSession, int64, error) {
	var sessions []*model.AIDevSession
	var total int64

	db := global.DB.Model(&model.AIDevSession{}).Where("user_id = ?", userID)
	if projectID > 0 {
		db = db.Where("project_id = ?", projectID)
	}

	db.Count(&total)
	err := db.Order("updated_at desc").Offset((page - 1) * limit).Limit(limit).Find(&sessions).Error
	return sessions, total, err
}

func (r *aiDevSessionRepo) UpdateSession(session *model.AIDevSession) error {
	return global.DB.Save(session).Error
}

func (r *aiDevSessionRepo) CreateExecutionRun(run *model.AIExecutionRun) error {
	return global.DB.Create(run).Error
}

func (r *aiDevSessionRepo) UpdateExecutionRun(run *model.AIExecutionRun) error {
	return global.DB.Save(run).Error
}

func (r *aiDevSessionRepo) GetExecutionRunsBySessionID(sessionID uint, page, limit int) ([]*model.AIExecutionRun, int64, error) {
	var runs []*model.AIExecutionRun
	var total int64
	db := global.DB.Model(&model.AIExecutionRun{}).Where("session_id = ?", sessionID)
	db.Count(&total)
	if limit > 0 {
		db = db.Offset((page - 1) * limit).Limit(limit)
	}
	err := db.Order("created_at asc").Find(&runs).Error
	return runs, total, err
}

func (r *aiDevSessionRepo) GetExecutionRunByID(id uint) (*model.AIExecutionRun, error) {
	var run model.AIExecutionRun
	err := global.DB.Where("id = ?", id).First(&run).Error
	return &run, err
}

func (r *aiDevSessionRepo) GetLatestExecutionRunBySessionID(sessionID uint) (*model.AIExecutionRun, error) {
	var run model.AIExecutionRun
	err := global.DB.Where("session_id = ?", sessionID).Order("created_at desc").First(&run).Error
	return &run, err
}

func (r *aiDevSessionRepo) CreateInstruction(instruction *model.AIInstruction) error {
	return global.DB.Create(instruction).Error
}

func (r *aiDevSessionRepo) GetInstructionByID(id uint) (*model.AIInstruction, error) {
	var instruction model.AIInstruction
	err := global.DB.Where("id = ?", id).First(&instruction).Error
	return &instruction, err
}

func (r *aiDevSessionRepo) GetInstructionsBySessionID(sessionID uint, page, limit int) ([]*model.AIInstruction, int64, error) {
	var instructions []*model.AIInstruction
	var total int64

	db := global.DB.Model(&model.AIInstruction{}).Where("session_id = ?", sessionID)
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&instructions).Error
	return instructions, total, err
}

func (r *aiDevSessionRepo) GetLatestInstructionBySessionID(sessionID uint) (*model.AIInstruction, error) {
	var instruction model.AIInstruction
	err := global.DB.Where("session_id = ?", sessionID).Order("created_at desc").First(&instruction).Error
	return &instruction, err
}

func (r *aiDevSessionRepo) GetPendingInstructionsBySessionID(sessionID uint) ([]*model.AIInstruction, error) {
	var instructions []*model.AIInstruction
	err := global.DB.
		Where("session_id = ? AND status = ?", sessionID, "queued").
		Order("created_at asc").
		Find(&instructions).Error
	return instructions, err
}

func (r *aiDevSessionRepo) ClaimInstruction(id uint) (bool, error) {
	result := global.DB.Model(&model.AIInstruction{}).
		Where("id = ? AND status = ?", id, "queued").
		Update("status", "running")
	return result.RowsAffected == 1, result.Error
}

func (r *aiDevSessionRepo) CancelQueuedInstructions(sessionID uint) error {
	return global.DB.Model(&model.AIInstruction{}).
		Where("session_id = ? AND status = ?", sessionID, "queued").
		Update("status", "cancelled").Error
}

func (r *aiDevSessionRepo) UpdateInstruction(instruction *model.AIInstruction) error {
	return global.DB.Save(instruction).Error
}

func (r *aiDevSessionRepo) CreatePreview(preview *model.AIPreview) error {
	return global.DB.Create(preview).Error
}

func (r *aiDevSessionRepo) GetPreviewsBySessionID(sessionID uint, limit int) ([]*model.AIPreview, error) {
	var previews []*model.AIPreview
	db := global.DB.Where("session_id = ?", sessionID).Order("updated_at desc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&previews).Error
	return previews, err
}

func (r *aiDevSessionRepo) FindPreviewByURL(sessionID uint, previewURL string) (*model.AIPreview, error) {
	var preview model.AIPreview
	err := global.DB.Where("session_id = ? AND url = ?", sessionID, previewURL).First(&preview).Error
	return &preview, err
}

func (r *aiDevSessionRepo) UpdatePreview(preview *model.AIPreview) error {
	return global.DB.Save(preview).Error
}

func (r *aiDevSessionRepo) CreateTimelineEvent(event *model.AITimelineEvent) error {
	return global.DB.Create(event).Error
}

func (r *aiDevSessionRepo) GetTimelineEventsBySessionID(sessionID uint, limit int) ([]*model.AITimelineEvent, error) {
	var events []*model.AITimelineEvent
	db := global.DB.Where("session_id = ?", sessionID).Order("created_at desc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&events).Error
	return events, err
}

func (r *aiDevSessionRepo) CreateApproval(approval *model.AIApproval) error {
	return global.DB.Create(approval).Error
}

func (r *aiDevSessionRepo) GetApprovalByID(id uint) (*model.AIApproval, error) {
	var approval model.AIApproval
	err := global.DB.Where("id = ?", id).First(&approval).Error
	return &approval, err
}

func (r *aiDevSessionRepo) GetApprovalsByUserID(userID uint, status string, limit int) ([]*model.AIApproval, error) {
	var approvals []*model.AIApproval
	db := global.DB.Where("request_user_id = ?", userID).Order("updated_at desc")
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&approvals).Error
	return approvals, err
}

func (r *aiDevSessionRepo) GetPendingApprovalBySessionID(sessionID uint) (*model.AIApproval, error) {
	var approval model.AIApproval
	err := global.DB.
		Where("session_id = ? AND status = ?", sessionID, "pending").
		Order("created_at desc").
		First(&approval).Error
	return &approval, err
}

func (r *aiDevSessionRepo) UpdateApproval(approval *model.AIApproval) error {
	return global.DB.Save(approval).Error
}
