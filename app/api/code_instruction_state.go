package api

import (
	"github.com/aihop/gopanel/app/model"
	"gorm.io/gorm"
)

func reconcileCodeTaskState(
	tx *gorm.DB,
	session *model.AIDevSession,
	task *model.AITask,
	terminalTaskStatus string,
	terminalSessionStage string,
) error {
	var statuses []string
	if err := tx.Model(&model.AIInstruction{}).
		Where("task_id = ?", task.ID).
		Pluck("status", &statuses).Error; err != nil {
		return err
	}
	taskStatus, sessionStage := terminalTaskStatus, terminalSessionStage
	if containsCodeInstructionStatus(statuses, "running") {
		taskStatus, sessionStage = "running", "executing"
	} else if containsCodeInstructionStatus(statuses, "pending_approval") {
		taskStatus, sessionStage = "pending_approval", "awaiting_approval"
	} else if containsCodeInstructionStatus(statuses, "queued") {
		taskStatus, sessionStage = "queued", "instruction_queued"
	}
	if taskStatus == "" {
		taskStatus = "completed"
	}
	if sessionStage == "" {
		sessionStage = "completed"
	}
	if err := tx.Model(&model.AITask{}).
		Where("id = ?", task.ID).
		Update("status", taskStatus).Error; err != nil {
		return err
	}
	updates := map[string]any{
		"status":        "active",
		"current_stage": sessionStage,
		"last_task_id":  task.ID,
	}
	if err := tx.Model(&model.AIDevSession{}).
		Where("id = ?", session.ID).
		Updates(updates).Error; err != nil {
		return err
	}
	task.Status = taskStatus
	session.Status = "active"
	session.CurrentStage = sessionStage
	session.LastTaskID = task.ID
	return nil
}

func containsCodeInstructionStatus(statuses []string, expected string) bool {
	for _, status := range statuses {
		if status == expected {
			return true
		}
	}
	return false
}
