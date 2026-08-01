package api

import (
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func CheckCodeSessionGitSync(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	result, session, err := checkCodeSessionGitSync(c)
	if err != nil {
		if session != nil {
			recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "session_git_sync_check", "failed", "session", err.Error(), c.IP(), startedAt, nil)
		}
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "session_git_sync_check", "success", "session", result.Status, c.IP(), startedAt, codeAuditMeta{"repositories": len(result.Repositories)})
	return c.JSON(e.Succ(result))
}

func SyncCodeSessionGitRepository(c fiber.Ctx) error {
	var req struct {
		RepositoryID string `json:"repositoryId"`
		Confirm      bool   `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil || !req.Confirm || strings.TrimSpace(req.RepositoryID) == "" {
		return c.JSON(e.Fail(errors.New("同步到会话需要选择仓库并明确确认")))
	}
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	result, session, err := syncCodeSessionGitRepositoryOperation(c, strings.TrimSpace(req.RepositoryID))
	if err != nil {
		if session != nil {
			recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "session_git_sync", "failed", req.RepositoryID, err.Error(), c.IP(), startedAt, nil)
		}
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "session_git_sync", "success", req.RepositoryID, result.Status, c.IP(), startedAt, nil)
	return c.JSON(e.Succ(result))
}
