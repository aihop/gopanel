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
	"github.com/aihop/gopanel/global"
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
	_, limit = normalizeCodePage(1, limit, 50)

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

	var instruction model.AIInstruction
	var session model.AIDevSession
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AIApproval{}).
			Where("id = ? AND status = ?", approval.ID, "pending").
			Updates(map[string]any{
				"status": decision, "decision": decision, "decision_reason": strings.TrimSpace(req.Reason),
				"approve_user_id": claims.UserId, "decision_at": time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("该审批已处理")
		}
		if err := tx.Where("id = ? AND session_id = ?", approval.InstructionID, approval.SessionID).First(&instruction).Error; err != nil {
			return errors.New("对应指令不存在")
		}
		if err := tx.First(&session, approval.SessionID).Error; err != nil {
			return err
		}
		var task model.AITask
		if approval.TaskID > 0 {
			if err := tx.First(&task, approval.TaskID).Error; err != nil {
				return err
			}
		}
		if decision == "approved" {
			instruction.Status = "queued"
		} else {
			instruction.Status = "rejected"
			if task.ID > 0 {
				if err := tx.Create(&model.AIMessage{SessionID: approval.SessionID, TaskID: task.ID, Role: "system", Content: "该开发指令已被人工拒绝执行。"}).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Save(&instruction).Error; err != nil {
			return err
		}
		if task.ID == 0 {
			return nil
		}
		terminalTaskStatus, terminalStage := "cancelled", "approval_rejected"
		if decision == "approved" {
			terminalTaskStatus, terminalStage = "queued", "instruction_queued"
		}
		return reconcileCodeTaskState(tx, &session, &task, terminalTaskStatus, terminalStage)
	}); err != nil {
		return c.JSON(e.Fail(err))
	}
	approval, _ = sessionRepo.GetApprovalByID(approval.ID)
	if decision == "approved" {
		enqueueCodeInstruction(instruction.ID)
	}

	return c.JSON(e.Succ(fiber.Map{
		"approval":    approval,
		"instruction": &instruction,
		"session":     &session,
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
