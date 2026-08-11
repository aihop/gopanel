package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func GetCodeDeliveryConflicts(c fiber.Ctx) error {
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	_, contexts, err := loadCodeDeliveryConflictContexts(session.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"repositories": codeDeliveryConflictRepositoryViews(contexts)}))
}

func GetCodeDeliveryConflictFile(c fiber.Ctx) error {
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	_, contexts, err := loadCodeDeliveryConflictContexts(session.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	conflictContext, file, err := findCodeDeliveryConflictContext(contexts, c.Query("repositoryId"), c.Query("path"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := readCodeDeliveryConflictFile(conflictContext, file)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func SaveCodeDeliveryConflictFile(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var request struct {
		RepositoryID string `json:"repositoryId"`
		Path         string `json:"path"`
		Resolution   string `json:"resolution"`
		Content      string `json:"content"`
		BaseVersion  string `json:"baseVersion"`
	}
	if err := c.Bind().JSON(&request); err != nil {
		return c.JSON(e.Fail(err))
	}
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var result codeDeliveryConflictFile
	err = runCodeDeliveryConflictMutation(session, func(_ *model.AICodeDeliveryJob, contexts []codeDeliveryConflictContext) error {
		conflictContext, file, err := findCodeDeliveryConflictContext(contexts, request.RepositoryID, request.Path)
		if err != nil {
			return err
		}
		result, err = saveCodeDeliveryConflictFile(
			conflictContext, file, strings.TrimSpace(request.Resolution), request.Content, request.BaseVersion,
		)
		return err
	})
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_conflict_file_save", "failed", request.Path, err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_conflict_file_save", "success", request.Path, request.Resolution, c.IP(), startedAt, codeAuditMeta{"repositoryId": request.RepositoryID})
	return c.JSON(e.Succ(result))
}

func CompleteCodeDeliveryConflicts(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var jobID uint
	err = runCodeDeliveryConflictMutation(session, func(job *model.AICodeDeliveryJob, contexts []codeDeliveryConflictContext) error {
		jobID = job.ID
		return completeCodeDeliveryConflict(job, contexts)
	})
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_conflict_complete", "failed", "delivery", err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_conflict_complete", "success", "delivery", "queued", c.IP(), startedAt, codeAuditMeta{"jobId": jobID})
	view, err := loadCodeDeliveryJobView(session.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(view))
}

func ConfirmManualCodeDeliveryConflict(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var jobID uint
	err = runCodeDeliveryConflictMutation(session, func(job *model.AICodeDeliveryJob, contexts []codeDeliveryConflictContext) error {
		jobID = job.ID
		return confirmManualCodeDeliveryConflict(job, contexts)
	})
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_conflict_manual_confirm", "failed", "delivery", err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "delivery_conflict_manual_confirm", "success", "delivery", "queued", c.IP(), startedAt, codeAuditMeta{"jobId": jobID})
	view, err := loadCodeDeliveryJobView(session.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(view))
}

func runCodeDeliveryConflictMutation(session *model.AIDevSession, operation func(*model.AICodeDeliveryJob, []codeDeliveryConflictContext) error) error {
	if session == nil || session.ID == 0 {
		return errors.New("开发会话不可用")
	}
	unlockLifecycle := codeSessionLifecycles.lock(session.ID)
	defer unlockLifecycle()
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionDelivery, false)
	if err != nil {
		return codeSessionWorkspaceMutationError(err)
	}
	defer lease.Release()
	job, contexts, err := loadCodeDeliveryConflictContexts(session.ID)
	if err != nil {
		return err
	}
	return operation(job, contexts)
}
