package api

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

var approvalDangerousPatterns = []string{
	"rm -rf",
	"sudo rm",
	"drop database",
	"drop table",
	"truncate table",
	"git push",
	"git reset --hard",
	"brew install",
	"apt install",
	"yum install",
	"apk add",
	"chmod 777",
	"chown -r",
}

func shouldRequireAIApproval(content string, requireApproval bool) bool {
	if !requireApproval {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(content))
	if normalized == "" {
		return false
	}
	for _, pattern := range approvalDangerousPatterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

func buildApprovalTitle(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "危险操作审批"
	}
	runes := []rune(content)
	if len(runes) > 28 {
		runes = runes[:28]
	}
	return "审批: " + strings.TrimSpace(string(runes))
}

func GetAIApprovals(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	status := strings.TrimSpace(c.Query("status"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	approvals, err := repo.NewAIDevSessionRepo().GetApprovalsByUserID(claims.UserId, status, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"items": approvals,
		"total": len(approvals),
	}))
}

func ApproveAIApproval(c fiber.Ctx) error {
	return decideAIApproval(c, "approved")
}

func RejectAIApproval(c fiber.Ctx) error {
	return decideAIApproval(c, "rejected")
}

func decideAIApproval(c fiber.Ctx, decision string) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	approvalID, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		Reason string `json:"reason"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}

	sessionRepo := repo.NewAIDevSessionRepo()
	approval, err := sessionRepo.GetApprovalByID(uint(approvalID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if approval.RequestUserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权处理该审批")))
	}
	if approval.Status != "pending" {
		return c.JSON(e.Fail(errors.New("该审批已处理")))
	}

	instructions, _, err := sessionRepo.GetInstructionsBySessionID(approval.SessionID, 1, 200)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var instruction *model.AIInstruction
	for _, item := range instructions {
		if item.ID == approval.InstructionID {
			instruction = item
			break
		}
	}
	if instruction == nil {
		return c.JSON(e.Fail(errors.New("对应指令不存在")))
	}

	session, err := sessionRepo.GetSessionByID(approval.SessionID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	now := time.Now()
	approval.Status = decision
	approval.Decision = decision
	approval.DecisionReason = strings.TrimSpace(req.Reason)
	approval.ApproveUserID = claims.UserId
	approval.DecisionAt = &now
	if err := sessionRepo.UpdateApproval(approval); err != nil {
		return c.JSON(e.Fail(err))
	}

	taskRepo := repo.NewAITaskRepo()
	if decision == "approved" {
		instruction.Status = "queued"
		session.CurrentStage = "instruction_queued"
		session.Status = "active"
		if approval.TaskID > 0 {
			if task, taskErr := taskRepo.GetTaskByID(approval.TaskID); taskErr == nil {
				task.Status = "queued"
				_ = taskRepo.UpdateTask(task)
			}
		}
	} else {
		instruction.Status = "rejected"
		session.CurrentStage = "approval_rejected"
		if approval.TaskID > 0 {
			if task, taskErr := taskRepo.GetTaskByID(approval.TaskID); taskErr == nil {
				task.Status = "failed"
				_ = taskRepo.UpdateTask(task)
				_ = taskRepo.CreateMessage(&model.AIMessage{
					TaskID:  task.ID,
					Role:    "system",
					Content: "该开发指令已被人工拒绝执行。",
				})
			}
		}
	}

	if err := sessionRepo.UpdateInstruction(instruction); err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := sessionRepo.UpdateSession(session); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fiber.Map{
		"approval":    approval,
		"instruction": instruction,
		"session":     session,
	}))
}

func getPendingApprovalForSession(sessionRepo repo.IAIDevSessionRepo, sessionID uint) *model.AIApproval {
	approval, err := sessionRepo.GetPendingApprovalBySessionID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return nil
	}
	return approval
}
