package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func enqueueCodeDeliveryJob(session *model.AIDevSession, userID uint, requestIP string) (*model.AICodeDeliveryJob, error) {
	job, err := persistCodeDeliveryJob(session, userID, requestIP)
	if err != nil {
		return nil, err
	}
	enqueueCodeDelivery(job.ID)
	return job, nil
}

func persistCodeDeliveryJob(session *model.AIDevSession, userID uint, requestIP string) (*model.AICodeDeliveryJob, error) {
	codeHostTerminalLifecycle.Lock()
	defer codeHostTerminalLifecycle.Unlock()
	unlockLifecycle := codeSessionLifecycles.lock(session.ID)
	defer unlockLifecycle()
	var job model.AICodeDeliveryJob
	queryErr := global.DB.Where("session_id = ?", session.ID).First(&job).Error
	if queryErr == nil && (job.Status == codeDeliveryJobQueued || job.Status == codeDeliveryJobRunning || job.Status == codeDeliveryJobCompleted) {
		return &job, nil
	}
	if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
		return nil, queryErr
	}
	var current model.AIDevSession
	if err := global.DB.First(&current, session.ID).Error; err != nil {
		return nil, err
	}
	if err := validateCodeSessionReadyForDelivery(global.DB, &current); err != nil {
		return nil, err
	}
	if err := stopHostTerminalsForCodeDelivery(context.Background(), userID); err != nil {
		return nil, err
	}
	keys, targetBranch, err := codeDeliveryRepositoryKeys(session)
	if err != nil {
		return nil, err
	}
	encodedKeys, _ := json.Marshal(keys)
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		var lockedSession model.AIDevSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedSession, session.ID).Error; err != nil {
			return err
		}
		queryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("session_id = ?", session.ID).First(&job).Error
		if queryErr == nil && (job.Status == codeDeliveryJobQueued || job.Status == codeDeliveryJobRunning || job.Status == codeDeliveryJobCompleted) {
			return nil
		}
		if err := validateCodeSessionReadyForDelivery(tx, &lockedSession); err != nil {
			return err
		}
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			job = model.AICodeDeliveryJob{
				SessionID: session.ID, TaskID: session.LastTaskID, ProjectID: session.ProjectID, UserID: userID,
				Status: codeDeliveryJobQueued, Stage: codeDeliveryStageQueued,
				RepositoryKeys: string(encodedKeys), TargetBranch: targetBranch, RequestIP: requestIP,
			}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
			return markCodeSessionDelivering(tx, &lockedSession)
		}
		if queryErr != nil {
			return queryErr
		}
		updates := map[string]any{
			"status": codeDeliveryJobQueued, "stage": codeDeliveryStageQueued, "progress": 0,
			"repository_keys": string(encodedKeys), "target_branch": targetBranch,
			"task_id": session.LastTaskID, "result_commit": "", "result_type": "", "repository_results": "",
			"error_message": "", "conflict_files": "", "request_ip": requestIP,
			"lease_owner": "", "lease_expires_at": nil, "started_at": nil, "completed_at": nil,
		}
		if err := tx.Model(&job).Updates(updates).Error; err != nil {
			return err
		}
		return markCodeSessionDelivering(tx, &lockedSession)
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}
