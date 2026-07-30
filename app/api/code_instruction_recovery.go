package api

import (
	"errors"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

func recoverInterruptedCodeInstructions() error {
	now := time.Now()
	return global.DB.Transaction(func(tx *gorm.DB) error {
		var instructions []model.AIInstruction
		if err := tx.Where("status = ?", "running").Find(&instructions).Error; err != nil {
			return err
		}
		if len(instructions) == 0 {
			return nil
		}
		instructionIDs := make([]uint, 0, len(instructions))
		taskIDs := make([]uint, 0, len(instructions))
		sessionIDs := make([]uint, 0, len(instructions))
		for _, instruction := range instructions {
			instructionIDs = append(instructionIDs, instruction.ID)
			taskIDs = append(taskIDs, instruction.TaskID)
			sessionIDs = append(sessionIDs, instruction.SessionID)
		}
		if err := tx.Model(&model.AIInstruction{}).
			Where("id IN ? AND status = ?", instructionIDs, "running").
			Update("status", "failed").Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AIExecutionRun{}).
			Where("instruction_id IN ? AND status = ?", instructionIDs, "running").
			Updates(map[string]any{
				"status": "failed", "exit_code": -1, "completed_at": now,
				"error_message": "GoPanel 服务重启，运行已中断，请确认后重试",
			}).Error; err != nil {
			return err
		}
		reconciledTasks := make(map[uint]struct{}, len(taskIDs))
		for index, taskID := range taskIDs {
			if _, exists := reconciledTasks[taskID]; exists {
				continue
			}
			reconciledTasks[taskID] = struct{}{}
			var task model.AITask
			var session model.AIDevSession
			if err := tx.First(&task, taskID).Error; err != nil {
				return err
			}
			if err := tx.First(&session, sessionIDs[index]).Error; err != nil {
				return err
			}
			if err := reconcileCodeTaskState(tx, &session, &task, "failed", "failed"); err != nil {
				return err
			}
		}
		for _, instruction := range instructions {
			event := &model.AITimelineEvent{
				SessionID: instruction.SessionID, TaskID: instruction.TaskID, InstructionID: instruction.ID,
				EventType: "execution_interrupted", Stage: "failed", Title: "运行因服务重启而中断",
				Content: "为避免重复执行外部操作，本次运行不会自动重试。", Status: "error",
			}
			if err := tx.Create(event).Error; err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
				return err
			}
		}
		return nil
	})
}
